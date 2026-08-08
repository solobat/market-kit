package identity

import "strings"

type Resolver struct {
	registry             Registry
	overridesByVenue     map[string][]MarketOverride
	overridesByCanonical map[string][]MarketOverride
	aliasesBySymbol      map[string][]baseAliasMatch
	collisionsBySymbol   map[string]AssetCollision
}

func NewResolver(registry Registry) *Resolver {
	registry.Normalize()
	r := &Resolver{
		registry:             registry,
		overridesByVenue:     map[string][]MarketOverride{},
		overridesByCanonical: map[string][]MarketOverride{},
		aliasesBySymbol:      map[string][]baseAliasMatch{},
		collisionsBySymbol:   map[string]AssetCollision{},
	}
	for _, item := range registry.MarketOverrides {
		marketType := normalizeMarketType(item.MarketType)
		venueKey := resolverVenueKey(item.Exchange, marketType, resolverVenueSymbol(item.Exchange, item.RawSymbol, marketType))
		r.overridesByVenue[venueKey] = append(r.overridesByVenue[venueKey], item)
		canonicalKey := resolverCanonicalKey(item.Exchange, marketType, item.CanonicalSymbol)
		r.overridesByCanonical[canonicalKey] = append(r.overridesByCanonical[canonicalKey], item)
	}
	for _, item := range registry.AssetAliases {
		r.aliasesBySymbol[item.Canonical] = append(r.aliasesBySymbol[item.Canonical], baseAliasMatch{rule: item})
		for _, alias := range item.Aliases {
			r.aliasesBySymbol[alias] = append(r.aliasesBySymbol[alias], baseAliasMatch{rule: item})
		}
		for _, alias := range item.UnitAliases {
			r.aliasesBySymbol[alias.Alias] = append(r.aliasesBySymbol[alias.Alias], baseAliasMatch{rule: item, unitAlias: alias.Alias, unitMultiplier: alias.Multiplier})
		}
	}
	for _, item := range registry.AssetCollisions {
		r.collisionsBySymbol[item.Symbol] = item
	}
	return r
}

