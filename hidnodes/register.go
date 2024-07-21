package hidnodes

import "github.com/neuroplastio/neuroplastio/internal/flowsvc"

func Register(reg *flowsvc.Registry) {
	reg.MustRegisterNode("input", Input{})
	reg.MustRegisterNode("output", Output{})
	reg.MustRegisterNode("bind", Bind{})
	reg.MustRegisterNode("mux", Mux{})
}
