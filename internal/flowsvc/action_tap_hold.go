package flowsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type ActionTapHold struct{}

func (a ActionTapHold) Metadata() HIDUsageActionMetadata {
	return HIDUsageActionMetadata{
		DisplayName: "Tap Hold",
		Description: "Tap and hold action",
		Declaration: "tapHold(onTap: Action, onHold: Action, delay: Duration = 250ms, tapDuration: Duration = 10ms)",
	}
}

type actionTapHoldHandler struct {
	onHold HIDUsageActionHandler
	onTap  HIDUsageActionHandler
	delay  time.Duration
}

func (a ActionTapHold) Handler(args actiondsl.Arguments, provider *HIDActionProvider) (HIDUsageActionHandler, error) {
	onHold, err := provider.ActionRegistry.New(args.Action("onHold"))
	if err != nil {
		return nil, fmt.Errorf("failed to create onHold action: %w", err)
	}

	onTap, err := provider.ActionRegistry.New(args.Action("onTap"))
	if err != nil {
		return nil, fmt.Errorf("failed to create onTap action: %w", err)
	}

	return &actionTapHoldHandler{
		onHold: onHold,
		onTap:  newTapActionHandler(onTap, args.Duration("tapDuration")),
		delay:  args.Duration("delay"),
	}, nil
}

func (a *actionTapHoldHandler) Usages() []hidparse.Usage {
	usages := a.onHold.Usages()
	usages = append(usages, a.onTap.Usages()...)
	return usages
}

func (a *actionTapHoldHandler) Activate(ctx context.Context, activator UsageActivator) func() {
	deactivateCh := make(chan struct{})
	timer := time.NewTimer(a.delay)
	go func() {
		defer func() {
			if !timer.Stop() {
				<-timer.C
			}
		}()
		var deactivateHold func()
		for {
			select {
			case <-timer.C:
				// start holding
				deactivateHold = a.onHold.Activate(ctx, activator)
			case <-deactivateCh:
				if deactivateHold == nil {
					// tap
					a.onTap.Activate(ctx, activator)()
				} else {
					// finish holding
					deactivateHold()
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() {
		close(deactivateCh)
	}
}
