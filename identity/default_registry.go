package identity

import (
	_ "embed"
	"encoding/json"
)

//go:embed default_registry.json
var defaultRegistryJSON []byte

//go:embed generated_registry.json
var generatedRegistryJSON []byte

func LoadDefaultRegistry() (Registry, error) {
	var registry Registry
	if err := json.Unmarshal(defaultRegistryJSON, &registry); err != nil {
		return Registry{}, err
	}
	registry.Normalize()

	var generated Registry
	if len(generatedRegistryJSON) > 0 {
		if err := json.Unmarshal(generatedRegistryJSON, &generated); err != nil {
			return Registry{}, err
		}
		generated.Normalize()
		registry = registry.Merge(generated)
	}

	return registry, nil
}
