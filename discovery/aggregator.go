package discovery

import (
	"fmt"
	"sort"
	"strings"

	"github.com/solobat/market-kit/identity"
)

type Aggregator struct {
	registry identity.Registry
	resolver *identity.Resolver
}

func NewAggregator(registry identity.Registry) *Aggregator {
	registry.Normalize()
	return &Aggregator{
		registry: registry,
		resolver: identity.NewResolver(registry),
	}
}

func (a *Aggregator) NormalizeImportedMarkets(items []ImportedMarket) []CandidateMarket {
	out := make([]CandidateMarket, 0, len(items))
	for _, item := range items {
		candidate := a.normalizeImportedMarket(item)
		if candidate == nil {
			continue
		}
		out = append(out, *candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CanonicalSymbol == out[j].CanonicalSymbol {
			if out[i].Exchange == out[j].Exchange {
				return out[i].RawSymbol < out[j].RawSymbol
			}
			return out[i].Exchange < out[j].Exchange
		}
		return out[i].CanonicalSymbol < out[j].CanonicalSymbol
	})
	return out
}

func (a *Aggregator) BuildAssetGroups(items []ImportedMarket) []AssetCandidateGroup {
	candidates := a.NormalizeImportedMarkets(items)
	grouped := map[string][]CandidateMarket{}
	for _, candidate := range candidates {
		grouped[candidateGroupKey(candidate)] = append(grouped[candidateGroupKey(candidate)], candidate)
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]AssetCandidateGroup, 0, len(keys))
	for _, key := range keys {
		out = append(out, summarizeGroup(key, grouped[key]))
	}
	return out
}

func (a *Aggregator) normalizeImportedMarket(item ImportedMarket) *CandidateMarket {
	if ShouldIgnoreImportedMarket(item) {
		return nil
	}

	exchange := normalizeExchange(a.registry, firstNonEmpty(item.PlatformID, item.Platform))
	rawSymbol := strings.TrimSpace(item.Symbol)
	marketType := normalizeMarketType(item.MarketType)
	evidence := []string{"imported from slipstream market inventory"}

	base := strings.ToUpper(strings.TrimSpace(item.BaseAsset))
	quote := strings.ToUpper(strings.TrimSpace(item.QuoteAsset))
	if base != "" && quote != "" {
		evidence = append(evidence, "used explicit base/quote from discovery source")
	}

	confidence := 0.7
	assetClass := normalizeImportedAssetClassHints(
		item.AssetClass,
		item.AssetClassHint,
		item.Category,
		item.UnderlyingCategory,
		item.Tags,
	)
	if assetClass != "" {
		confidence = 0.95
		evidence = append(evidence, "used exchange-provided asset classification")
	} else {
		assetClass = "unknown"
	}
	canonicalBase, baseClass, aliasMatched := resolveAssetAlias(a.registry, base)
	if aliasMatched {
		base = canonicalBase
		if assetClass == "unknown" && baseClass != "" {
			assetClass = baseClass
		}
		confidence = 0.9
		evidence = append(evidence, "matched known asset alias")
	}

	if base == "" || quote == "" || marketType == identity.MarketTypeUnknown {
		resolved := a.resolver.Resolve(identity.ResolveRequest{
			Exchange:       exchange,
			Symbol:         rawSymbol,
			MarketTypeHint: item.MarketType,
		})
		if resolved.Market != nil {
			if base == "" {
				base = resolved.Market.BaseAsset
			}
			if quote == "" {
				quote = resolved.Market.QuoteAsset
			}
			if marketType == identity.MarketTypeUnknown {
				marketType = resolved.Market.MarketType
			}
			if assetClass == "unknown" && strings.TrimSpace(resolved.Market.AssetClass) != "" {
				assetClass = resolved.Market.AssetClass
			}
			evidence = append(evidence, resolved.Reason)
			if resolved.Confidence > confidence {
				confidence = resolved.Confidence
			}
		}
	}

	venueSymbol := rawSymbol
	if resolved := a.resolver.Resolve(identity.ResolveRequest{
		Exchange:       exchange,
		Symbol:         rawSymbol,
		MarketTypeHint: string(marketType),
	}); resolved.Market != nil {
		venueSymbol = resolved.Market.VenueSymbol
		if resolved.Confidence >= 1 {
			base = resolved.Market.BaseAsset
			quote = resolved.Market.QuoteAsset
			if resolved.Market.MarketType != identity.MarketTypeUnknown {
				marketType = resolved.Market.MarketType
			}
			if strings.TrimSpace(resolved.Market.AssetClass) != "" {
				assetClass = resolved.Market.AssetClass
			}
			evidence = append(evidence, resolved.Reason)
		}
		if base == "" {
			base = resolved.Market.BaseAsset
		}
		if quote == "" {
			quote = resolved.Market.QuoteAsset
		}
		if assetClass == "unknown" && strings.TrimSpace(resolved.Market.AssetClass) != "" {
			assetClass = resolved.Market.AssetClass
		}
		if resolved.Confidence > confidence {
			confidence = resolved.Confidence
		}
	}

	canonicalSymbol := ""
	if base != "" && quote != "" {
		canonicalSymbol = base + "/" + quote
	}

	candidate := CandidateMarket{
		SourceID:        firstNonEmpty(item.SourceID, string(SourceKindSlipstream)),
		PlatformID:      strings.TrimSpace(item.PlatformID),
		Platform:        strings.TrimSpace(item.Platform),
		Exchange:        exchange,
		VenueType:       strings.ToLower(strings.TrimSpace(item.VenueType)),
		MarketType:      marketType,
		RawSymbol:       rawSymbol,
		VenueSymbol:     venueSymbol,
		BaseAsset:       base,
		QuoteAsset:      quote,
		CanonicalSymbol: canonicalSymbol,
		AssetClass:      assetClass,
		Chain:           strings.TrimSpace(item.Chain),
		Status:          strings.TrimSpace(item.Status),
		ExternalURL:     strings.TrimSpace(item.ExternalURL),
		Confidence:      confidence,
		Evidence:        dedupeStrings(evidence),
		FirstSeenAt:     item.FirstSeenAt,
		LastSeenAt:      item.LastSeenAt,
	}
	return &candidate
}

