package flowsvc

import (
	"context"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

func newActionUsageHandler(usages []hidparse.Usage) *actionUsageHandler {
	return &actionUsageHandler{
		usages: usages,
	}
}

type actionUsageHandler struct {
	usages []hidparse.Usage
}

func (a *actionUsageHandler) Usages() []hidparse.Usage {
	return a.usages
}

func (a *actionUsageHandler) Activate(ctx context.Context, activator UsageActivator) func() {
	return activator(a.Usages())
}
