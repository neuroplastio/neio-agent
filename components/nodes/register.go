package nodes

import (
	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"go.uber.org/zap"
)

func Register(log *zap.Logger, reg *flowsvc.Registry) {
	reg.MustRegisterNodeType("bind", BindType{
		log: log,
	})
	reg.MustRegisterNodeType("mux", MuxType{
		log: log,
	})
}