func (r *Resolver) Resolve(req ResolveRequest) ResolveResult {
	exchange := r.normalizeExchange(req.Exchange)
	rawSymbol := strings.TrimSpace(req.Symbol)
	if exchange == "" {
		return ResolveResult{
			Status: ResolveUnresolved,
			Reason: "exchange is required",
		}
	}
	if rawSymbol == "" {
		return ResolveResult{
			Status: ResolveUnresolved,
			Reason: "symbol is required",
		}
	}

	if candidates := r.resolveOverrides(exchange, rawSymbol, req.MarketTypeHint); len(candidates) == 1 {
		return ResolveResult{
			Status:     ResolveResolved,
			Confidence: 1,
			Reason:     "matched explicit market override",
			Market:     &candidates[0],
		}
	} else if len(candidates) > 1 {
		return ResolveResult{
			Status:     ResolveAmbiguous,
			Confidence: 0.5,
			Reason:     "multiple explicit market overrides matched",
			Candidates: candidates,
		}
	}

	marketType, confident := inferMarketType(exchange, rawSymbol, req.MarketTypeHint, req.InstType, req.ProductType)
	if marketType == MarketTypeUnknown {
		return ResolveResult{
			Status: ResolveUnresolved,
			Reason: "market type could not be inferred",
		}
	}

	base, quote, ok := parseBaseQuote(exchange, rawSymbol, marketType, req.CanonicalSymbolHint)
	if !ok {
		return ResolveResult{
			Status: ResolveUnresolved,
			Reason: "base/quote could not be derived",
		}
	}

	baseCanonical, assetClass, assetID, underlyingID, unitAlias, unitMultiplier, ambiguous := r.resolveBaseAlias(base)
	if ambiguous {
		return ResolveResult{
			Status: ResolveAmbiguous,
			Reason: "base asset alias matched multiple candidates",
		}
	}
	if baseCanonical == "" {
		baseCanonical = base
	}
	if quote == "" {
		return ResolveResult{
			Status: ResolveUnresolved,
			Reason: "quote asset could not be derived",
		}
	}
	canonicalSymbol := baseCanonical + "/" + quote

	if candidates := r.resolveCanonicalOverrides(exchange, canonicalSymbol, marketType); len(candidates) == 1 {
		return ResolveResult{
			Status:     ResolveResolved,
			Confidence: 1,
			Reason:     "matched canonical symbol to explicit market override",
			Market:     &candidates[0],
		}
	} else if len(candidates) > 1 {
		return ResolveResult{
			Status:     ResolveAmbiguous,
			Confidence: 0.5,
			Reason:     "multiple explicit market overrides share the canonical symbol",
			Candidates: candidates,
		}
	}

	identity := MarketIdentity{
		Exchange:        exchange,
		MarketType:      marketType,
		RawSymbol:       rawSymbol,
		VenueSymbol:     normalizeVenueSymbol(exchange, rawSymbol, marketType),
		CanonicalSymbol: canonicalSymbol,
		BaseAsset:       baseCanonical,
		QuoteAsset:      quote,
		AssetClass:      assetClass,
		AssetID:         assetID,
		UnderlyingID:    underlyingID,
		RegistryVersion: r.registry.GeneratedVersion,
	}
	applyUnitConversion(&identity, unitAlias, unitMultiplier)
	if identity.AssetClass == "" {
		identity.AssetClass = "unknown"
	}
	if identity.AssetID == "" {
		identity.AssetID = CanonicalAssetID(identity.AssetClass, identity.BaseAsset)
	}
	if identity.UnderlyingID == "" {
		identity.UnderlyingID = identity.AssetID
	}
	identity.InstrumentKind = DefaultInstrumentKind(identity.MarketType, identity.AssetClass)
	identity.SettlementAsset = identity.QuoteAsset
	identity.IdentitySource = "heuristic"
	identity.ComparisonKey = DefaultComparisonKey(identity.UnderlyingID, identity.QuoteAsset)
	identity.ComparisonStatus = ComparisonEligible
	if identity.ComparisonKey == "" {
		identity.ComparisonStatus = ComparisonAmbiguous
	}
	if _, collides := r.collisionsBySymbol[base]; collides {
		identity.ComparisonStatus = ComparisonAmbiguous
		return ResolveResult{
			Status:     ResolveAmbiguous,
			Confidence: 0.5,
			Reason:     "asset identity requires explicit market override",
			Candidates: []MarketIdentity{identity},
		}
	}

	confidence := 0.85
	reason := "resolved using heuristics"
	if confident {
		confidence = 0.95
		reason = "resolved using exchange-specific market inference"
	}
	return ResolveResult{
		Status:     ResolveResolved,
		Confidence: confidence,
		Reason:     reason,
		Market:     &identity,
	}
}

func (r *Resolver) normalizeExchange(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := r.registry.ExchangeAliases[value]; ok {
		return normalized
	}
	return value
}

func (r *Resolver) resolveOverrides(exchange string, rawSymbol string, marketTypeHint string) []MarketIdentity {
	matches := make([]MarketIdentity, 0, 1)
	hinted := normalizeMarketType(marketTypeHint)
	marketTypes := []MarketType{hinted}
	if hinted == MarketTypeUnknown {
		marketTypes = []MarketType{MarketTypeSpot, MarketTypePerpetual, MarketTypeFuture}
	}
	seen := map[string]bool{}
	for _, marketType := range marketTypes {
		venueSymbol := resolverVenueSymbol(exchange, rawSymbol, marketType)
		for _, item := range r.overridesByVenue[resolverVenueKey(exchange, marketType, venueSymbol)] {
			key := marketOverrideKey(item)
			if seen[key] || !overrideRawSymbolMatches(exchange, item.RawSymbol, rawSymbol, marketType) {
				continue
			}
			seen[key] = true
			matches = append(matches, r.marketIdentityFromOverride(exchange, item, marketType))
		}
	}
	return matches
}

