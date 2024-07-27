package flowapi

import (
	"context"
	"sync"
	"time"

	"github.com/neuroplastio/neio-agent/hidapi"
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

type AsyncActionContext interface {
	Interrupt() <-chan struct{}
	After(duration time.Duration) <-chan struct{}
	Finished() <-chan struct{}
	Do(fn func(ac ActionContext))
	Action(action ActionHandler) ActionFinalizer
	Finish(finalizer ActionFinalizer)
	OnFinish(finalizer ActionFinalizer)
}

type asyncActionContext struct {
	ac             *actionContext
	interrupt      chan struct{}
	finished       chan struct{}
	done           chan struct{}
	onFinish       []ActionFinalizer
	capturedUsages map[hidapi.Usage]struct{}
	captured       chan ActionContext
}

func NewActionContextPool(ctx context.Context, log *zap.Logger, hidChan chan<- *hidapi.Event) *ActionContextPool {
	pool := &ActionContextPool{
		ctx:            ctx,
		log:            log,
		hidChan:        hidChan,
		activeContexts: make(map[*asyncActionContext]struct{}),
	}
	return pool
}

type ActionContextPool struct {
	log     *zap.Logger
	ctx     context.Context
	hidChan chan<- *hidapi.Event

	mu             sync.Mutex
	activeContexts map[*asyncActionContext]struct{}
}

func (a *ActionContextPool) New(event *hidapi.Event) ActionContext {
	ac := &actionContext{
		event: event,
		pool:  a,
	}
	return ac
}

func (a *ActionContextPool) Interrupt() {
	a.mu.Lock()
	for ac := range a.activeContexts {
		if ac.interrupt != nil {
			close(ac.interrupt)
		}
	}
	for ac := range a.activeContexts {
		if ac.interrupt != nil {
			select {
			case <-ac.done:
			case <-a.ctx.Done():
			}
		}
		ac.interrupt = nil
	}
	a.mu.Unlock()
}

func (a *ActionContextPool) runAsync(ac *actionContext, fn func(ac AsyncActionContext)) ActionFinalizer {
	asyncCtx := &asyncActionContext{
		ac:       ac,
		finished: make(chan struct{}),
		done:     make(chan struct{}),
	}
	a.mu.Lock()
	a.activeContexts[asyncCtx] = struct{}{}
	a.mu.Unlock()
	go func() {
		defer close(asyncCtx.done)
		fn(asyncCtx)
	}()
	return func(ac ActionContext) {
		a.mu.Lock()
		delete(a.activeContexts, asyncCtx)
		a.mu.Unlock()
		for _, onFinish := range asyncCtx.onFinish {
			onFinish(ac)
		}
		close(asyncCtx.finished)
	}
}

func (a *asyncActionContext) After(duration time.Duration) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-time.After(duration)
		close(ch)
	}()
	return ch
}

func (a *asyncActionContext) Finished() <-chan struct{} {
	return a.finished
}

func (a *asyncActionContext) Interrupt() <-chan struct{} {
	if a.interrupt == nil {
		a.interrupt = make(chan struct{})
	}
	return a.interrupt
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
