package identity

import (
	"encoding/json"
	"os"
	"strings"
)

func LoadRegistryFile(path string) (Registry, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Registry{}, err
	}

	var registry Registry
	if err := json.Unmarshal(payload, &registry); err != nil {
		return Registry{}, err
	}
	registry.Normalize()
	return registry, nil
}

func (r *Registry) Normalize() {
	if r.ExchangeAliases == nil {
		r.ExchangeAliases = map[string]string{}
	}
	nextAliases := make(map[string]string, len(r.ExchangeAliases))
	for source, target := range r.ExchangeAliases {
		source = strings.ToLower(strings.TrimSpace(source))
		target = strings.ToLower(strings.TrimSpace(target))
		if source == "" || target == "" {
			continue
		}
		nextAliases[source] = target
	}
	r.ExchangeAliases = nextAliases

	for i := range r.AssetAliases {
		r.AssetAliases[i].Canonical = strings.ToUpper(strings.TrimSpace(r.AssetAliases[i].Canonical))
		r.AssetAliases[i].AssetClass = strings.TrimSpace(r.AssetAliases[i].AssetClass)
		for j := range r.AssetAliases[i].Aliases {
			r.AssetAliases[i].Aliases[j] = strings.ToUpper(strings.TrimSpace(r.AssetAliases[i].Aliases[j]))
		}
	}

	for i := range r.MarketOverrides {
		r.MarketOverrides[i].Exchange = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].Exchange))
		r.MarketOverrides[i].RawSymbol = strings.TrimSpace(r.MarketOverrides[i].RawSymbol)
		r.MarketOverrides[i].MarketType = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].MarketType))
		r.MarketOverrides[i].CanonicalSymbol = strings.ToUpper(strings.TrimSpace(r.MarketOverrides[i].CanonicalSymbol))
	}
}
