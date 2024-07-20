package flowsvc

import (
	"encoding/json"
	"fmt"
)

type FlowConfig struct {
	// Nodes is a list of node configurations.
	Nodes []NodeConfig `json:"nodes"`
}

type NodeConfig struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	To     []string        `json:"to"`
	Config json.RawMessage `json:"config"`
}

func (n *NodeConfig) UnmarshalJSON(data []byte) error {
	idStruct := struct {
		ID string `json:"id"`
		To []string
	}{}
	if err := json.Unmarshal(data, &idStruct); err != nil {
		return err
	}
	mm := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &mm); err != nil {
		return err
	}
	delete(mm, "id")
	delete(mm, "to")
	for key, val := range mm {
		n.ID = idStruct.ID
		n.To = idStruct.To
		n.Type = key
		n.Config = val

		return nil
	}
	return fmt.Errorf("missing type in node config")
}

func (n *NodeConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":   n.ID,
		n.Type: n.Config,
	})
}

type HIDUsageActionConfig struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

func (n *HIDUsageActionConfig) UnmarshalJSON(data []byte) error {
	mm := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &mm); err != nil {
		return err
	}
	delete(mm, "type")
	for key, val := range mm {
		n.Type = key
		n.Config = val
		return nil
	}
	return fmt.Errorf("missing type in action config")
}

func (n *HIDUsageActionConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		n.Type: n.Config,
	})
}
