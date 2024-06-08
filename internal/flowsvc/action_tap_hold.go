package flowsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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

type ActionTapHold struct {
	onHold HIDUsageAction
	onTap  HIDUsageAction
	delay  time.Duration
}

type tapHoldConfig struct {
	OnHold json.RawMessage `json:"onHold"`
	OnTap  json.RawMessage `json:"onTap"`
	Delay  Duration        `json:"delay"`
}

func NewTapHoldAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	var cfg tapHoldConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	onHold, err := provider.ActionRegistry.NewFromJSON(cfg.OnHold)
	if err != nil {
		return nil, fmt.Errorf("failed to create onHold action: %w", err)
	}

	onTap, err := provider.ActionRegistry.NewFromJSON(cfg.OnTap)
	if err != nil {
		return nil, fmt.Errorf("failed to create onTap action: %w", err)
	}

	return &ActionTapHold{
		onHold: onHold,
		onTap:  newTapAction(onTap, 10*time.Millisecond),
		delay:  time.Duration(cfg.Delay),
	}, nil
}

func (a *ActionTapHold) Usages() []hidparse.Usage {
	usages := a.onHold.Usages()
	usages = append(usages, a.onTap.Usages()...)
	return usages
}

func (a *ActionTapHold) Activate(ctx context.Context, activator UsageActivator) func() {
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
