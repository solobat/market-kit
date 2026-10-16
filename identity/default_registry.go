package identity

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed default_registry.json
var defaultRegistryJSON []byte

//go:embed generated_registry.json
var generatedRegistryJSON []byte

//go:embed reviewed_registry.json
var reviewedRegistryJSON []byte

var reviewedOnce sync.Once
var reviewedTemplate Registry
var reviewedError error

func LoadReviewedRegistry() (Registry, error) {
	reviewedOnce.Do(func() {
		reviewedError = json.Unmarshal(reviewedRegistryJSON, &reviewedTemplate)
		if reviewedError == nil {
			reviewedTemplate.Normalize()
		}
	})
	if reviewedError != nil {
		return Registry{}, reviewedError
	}
	return reviewedTemplate.Clone(), nil
}

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

	reviewed, err := LoadReviewedRegistry()
	if err != nil {
		return Registry{}, err
	}
	return registry.Overlay(reviewed), nil
}
