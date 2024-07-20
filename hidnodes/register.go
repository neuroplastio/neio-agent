package hidnodes

import "github.com/neuroplastio/neuroplastio/internal/flowsvc"

func Register(reg *flowsvc.NodeRegistry) {
	reg.Register("input", Input{})
	reg.Register("output", Output{})
	reg.Register("bind", Bind{})
	reg.Register("mux", Mux{})
}
