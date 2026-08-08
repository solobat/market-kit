package identity

import (
	"encoding/json"
	"os"
	"sort"
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
		r.AssetAliases[i].AssetClass = strings.ToLower(strings.TrimSpace(r.AssetAliases[i].AssetClass))
		r.AssetAliases[i].AssetID = strings.ToLower(strings.TrimSpace(r.AssetAliases[i].AssetID))
		r.AssetAliases[i].UnderlyingID = strings.ToLower(strings.TrimSpace(r.AssetAliases[i].UnderlyingID))
		for j := range r.AssetAliases[i].Aliases {
			r.AssetAliases[i].Aliases[j] = strings.ToUpper(strings.TrimSpace(r.AssetAliases[i].Aliases[j]))
		}
		for j := range r.AssetAliases[i].UnitAliases {
			r.AssetAliases[i].UnitAliases[j].Alias = strings.ToUpper(strings.TrimSpace(r.AssetAliases[i].UnitAliases[j].Alias))
		}
	}

	for i := range r.MarketOverrides {
		r.MarketOverrides[i].Exchange = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].Exchange))
		r.MarketOverrides[i].RawSymbol = strings.TrimSpace(r.MarketOverrides[i].RawSymbol)
		r.MarketOverrides[i].MarketType = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].MarketType))
		r.MarketOverrides[i].CanonicalSymbol = strings.ToUpper(strings.TrimSpace(r.MarketOverrides[i].CanonicalSymbol))
		r.MarketOverrides[i].AssetClass = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].AssetClass))
		r.MarketOverrides[i].AssetID = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].AssetID))
		r.MarketOverrides[i].UnderlyingID = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].UnderlyingID))
		r.MarketOverrides[i].ComparisonKey = normalizeComparisonKey(r.MarketOverrides[i].ComparisonKey)
		r.MarketOverrides[i].InstrumentKind = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].InstrumentKind))
		r.MarketOverrides[i].ContractType = strings.ToLower(strings.TrimSpace(r.MarketOverrides[i].ContractType))
		r.MarketOverrides[i].SettlementAsset = strings.ToUpper(strings.TrimSpace(r.MarketOverrides[i].SettlementAsset))
		if r.MarketOverrides[i].ExpiryAtMs < 0 {
			r.MarketOverrides[i].ExpiryAtMs = 0
		}
		if r.MarketOverrides[i].ComparisonStatus != "" {
			r.MarketOverrides[i].ComparisonStatus = NormalizeComparisonStatus(r.MarketOverrides[i].ComparisonStatus)
		}
		r.MarketOverrides[i].UnitAlias = strings.ToUpper(strings.TrimSpace(r.MarketOverrides[i].UnitAlias))
		if r.MarketOverrides[i].UnitMultiplier <= 0 {
			r.MarketOverrides[i].UnitMultiplier = 0
		}
	}

	r.normalizeAssetCollisions()

	scaledAliases := r.normalizeAssetAliases()
	r.normalizeMarketOverrides(scaledAliases)
}

func (r *Registry) normalizeAssetCollisions() {
	merged := map[string]AssetCollision{}
	for _, item := range r.AssetCollisions {
		item.Symbol = strings.ToUpper(strings.TrimSpace(item.Symbol))
		item.Reason = strings.TrimSpace(item.Reason)
		if item.Symbol == "" {
			continue
		}
		classes := make([]string, 0, len(item.AssetClasses))
		seen := map[string]bool{}
		for _, value := range item.AssetClasses {
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			classes = append(classes, value)
		}
		sort.Strings(classes)
		item.AssetClasses = classes
		if existing, ok := merged[item.Symbol]; ok {
			item = mergeAssetCollision(existing, item)
			sort.Strings(item.AssetClasses)
		}
		merged[item.Symbol] = item
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	r.AssetCollisions = make([]AssetCollision, 0, len(keys))
	for _, key := range keys {
		r.AssetCollisions = append(r.AssetCollisions, merged[key])
	}
}

func normalizeComparisonKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return strings.ToLower(value)
	}
	return strings.ToLower(strings.TrimSpace(parts[0])) + "/" + strings.ToUpper(strings.TrimSpace(parts[1]))
}

