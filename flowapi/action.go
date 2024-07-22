package flowapi

import (
	"context"
	"sync"
	"time"

	"github.com/neuroplastio/neuroplastio/hidapi"
	"go.uber.org/zap"
)

type ActionDescriptor struct {
	DisplayName string
	Description string

	Signature string
}

type SignalDescriptor struct {
	DisplayName string
	Description string

	Signature string
}

type Action interface {
	Descriptor() ActionDescriptor
	CreateHandler(provider ActionProvider) (ActionHandler, error)
}

type ActionContext interface {
	Context() context.Context

	// HIDEvent returns an event that is being processed at the moment.
	// It can only be called synchronously. Asynchronous calls have undefined behavior.
	HIDEvent() *hidapi.Event

	// Async branches out action into an asynchronous function.
	// You should return finalizer function that will be called when the action is finished.
	// When async action is finished, asyncCtx.Done() channel will be closed.
	Async(fn func(asyncCtx AsyncActionContext)) ActionFinalizer
}

type AsyncActionContext interface {
	// Done returns a channel that will be closed when the action is finished.
	Done() <-chan struct{}

	Capture(fn func(ac ActionContext) bool) func()

	// After returns a channel that will be closed after the specified duration.
	After(duration time.Duration) <-chan time.Time

	Do(fn func(ac ActionContext))

	Action(action ActionHandler) ActionFinalizer
	Finish(finalizer ActionFinalizer)
	OnFinish(finalizer ActionFinalizer)
}

type ActionFinalizer func(ac ActionContext)
type ActionHandler func(ac ActionContext) ActionFinalizer
type SignalHandler func(ctx context.Context)

type ActionProvider interface {
	Context() context.Context
	Args() Arguments
	ActionArg(argName string) (ActionHandler, error)
	SignalArg(argName string) (SignalHandler, error)
}

type ActionCreator func(p ActionProvider) (ActionHandler, error)
type SignalCreator func(p ActionProvider) (SignalHandler, error)

func NewActionUsageHandler(usages ...hidapi.Usage) ActionHandler {
	return func(ac ActionContext) ActionFinalizer {
		ac.HIDEvent().Activate(usages...)
		return func(ac ActionContext) {
			ac.HIDEvent().Deactivate(usages...)
		}
	}
}

type actionContext struct {
	event *hidapi.Event
	pool  *ActionContextPool
}

func (a *actionContext) Context() context.Context {
	return a.pool.ctx
}

func (a *actionContext) HIDEvent() *hidapi.Event {
	return a.event
}

func (a *actionContext) Async(fn func(asyncCtx AsyncActionContext)) ActionFinalizer {
	return a.pool.runAsync(a.clone(), fn)
}

func (a *actionContext) clone() *actionContext {
	return &actionContext{
		event: a.event.Clone(),
		pool:  a.pool,
	}
}

// TODO: clean up and refactor this mess, fix data races

type asyncActionContext struct {
	ac             *actionContext
	capture        func(ac ActionContext) bool
	done           chan struct{}
	onFinish       []ActionFinalizer
	capturedUsages map[hidapi.Usage]struct{}
	captured       chan ActionContext
}

func NewActionContextPool(ctx context.Context, log *zap.Logger, hidChan chan<- *hidapi.Event) *ActionContextPool {
	pool := &ActionContextPool{
		ctx:       ctx,
		log:       log,
		capturers: make(map[*asyncActionContext]func(ac ActionContext) bool),
		hidChan:   hidChan,
		flush:     make(chan ActionContext, 64),
	}
	return pool
}

type ActionContextPool struct {
	log     *zap.Logger
	ctx     context.Context
	hidChan chan<- *hidapi.Event
	flush   chan ActionContext

	mu        sync.Mutex
	capturers map[*asyncActionContext]func(ac ActionContext) bool
}

func (a *ActionContextPool) New(event *hidapi.Event) ActionContext {
	ac := &actionContext{
		event: event,
		pool:  a,
	}
	return ac
}

func (a *ActionContextPool) TryCapture(ac ActionContext) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for async, capturer := range a.capturers {
		for _, usage := range ac.HIDEvent().Usages() {
			if _, ok := async.capturedUsages[usage.Usage]; ok {
				a.log.Debug("captured usages", zap.Any("usages", ac.HIDEvent().Usages()))
				async.captured <- ac
				return true
			}
		}
		if capturer != nil && capturer(ac) {
			a.log.Debug("captured", zap.Any("usages", ac.HIDEvent().Usages()))
			for _, usage := range ac.HIDEvent().Usages() {
				async.capturedUsages[usage.Usage] = struct{}{}
			}
			async.captured <- ac
			return true
		}
	}
	return false
}

func (a *ActionContextPool) Flush() <-chan ActionContext {
	return a.flush
}

func (a *ActionContextPool) runAsync(ac *actionContext, fn func(ac AsyncActionContext)) ActionFinalizer {
	asyncCtx := &asyncActionContext{
		ac:             ac,
		done:           make(chan struct{}),
		capturedUsages: make(map[hidapi.Usage]struct{}),
		captured:       make(chan ActionContext, 64),
	}
	go func() {
		for captured := range asyncCtx.captured {
			time.Sleep(1 * time.Millisecond)
			a.flush <- captured
		}
	}()
	go fn(asyncCtx)

	return func(ac ActionContext) {
		a.mu.Lock()
		delete(a.capturers, asyncCtx)
		a.mu.Unlock()
		for _, onFinish := range asyncCtx.onFinish {
			onFinish(ac)
		}
		a.log.Debug("finalizer", zap.Any("usages", ac.HIDEvent().Usages()))
		close(asyncCtx.done)
		close(asyncCtx.captured)
	}
}

func (a *asyncActionContext) After(duration time.Duration) <-chan time.Time {
	// TODO: pooling
	return time.After(duration)
}

func (a *asyncActionContext) Done() <-chan struct{} {
	return a.done
}

func (a *asyncActionContext) Capture(fn func(ac ActionContext) bool) func() {
	a.ac.pool.mu.Lock()
	a.ac.pool.capturers[a] = fn
	a.ac.pool.mu.Unlock()
	return func() {
		a.ac.pool.mu.Lock()
		a.ac.pool.capturers[a] = nil
		a.ac.pool.mu.Unlock()
	}
}

func (a *asyncActionContext) NewActionContext() ActionContext {
	return &actionContext{
		pool:  a.ac.pool,
		event: hidapi.NewEvent(),
	}
}

func (a *asyncActionContext) Action(action ActionHandler) ActionFinalizer {
	var fin ActionFinalizer
	a.Do(func(ac ActionContext) {
		fin = action(ac)
	})
	return fin
}

func (a *asyncActionContext) Finish(fin ActionFinalizer) {
	if fin == nil {
		return
	}
	a.Do(func(ac ActionContext) {
		fin(ac)
	})
}

func (a *asyncActionContext) Do(fn func(ac ActionContext)) {
	ac := a.NewActionContext()
	fn(ac)
	event := ac.HIDEvent()
	if !event.IsEmpty() {
		a.ac.pool.hidChan <- event
	}
}

func (a *asyncActionContext) OnFinish(fin ActionFinalizer) {
	if fin == nil {
		return
	}
	a.onFinish = append(a.onFinish, fin)
}
