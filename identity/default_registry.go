package identity

import (
	_ "embed"
	"encoding/json"
)

//go:embed default_registry.json
var defaultRegistryJSON []byte

func LoadDefaultRegistry() (Registry, error) {
	var registry Registry
	if err := json.Unmarshal(defaultRegistryJSON, &registry); err != nil {
		return Registry{}, err
	}
	registry.Normalize()
	return registry, nil
}
