package flowsvc

import (
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
)

func NewActionUsageHandler(usages []hidparse.Usage) ActionHandler {
	return func(ac ActionContext) ActionFinalizer {
		ac.HIDEvent(func(e *hidevent.HIDEvent) {
			e.Activate(usages...)
		})
		return func(ac ActionContext) {
			ac.HIDEvent(func(e *hidevent.HIDEvent) {
				e.Deactivate(usages...)
			})
		}
	}
}
