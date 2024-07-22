package actions

import (
	"context"
	"time"

	"github.com/neuroplastio/neuroplastio/flowapi"
	"github.com/neuroplastio/neuroplastio/hidapi"
)

type Mod struct{}

func (a Mod) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Modifier",
		Description: "Perform an action with active modifier key (or any other HID usage)",
		Signature:   "mod(modifier: Usage, action: Action, delay: Duration = 1ms)",
	}
}

func (a Mod) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	modifier, err := p.Args().Usages("modifier")
	if err != nil {
		return nil, err
	}
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}
	return NewModHandler(p.Context(), modifier, action, p.Args().Duration("delay")), nil
}

type Sleeper struct {
	duration time.Duration
	jobCh    chan job
	cancelCh chan struct{}
}

func NewSleeper(ctx context.Context, duration time.Duration) *Sleeper {
	s := &Sleeper{
		duration: duration,
		jobCh:    make(chan job, 1),
		cancelCh: make(chan struct{}),
	}
	s.start(ctx)
	return s
}

func (s *Sleeper) start(ctx context.Context) {
	go func() {
		timer := time.NewTimer(0)
		defer func() {
			if timer != nil {
				if !timer.Stop() {
					<-timer.C
				}
			}
			close(s.jobCh)
			close(s.cancelCh)
		}()
		var jj *job
		for {
			select {
			case j := <-s.jobCh:
				timer.Reset(s.duration)
				jj = &j
			case <-s.cancelCh:
				if jj == nil {
					continue
				}
				if jj.onCancel != nil {
					jj.onCancel()
				}
				jj = nil
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				if jj == nil {
					continue
				}
				jj.fn()
				jj = nil
			case <-ctx.Done():
				return
			}
		}
	}()
}

type job struct {
	fn       func()
	onCancel func()
}

func (s *Sleeper) do(fn func(), onCancel func()) {
	s.jobCh <- job{fn: fn, onCancel: onCancel}
}

func (s *Sleeper) cancel() {
	s.cancelCh <- struct{}{}
}

func NewModHandler(ctx context.Context, modifier []hidapi.Usage, action flowapi.ActionHandler, duration time.Duration) flowapi.ActionHandler {
	sleeper := NewSleeper(ctx, duration)
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		ac.HIDEvent(func(e *hidapi.Event) {
			e.Activate(modifier...)
		})
		var fin flowapi.ActionFinalizer
		sleeper.do(func() {
			fin = action(ac)
		}, nil)
		return func(ac flowapi.ActionContext) {
			sleeper.cancel()
			if fin != nil {
				fin(ac)
			}
			ac.HIDEvent(func(e *hidapi.Event) {
				e.Deactivate(modifier...)
			})
		}
	}
}
