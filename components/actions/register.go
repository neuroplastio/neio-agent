package actions

import "github.com/neuroplastio/neuroplastio/flowapi"

func Register(reg flowapi.Registry) {
	reg.MustRegisterAction(None{})
	reg.MustRegisterAction(Mod{})
	reg.MustRegisterAction(Char{})
	reg.MustRegisterAction(SendString{})
	reg.MustRegisterAction(Tap{})
	reg.MustRegisterAction(TapHold{})
	reg.MustRegisterAction(Lock{})
	reg.MustRegisterAction(Signal{})
}