func summarizeGroup(key string, markets []CandidateMarket) AssetCandidateGroup {
	if len(markets) == 0 {
		return AssetCandidateGroup{GroupKey: key}
	}

	exchanges := map[string]bool{}
	marketTypes := map[identity.MarketType]bool{}
	venueTypes := map[string]bool{}
	chains := map[string]bool{}
	evidence := []string{}
	needsReview := false
	primaryConfidence := 1.0

	for _, market := range markets {
		if market.Exchange != "" {
			exchanges[market.Exchange] = true
		}
		if market.MarketType != "" && market.MarketType != identity.MarketTypeUnknown {
			marketTypes[market.MarketType] = true
		}
		if market.VenueType != "" {
			venueTypes[market.VenueType] = true
		}
		if market.Chain != "" {
			chains[market.Chain] = true
		}
		evidence = append(evidence, market.Evidence...)
		if market.Confidence < primaryConfidence {
			primaryConfidence = market.Confidence
		}
		if market.AssetClass == "unknown" || market.BaseAsset == "" || market.QuoteAsset == "" {
			needsReview = true
		}
	}

	if len(marketTypes) > 1 {
		evidence = append(evidence, "group spans multiple market types")
	}

	representative := markets[0]
	return AssetCandidateGroup{
		GroupKey:          key,
		CanonicalAsset:    representative.BaseAsset,
		CanonicalSymbol:   representative.CanonicalSymbol,
		QuoteAsset:        representative.QuoteAsset,
		AssetClass:        representative.AssetClass,
		Exchanges:         sortedStringKeys(exchanges),
		MarketTypes:       sortedMarketTypes(marketTypes),
		VenueTypes:        sortedStringKeys(venueTypes),
		Chains:            sortedStringKeys(chains),
		NeedsReview:       needsReview,
		PrimaryConfidence: primaryConfidence,
		Evidence:          dedupeStrings(evidence),
		Markets:           markets,
	}
}

