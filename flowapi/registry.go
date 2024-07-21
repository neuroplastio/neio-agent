package flowapi

type Registry interface {
	MustRegisterNodeType(typ string, node NodeType)
	MustRegisterAction(action Action)
}
