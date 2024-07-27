package nodes

import (
	"github.com/neuroplastio/neio-agent/internal/flowsvc"
	"go.uber.org/zap"
)

func Register(log *zap.Logger, reg *flowsvc.Registry) {
	reg.MustRegisterNodeType("bind", BindType{
		log: log.Named("bind"),
	})
	reg.MustRegisterNodeType("mux", MuxType{
		log: log.Named("mux"),
	})
}
