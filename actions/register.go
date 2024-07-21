package actions

import "github.com/neuroplastio/neuroplastio/internal/flowsvc"

func Register(reg *flowsvc.Registry) {
	reg.MustRegisterAction(ActionNone{})
	reg.MustRegisterAction(ActionTap{})
	reg.MustRegisterAction(ActionTapHold{})
	reg.MustRegisterAction(ActionLock{})
	reg.MustRegisterAction(ActionSignal{})
}
