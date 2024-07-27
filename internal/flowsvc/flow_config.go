package flowsvc

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/goccy/go-yaml"
)

type FlowConfig struct {
	// Nodes is a list of node configurations.
	Nodes []NodeConfig `yaml:"nodes"`
}

func (f FlowConfig) treeHash() uint64 {
	var tokens []string
	for _, node := range f.Nodes {
		tokens = append(tokens, node.ID)
		tokens = append(tokens, node.Type)
		tokens = append(tokens, node.To...)
	}
	return xxhash.Sum64String(strings.Join(tokens, "|"))
}

type NodeConfig struct {
	ID     string          `yaml:"id"`
	Type   string          `yaml:"type"`
	To     []string        `yaml:"to"`
	Config json.RawMessage `yaml:"config"`
}

func (n *NodeConfig) UnmarshalYAML(data []byte) error {
	idStruct := struct {
		ID string   `yaml:"id"`
		To []string `yaml:"to"`
	}{}
	if err := yaml.Unmarshal(data, &idStruct); err != nil {
		return fmt.Errorf("error unmarshalling idStruct: %w", err)
	}
	mm := make(map[string]any)
	if err := yaml.UnmarshalWithOptions(data, &mm, yaml.UseOrderedMap()); err != nil {
		return fmt.Errorf("error unmarshalling node map: %w", err)
	}
	delete(mm, "id")
	delete(mm, "to")
	for key, val := range mm {
		n.ID = idStruct.ID
		n.To = idStruct.To
		n.Type = key
		cfg, err := yaml.Marshal(val)
		if err != nil {
			return fmt.Errorf("error marshalling config: %w", err)
		}
		n.Config = cfg

		return nil
	}
	return fmt.Errorf("missing type in node config")
}

func (n *NodeConfig) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(map[string]any{
		"id":   n.ID,
		"to":   n.To,
		n.Type: n.Config,
	})
}