func (r *Resolver) resolveCanonicalOverrides(exchange string, canonicalSymbol string, marketType MarketType) []MarketIdentity {
	if exchange == "" || canonicalSymbol == "" || marketType == MarketTypeUnknown {
		return nil
	}

	matches := make([]MarketIdentity, 0, 1)
	for _, item := range r.overridesByCanonical[resolverCanonicalKey(exchange, marketType, canonicalSymbol)] {
		matches = append(matches, r.marketIdentityFromOverride(exchange, item, marketType))
	}
	return matches
}

func (r *Resolver) marketIdentityFromOverride(exchange string, override MarketOverride, marketType MarketType) MarketIdentity {
	rawSymbol := strings.TrimSpace(override.RawSymbol)
	canonicalSymbol := strings.TrimSpace(override.CanonicalSymbol)
	base, quote := splitCanonicalSymbol(canonicalSymbol)
	baseCanonical, aliasClass, aliasAssetID, aliasUnderlyingID, _, _, _ := r.resolveBaseAlias(base)
	if baseCanonical == "" {
		baseCanonical = base
	}
	identity := MarketIdentity{
		Exchange:         exchange,
		MarketType:       marketType,
		RawSymbol:        rawSymbol,
		VenueSymbol:      normalizeVenueSymbol(exchange, rawSymbol, marketType),
		CanonicalSymbol:  canonicalSymbol,
		BaseAsset:        baseCanonical,
		QuoteAsset:       quote,
		AssetClass:       firstNonEmpty(override.AssetClass, aliasClass, "unknown"),
		AssetID:          firstNonEmpty(override.AssetID, aliasAssetID),
		UnderlyingID:     firstNonEmpty(override.UnderlyingID, aliasUnderlyingID),
		ComparisonKey:    override.ComparisonKey,
		ComparisonStatus: override.ComparisonStatus,
		RegistryVersion:  r.registry.GeneratedVersion,
		InstrumentKind:   override.InstrumentKind,
		ContractType:     override.ContractType,
		SettlementAsset:  override.SettlementAsset,
		ExpiryAtMs:       override.ExpiryAtMs,
		IdentitySource:   "explicit_override",
	}
	if identity.AssetID == "" {
		identity.AssetID = CanonicalAssetID(identity.AssetClass, identity.BaseAsset)
	}
	if identity.UnderlyingID == "" {
		identity.UnderlyingID = identity.AssetID
	}
	if identity.ComparisonKey == "" {
		identity.ComparisonKey = DefaultComparisonKey(identity.UnderlyingID, identity.QuoteAsset)
	}
	if identity.ComparisonStatus == "" {
		identity.ComparisonStatus = ComparisonEligible
	}
	if identity.InstrumentKind == "" {
		identity.InstrumentKind = DefaultInstrumentKind(identity.MarketType, identity.AssetClass)
	}
	if identity.SettlementAsset == "" {
		identity.SettlementAsset = identity.QuoteAsset
	}
	identity.ComparisonStatus = NormalizeComparisonStatus(identity.ComparisonStatus)
	if identity.ComparisonStatus == ComparisonEligible && identity.ComparisonKey == "" {
		identity.ComparisonStatus = ComparisonAmbiguous
	}

	rawBase, _, ok := parseBaseQuote(exchange, rawSymbol, marketType, "")
	if override.UnitMultiplier > 0 {
		unitAlias := strings.ToUpper(strings.TrimSpace(override.UnitAlias))
		if unitAlias == "" && ok {
			unitAlias = rawBase
		}
		if unitAlias == "" {
			unitAlias = baseCanonical
		}
		applyUnitConversion(&identity, unitAlias, override.UnitMultiplier)
		return identity
	}
	if ok {
		rawBaseCanonical, _, _, _, unitAlias, unitMultiplier, ambiguous := r.resolveBaseAlias(rawBase)
		if !ambiguous && unitMultiplier > 0 && rawBaseCanonical == baseCanonical {
			applyUnitConversion(&identity, unitAlias, unitMultiplier)
		}
	}
	return identity
}

type baseAliasMatch struct {
	rule           AssetAliasRule
	unitAlias      string
	unitMultiplier float64
}

