package flowsvc

import (
	"context"
	"encoding/json"
	"time"
)

type ActionTapHold struct {
	onHold HIDUsageAction
	onTap  HIDUsageAction
	delay  time.Duration
}

type tapHoldConfig struct {
	OnHold HIDUsageActionConfig `json:"onHold"`
	OnTap  HIDUsageActionConfig `json:"onTap"`
	Delay  time.Duration        `json:"delay"`
}

func NewTapHoldAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	var cfg tapHoldConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	onHold, err := provider.ActionRegistry.New(cfg.OnHold.Type, cfg.OnHold.Config)
	if err != nil {
		return nil, err
	}

	onTap, err := provider.ActionRegistry.New(cfg.OnTap.Type, cfg.OnTap.Config)
	if err != nil {
		return nil, err
	}

	return &ActionTapHold{
		onHold: onHold,
		onTap:  newTapAction(onTap, 10*time.Millisecond),
		delay:  cfg.Delay,
	}, nil
}

func (a *ActionTapHold) Usages() []Usage {
	usages := a.onHold.Usages()
	usages = append(usages, a.onTap.Usages()...)
	return usages
}

func (a *ActionTapHold) Activate(ctx context.Context, activator UsageActivator) func() {
	deactivateCh := make(chan struct{})
	go func() {
		timer := time.NewTimer(a.delay)
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