func candidateGroupKey(market CandidateMarket) string {
	base := firstNonEmpty(market.BaseAsset, "UNKNOWN")
	quote := firstNonEmpty(market.QuoteAsset, "UNKNOWN")
	return fmt.Sprintf("%s/%s", base, quote)
}

func normalizeExchange(reg identity.Registry, value string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := reg.ExchangeAliases[raw]; ok {
		return normalized
	}
	return raw
}

func normalizeMarketType(value string) identity.MarketType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "spot":
		return identity.MarketTypeSpot
	case "perp", "perpetual", "swap", "linear":
		return identity.MarketTypePerpetual
	case "future", "futures", "delivery":
		return identity.MarketTypeFuture
	default:
		return identity.MarketTypeUnknown
	}
}

func resolveAssetAlias(reg identity.Registry, value string) (canonical string, assetClass string, matched bool) {
	needle := strings.ToUpper(strings.TrimSpace(value))
	if needle == "" {
		return "", "", false
	}
	for _, rule := range reg.AssetAliases {
		if rule.Canonical == needle {
			return rule.Canonical, firstNonEmpty(rule.AssetClass, "unknown"), true
		}
		matched := false
		for _, alias := range rule.Aliases {
			if strings.EqualFold(alias, needle) {
				matched = true
				break
			}
		}
		if matched {
			return rule.Canonical, firstNonEmpty(rule.AssetClass, "unknown"), true
		}
		for _, alias := range rule.UnitAliases {
			if strings.EqualFold(alias.Alias, needle) {
				matched = true
				break
			}
		}
		if matched {
			return rule.Canonical, firstNonEmpty(rule.AssetClass, "unknown"), true
		}
	}
	return "", "", false
}

func normalizeImportedAssetClassHints(hints ...any) string {
	for _, hint := range hints {
		switch value := hint.(type) {
		case string:
			if assetClass := normalizeImportedAssetClassHint(value); assetClass != "" {
				return assetClass
			}
		case []string:
			for _, item := range value {
				if assetClass := normalizeImportedAssetClassHint(item); assetClass != "" {
					return assetClass
				}
			}
		}
	}
	return ""
}

func normalizeImportedAssetClassHint(value string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	raw = strings.ReplaceAll(raw, "_", " ")
	raw = strings.ReplaceAll(raw, "-", " ")
	raw = strings.Join(strings.Fields(raw), " ")

	switch {
	case raw == "":
		return ""
	case strings.Contains(raw, "stable"):
		return "fiat_stable"
	case strings.Contains(raw, "stock"),
		strings.Contains(raw, "equity"),
		strings.Contains(raw, "share"),
		strings.Contains(raw, "security"),
		strings.Contains(raw, "etf"),
		strings.Contains(raw, "index"):
		return "rwa_stock"
	case strings.Contains(raw, "commodity"),
		strings.Contains(raw, "metal"),
		strings.Contains(raw, "gold"),
		strings.Contains(raw, "silver"),
		strings.Contains(raw, "oil"),
		strings.Contains(raw, "crude"),
		strings.Contains(raw, "gas"),
		strings.Contains(raw, "energy"):
		return "rwa_commodity"
	case strings.Contains(raw, "crypto"),
		strings.Contains(raw, "blockchain"),
		strings.Contains(raw, "defi"),
		strings.Contains(raw, "meme"),
		strings.Contains(raw, "layer 1"),
		strings.Contains(raw, "layer1"),
		strings.Contains(raw, "token"),
		strings.Contains(raw, "coin"):
		return "crypto"
	default:
		return ""
	}
}

func sortedStringKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sortedMarketTypes(values map[identity.MarketType]bool) []identity.MarketType {
	out := make([]identity.MarketType, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Slice(out, func(i, j int) bool {
		return string(out[i]) < string(out[j])
	})
	return out
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