func (r *Resolver) resolveBaseAlias(base string) (canonical string, assetClass string, assetID string, underlyingID string, unitAlias string, unitMultiplier float64, ambiguous bool) {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		return "", "", "", "", "", 0, false
	}

	matches := r.aliasesBySymbol[base]

	if len(matches) == 0 {
		return "", "", "", "", "", 0, false
	}
	if len(matches) > 1 {
		return "", "", "", "", "", 0, true
	}
	match := matches[0]
	return match.rule.Canonical, match.rule.AssetClass, match.rule.AssetID, match.rule.UnderlyingID, match.unitAlias, match.unitMultiplier, false
}

func resolverVenueKey(exchange string, marketType MarketType, venueSymbol string) string {
	return strings.ToLower(strings.TrimSpace(exchange)) + "|" + string(marketType) + "|" + strings.ToUpper(strings.TrimSpace(venueSymbol))
}

func resolverVenueSymbol(exchange string, rawSymbol string, marketType MarketType) string {
	switch strings.ToLower(strings.TrimSpace(exchange)) {
	case "binance", "bybit", "bitget", "aster":
		return compactDerivativeSymbol(rawSymbol, marketType)
	default:
		return normalizeVenueSymbol(exchange, rawSymbol, marketType)
	}
}

func resolverCanonicalKey(exchange string, marketType MarketType, canonicalSymbol string) string {
	return strings.ToLower(strings.TrimSpace(exchange)) + "|" + string(marketType) + "|" + strings.ToUpper(strings.TrimSpace(canonicalSymbol))
}

func applyUnitConversion(identity *MarketIdentity, unitAlias string, unitMultiplier float64) {
	if identity == nil || unitAlias == "" || unitMultiplier <= 0 {
		return
	}
	identity.UnitAlias = unitAlias
	identity.UnitMultiplier = unitMultiplier
	identity.CanonicalPriceMultiplier = 1 / unitMultiplier
	identity.CanonicalQuantityMultiplier = unitMultiplier
}

func inferMarketType(exchange string, rawSymbol string, marketTypeHint string, instType string, productType string) (MarketType, bool) {
	if hinted := normalizeMarketType(marketTypeHint); hinted != MarketTypeUnknown {
		return hinted, true
	}

	instType = strings.ToLower(strings.TrimSpace(instType))
	productType = strings.ToLower(strings.TrimSpace(productType))
	raw := strings.ToUpper(strings.TrimSpace(rawSymbol))

	switch {
	case strings.Contains(instType, "spot"), strings.Contains(productType, "spot"):
		return MarketTypeSpot, true
	case strings.Contains(instType, "swap"), strings.Contains(productType, "swap"), strings.Contains(instType, "perp"), strings.Contains(productType, "perp"):
		return MarketTypePerpetual, true
	case strings.Contains(instType, "future"), strings.Contains(productType, "future"):
		return MarketTypeFuture, true
	}

	switch exchange {
	case "okx":
		if strings.HasSuffix(raw, "-SWAP") {
			return MarketTypePerpetual, true
		}
		if strings.Count(raw, "-") == 1 {
			return MarketTypeSpot, true
		}
	case "hyperliquid":
		if strings.Contains(raw, "/") {
			return MarketTypeSpot, true
		}
		if strings.Contains(raw, ":") || raw != "" {
			return MarketTypePerpetual, true
		}
	case "gate":
		if strings.Contains(raw, "_") {
			return MarketTypeUnknown, false
		}
	case "binance", "bybit", "bitget", "aster":
		if strings.Contains(raw, "/") || strings.Contains(raw, "-") || strings.Contains(raw, "_") {
			return MarketTypeSpot, false
		}
		return MarketTypePerpetual, false
	}

	return MarketTypeUnknown, false
}

