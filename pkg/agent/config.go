package agent

// Config is loaded from /etc/nplast/agent.yml, but it points to the location of the user-driven configuration files.
// Live reload only applies to user-driven configuration files.
// User configuration is stored at devices.yml and flow.yml.
type Config struct {
	DataDir               string `json:"dataDir"`
	FlowConfig            string `json:"flowConfig"`
	DeviceConfig          string `json:"deviceConfig"`
	UhidConfig            string `json:"uhidConfig"`
}

type FlowConfig struct {
	// Nodes is a list of node configurations.
	Nodes []Node `json:"nodes"`
}

// InputConfig is the configuration for an input node.
// TODO: support virtual / test devices
type Input struct {
	Hid *HidDevice `json:"hid,omitempty"`
}

type HidDevice struct {
	DeviceID string `json:"deviceId"`
}

type Mutator struct {
	Merge *MergeMutator `json:"merge,omitempty"`
}

type MergeMutator struct {
	// TODO: configure merge strategy of identical usage IDs
}

type Output struct {
	// TODO: output vendor ID / product ID configuration
	Gadget *UsbGadget `json:"gadget,omitempty"`
}

type UsbGadget struct {
	Path string `json:"path"`
}

type Node struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`

	Input   *Input   `json:"input,omitempty"`
	Mutator *Mutator `json:"mutator,omitempty"`
	Output  *Output  `json:"output,omitempty"`

	// Links are the connections between nodes
	Links []string `json:"links"`
}