func (r *Registry) normalizeAssetAliases() map[string]string {
	merged := map[string]AssetAliasRule{}
	order := make([]string, 0, len(r.AssetAliases))
	for _, item := range r.AssetAliases {
		if item.Canonical == "" {
			continue
		}
		if existing, ok := merged[item.Canonical]; ok {
			merged[item.Canonical] = mergeAssetAliasRule(existing, item)
			continue
		}
		merged[item.Canonical] = item
		order = append(order, item.Canonical)
	}

	canonicalSet := map[string]bool{}
	for canonical := range merged {
		canonicalSet[canonical] = true
	}

	scaledAliases := map[string]string{}
	for _, canonical := range append([]string(nil), order...) {
		item, ok := merged[canonical]
		if !ok {
			continue
		}
		base, multiplier, scaled := inferScaledUnitAlias(item.Canonical, canonicalSet)
		if !scaled {
			continue
		}
		baseRule, exists := merged[base]
		if !exists {
			continue
		}

		unitRule := AssetAliasRule{
			Canonical:  base,
			AssetClass: item.AssetClass,
			UnitAliases: []AssetUnitAlias{
				{Alias: item.Canonical, Multiplier: multiplier},
			},
		}
		for _, alias := range item.Aliases {
			unitRule.UnitAliases = append(unitRule.UnitAliases, AssetUnitAlias{
				Alias:      alias,
				Multiplier: multiplier,
			})
		}
		for _, alias := range item.UnitAliases {
			if alias.Multiplier == 0 {
				alias.Multiplier = multiplier
			}
			unitRule.UnitAliases = append(unitRule.UnitAliases, alias)
		}

		merged[base] = mergeAssetAliasRule(baseRule, unitRule)
		delete(merged, item.Canonical)
		delete(canonicalSet, item.Canonical)
		scaledAliases[item.Canonical] = base
	}

	keys := make([]string, 0, len(merged))
	for canonical := range merged {
		keys = append(keys, canonical)
	}
	sort.Strings(keys)

	normalized := make([]AssetAliasRule, 0, len(keys))
	for _, canonical := range keys {
		item := merged[canonical]
		item.Aliases = normalizeAliasList(item.Canonical, item.Aliases)
		item.UnitAliases = normalizeUnitAliasList(item.Canonical, item.UnitAliases)
		normalized = append(normalized, item)
	}
	r.AssetAliases = normalized
	return scaledAliases
}

func (r *Registry) normalizeMarketOverrides(scaledAliases map[string]string) {
	overrideKeys := map[string]bool{}
	normalized := make([]MarketOverride, 0, len(r.MarketOverrides))
	for _, item := range r.MarketOverrides {
		if base, quote := splitCanonicalSymbol(item.CanonicalSymbol); base != "" && quote != "" {
			if canonicalBase, ok := scaledAliases[base]; ok {
				item.CanonicalSymbol = canonicalBase + "/" + quote
			}
		}
		key := marketOverrideKey(item)
		if key == "||" || overrideKeys[key] {
			continue
		}
		overrideKeys[key] = true
		normalized = append(normalized, item)
	}
	r.MarketOverrides = normalized
}

func normalizeAliasList(canonical string, aliases []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		alias = strings.ToUpper(strings.TrimSpace(alias))
		if alias == "" || alias == canonical || seen[alias] {
			continue
		}
		seen[alias] = true
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

func normalizeUnitAliasList(canonical string, aliases []AssetUnitAlias) []AssetUnitAlias {
	seen := map[string]bool{}
	out := make([]AssetUnitAlias, 0, len(aliases))
	for _, alias := range aliases {
		alias.Alias = strings.ToUpper(strings.TrimSpace(alias.Alias))
		if alias.Alias == "" || alias.Alias == canonical || alias.Multiplier <= 0 || seen[alias.Alias] {
			continue
		}
		seen[alias.Alias] = true
		out = append(out, alias)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Multiplier == out[j].Multiplier {
			return out[i].Alias < out[j].Alias
		}
		return out[i].Multiplier < out[j].Multiplier
	})
	return out
}

func inferScaledUnitAlias(value string, canonicalSet map[string]bool) (string, float64, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", 0, false
	}

	type unitPrefix struct {
		token      string
		multiplier float64
	}
	prefixes := []unitPrefix{
		{token: "1000000", multiplier: 1000000},
		{token: "10000", multiplier: 10000},
		{token: "1000", multiplier: 1000},
		{token: "100", multiplier: 100},
		{token: "1M", multiplier: 1000000},
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix.token) && len(value) > len(prefix.token) {
			base := value[len(prefix.token):]
			if isValidScaledBase(base) && canonicalSet[base] {
				return base, prefix.multiplier, true
			}
		}
		if strings.HasSuffix(value, prefix.token) && len(value) > len(prefix.token) {
			base := value[:len(value)-len(prefix.token)]
			if isValidScaledBase(base) && canonicalSet[base] {
				return base, prefix.multiplier, true
			}
		}
	}
	return "", 0, false
}

func isValidScaledBase(value string) bool {
	if len(value) < 2 {
		return false
	}
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