func parseBaseQuote(exchange string, rawSymbol string, marketType MarketType, canonicalHint string) (base string, quote string, ok bool) {
	if canonicalHint != "" {
		base, quote = splitCanonicalSymbol(canonicalHint)
		if base != "" && quote != "" {
			return base, quote, true
		}
	}

	raw := strings.ToUpper(strings.TrimSpace(rawSymbol))
	if exchange == "okx" && strings.HasSuffix(raw, "-SWAP") {
		raw = strings.TrimSuffix(raw, "-SWAP")
	}
	if exchange == "hyperliquid" && marketType == MarketTypePerpetual && !strings.Contains(raw, "/") {
		return hyperliquidPerpBase(raw), "USDT", true
	}

	for _, sep := range []string{"/", "-", "_"} {
		if strings.Contains(raw, sep) {
			parts := strings.SplitN(raw, sep, 2)
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				quote := parts[1]
				if exchange == "hyperliquid" && quote == "USDH" {
					quote = "USDT"
				}
				return parts[0], quote, true
			}
		}
	}

	for _, q := range []string{"USDT", "USDC", "USD"} {
		if strings.HasSuffix(raw, q) && len(raw) > len(q) {
			return strings.TrimSuffix(raw, q), q, true
		}
	}
	return "", "", false
}

func normalizeVenueSymbol(exchange string, rawSymbol string, marketType MarketType) string {
	raw := strings.ToUpper(strings.TrimSpace(rawSymbol))
	switch exchange {
	case "okx":
		if marketType == MarketTypePerpetual && !strings.HasSuffix(raw, "-SWAP") {
			base, quote, ok := parseBaseQuote(exchange, raw, MarketTypeSpot, "")
			if ok {
				return base + "-" + quote + "-SWAP"
			}
		}
		if marketType == MarketTypeSpot && strings.HasSuffix(raw, "-SWAP") {
			return strings.TrimSuffix(raw, "-SWAP")
		}
		return raw
	case "gate":
		return strings.ReplaceAll(strings.ReplaceAll(raw, "/", "_"), "-", "_")
	case "binance", "bybit", "bitget", "aster":
		raw = strings.ReplaceAll(raw, "/", "")
		raw = strings.ReplaceAll(raw, "-", "")
		raw = strings.ReplaceAll(raw, "_", "")
		return raw
	case "hyperliquid":
		if marketType == MarketTypeSpot && strings.Contains(raw, "/") {
			return raw
		}
		if strings.Contains(raw, ":") || strings.HasPrefix(raw, "@") {
			return raw
		}
		return rawSymbolBase(raw)
	default:
		return raw
	}
}

func overrideRawSymbolMatches(exchange string, overrideRaw string, requestRaw string, marketType MarketType) bool {
	overrideRaw = strings.TrimSpace(overrideRaw)
	requestRaw = strings.TrimSpace(requestRaw)
	if strings.EqualFold(overrideRaw, requestRaw) {
		return true
	}

	switch exchange {
	case "binance", "bybit", "bitget", "aster":
		return compactDerivativeSymbol(overrideRaw, marketType) == compactDerivativeSymbol(requestRaw, marketType)
	default:
		return false
	}
}

func compactDerivativeSymbol(raw string, marketType MarketType) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "/", "")
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, "_", "")
	if marketType == MarketTypePerpetual {
		for _, suffix := range []string{"PERPETUAL", "PERP", "SWAP"} {
			raw = strings.TrimSuffix(raw, suffix)
		}
	}
	return raw
}

func normalizeMarketType(value string) MarketType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "spot":
		return MarketTypeSpot
	case "perpetual", "perp", "swap", "linear":
		return MarketTypePerpetual
	case "future", "futures", "delivery":
		return MarketTypeFuture
	default:
		return MarketTypeUnknown
	}
}

func splitCanonicalSymbol(value string) (string, string) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if strings.Contains(value, "/") {
		parts := strings.SplitN(value, "/", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", ""
}

func rawSymbolBase(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, ":") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	for _, sep := range []string{"/", "-", "_"} {
		if strings.Contains(raw, sep) {
			parts := strings.SplitN(raw, sep, 2)
			return strings.TrimSpace(parts[0])
		}
	}
	return raw
}

func hyperliquidPerpBase(raw string) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if strings.Contains(raw, ":") {
		return raw
	}
	return rawSymbolBase(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
