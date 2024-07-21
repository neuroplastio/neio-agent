package agent

// Config is loaded from /etc/nplast/agent.yml, but it points to the location of the user-driven configuration files.
// Live reload only applies to user-driven configuration files.
// User configuration is stored at devices.yml and flow.yml.
type Config struct {
	DataDir      string `json:"dataDir"`
	FlowConfig   string `json:"flowConfig"`
	DeviceConfig string `json:"deviceConfig"`
	UhidConfig   string `json:"uhidConfig"`
}
