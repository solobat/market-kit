package curation

import (
	"sort"
	"strings"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

var (
	stableAssets = map[string]bool{
		"USDT": true, "USDC": true, "USD": true, "FDUSD": true, "USDE": true, "USDS": true,
		"DAI": true, "BUSD": true, "USDB": true, "USD1": true, "USDY": true, "RLUSD": true,
		"EURC": true, "EURT": true, "AEUR": true, "EURI": true, "PYUSD": true, "TUSD": true,
		"USDG": true, "USDGO": true, "USDTB": true, "WUSD": true, "XUSD": true,
	}
	rwaStockAssets = map[string]bool{
		"AAPL": true, "AAPLX": true, "AAOI": true, "ACN": true, "AMAT": true, "AMD": true, "AMZN": true,
		"ANTHROPIC": true,
		"APLD":      true, "APP": true, "ARM": true, "ASML": true, "AVGO": true, "BA": true,
		"BABA": true, "BILL": true, "BMNR": true, "COHR": true, "COIN": true, "COST": true, "CRCL": true,
		"CRWV": true, "CSCO": true, "D": true, "DAR": true, "DIA": true,
		"DIS": true, "DRAM": true, "EAT": true, "EWH": true, "EWJ": true, "EWT": true,
		"EWY": true, "F": true, "FIS": true, "FUTU": true, "GE": true, "GOOGL": true,
		"GME": true,
		"HD":  true, "HEI": true, "HIMS": true, "HOOD": true, "IAG": true, "IAU": true,
		"IBM": true, "INDA": true, "INTC": true, "IONQ": true, "ITOT": true, "IVV": true, "IWM": true,
		"JD": true, "KLAC": true, "KOPN": true, "KWEB": true, "LLY": true, "LWLG": true,
		"KORU": true, "MA": true, "MAS": true, "MCD": true, "META": true, "MP": true, "MPLX": true, "MRVL": true,
		"MSFT": true, "MSTR": true, "MU": true, "NBIS": true, "NFLX": true, "NIO": true,
		"NVDA": true, "OKLO": true, "ORCL": true, "OXY": true, "PAYP": true, "PLTR": true,
		"OPENAI": true, "PEP": true, "QCOM": true, "QQQ": true, "RDDT": true, "RIOT": true,
		"RKLB": true, "RTX": true, "SLV": true, "SNDK": true,
		"SKHYNIX": true, "SOXL": true, "SOXS": true, "SPACEX": true, "SPCX": true, "SPY": true, "SQQQ": true,
		"STXSTOCK": true, "TCOM": true, "TQQQ": true, "TSLA": true, "TSM": true, "UBER": true,
		"UNH": true, "USAR": true, "USO": true, "WDC": true, "WMT": true, "XLE": true, "XOM": true,
	}
	rwaCommodityAssets = map[string]bool{
		"BRENT": true, "BRENTOIL": true, "CL": true, "GOLD": true, "NATGAS": true, "NG": true,
		"PAXG": true, "SILVER": true, "WTI": true, "WTIOIL": true, "XAG": true, "XAGT": true,
		"XAU": true, "XAUT": true,
	}
)

type SuspiciousCryptoCandidate struct {
	Asset      string `json:"asset"`
	Exchange   string `json:"exchange,omitempty"`
	Symbol     string `json:"symbol,omitempty"`
	MarketType string `json:"marketType,omitempty"`
	Reason     string `json:"reason"`
}

func BuildGeneratedRegistry(items []discovery.ImportedMarket) identity.Registry {
	assets := map[string]identity.AssetAliasRule{}
	overrides := map[string]identity.MarketOverride{}
	hyperliquidAliases := inferHyperliquidHIP3AliasTargets(items)

	for _, item := range items {
		if !shouldInclude(item) {
			continue
		}

		exchange := strings.ToLower(strings.TrimSpace(item.PlatformID))
		base := strings.ToUpper(strings.TrimSpace(item.BaseAsset))
		quote := strings.ToUpper(strings.TrimSpace(item.QuoteAsset))
		symbol := strings.TrimSpace(item.Symbol)
		marketType := normalizeGeneratedMarketType(item.MarketType)

		if exchange == "" || base == "" || quote == "" || symbol == "" || marketType == "" {
			continue
		}

		venueBase := hyperliquidHIP3Base(symbol, base)
		if exchange == "hyperliquid" && marketType == "perpetual" {
			if target, ok := hyperliquidAliases[venueBase]; ok {
				base = target.Base
				quote = target.Quote
				ensureAsset(assets, base, target.AssetClass)
				addAssetAlias(assets, base, venueBase)
			}
		}

		ensureAsset(assets, base, classifyGeneratedAsset(item, base))
		ensureAsset(assets, quote, classifyGeneratedAsset(item, quote))

		override := identity.MarketOverride{
			Exchange:        exchange,
			RawSymbol:       symbol,
			MarketType:      marketType,
			CanonicalSymbol: base + "/" + quote,
		}
		overrides[overrideKey(override)] = override
	}

	assetList := make([]identity.AssetAliasRule, 0, len(assets))
	for _, item := range assets {
		assetList = append(assetList, item)
	}
	sort.Slice(assetList, func(i, j int) bool {
		return assetList[i].Canonical < assetList[j].Canonical
	})

	overrideList := make([]identity.MarketOverride, 0, len(overrides))
	for _, item := range overrides {
		overrideList = append(overrideList, item)
	}
	sort.Slice(overrideList, func(i, j int) bool {
		if overrideList[i].Exchange == overrideList[j].Exchange {
			if overrideList[i].RawSymbol == overrideList[j].RawSymbol {
				return overrideList[i].MarketType < overrideList[j].MarketType
			}
			return overrideList[i].RawSymbol < overrideList[j].RawSymbol
		}
		return overrideList[i].Exchange < overrideList[j].Exchange
	})

	registry := identity.Registry{
		ExchangeAliases: map[string]string{},
		AssetAliases:    assetList,
		MarketOverrides: overrideList,
	}
	registry.Normalize()
	return registry
}

func SuspiciousCryptoCandidates(items []discovery.ImportedMarket, registry identity.Registry, limit int) []SuspiciousCryptoCandidate {
	if limit <= 0 {
		limit = 25
	}
	registry.Normalize()
	assetClasses := map[string]string{}
	for _, item := range registry.AssetAliases {
		assetClasses[strings.ToUpper(strings.TrimSpace(item.Canonical))] = strings.TrimSpace(item.AssetClass)
	}

	seen := map[string]bool{}
	out := make([]SuspiciousCryptoCandidate, 0)
	for _, item := range items {
		base := strings.ToUpper(strings.TrimSpace(item.BaseAsset))
		if base == "" || stableAssets[base] || assetClasses[base] != "crypto" {
			continue
		}
		reason := suspiciousCryptoReason(item, base)
		if reason == "" {
			continue
		}
		key := base + "|" + strings.ToLower(strings.TrimSpace(item.PlatformID)) + "|" + strings.TrimSpace(item.Symbol)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, SuspiciousCryptoCandidate{
			Asset:      base,
			Exchange:   strings.ToLower(strings.TrimSpace(item.PlatformID)),
			Symbol:     strings.TrimSpace(item.Symbol),
			MarketType: normalizeGeneratedMarketType(item.MarketType),
			Reason:     reason,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Asset == out[j].Asset {
			if out[i].Exchange == out[j].Exchange {
				return out[i].Symbol < out[j].Symbol
			}
			return out[i].Exchange < out[j].Exchange
		}
		return out[i].Asset < out[j].Asset
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func suspiciousCryptoReason(item discovery.ImportedMarket, asset string) string {
	if hasExplicitRWAHint(item) {
		return "crypto asset has explicit RWA/stock metadata"
	}
	if wrapped, base := wrappedKnownStockAsset(asset); wrapped {
		return "crypto asset looks like wrapped stock ticker " + base
	}
	if strings.Contains(normalizedHintText(item.Symbol), "stock") {
		return "crypto market symbol contains stock marker"
	}
	return ""
}

func hasExplicitRWAHint(item discovery.ImportedMarket) bool {
	hints := []string{
		item.AssetClass,
		item.AssetClassHint,
		item.Category,
		item.UnderlyingCategory,
	}
	hints = append(hints, item.Tags...)
	for _, hint := range hints {
		class := normalizeAssetClassHint(hint)
		if class == "rwa_stock" || class == "rwa_commodity" {
			return true
		}
	}
	return false
}

func wrappedKnownStockAsset(asset string) (bool, string) {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	wrappers := []string{"STOCK", "ON", "G", "B", "X"}
	for _, suffix := range wrappers {
		if strings.HasSuffix(asset, suffix) && len(asset) > len(suffix) {
			base := strings.TrimSuffix(asset, suffix)
			if rwaStockAssets[base] {
				return true, base
			}
		}
	}
	if strings.HasPrefix(asset, "X") && len(asset) > 1 {
		base := strings.TrimPrefix(asset, "X")
		if rwaStockAssets[base] {
			return true, base
		}
	}
	return false, ""
}

type hyperliquidHIP3AliasTarget struct {
	Base       string
	Quote      string
	AssetClass string
}

func inferHyperliquidHIP3AliasTargets(items []discovery.ImportedMarket) map[string]hyperliquidHIP3AliasTarget {
	anchors := map[string]hyperliquidHIP3AliasTarget{}
	for _, item := range items {
		if !shouldInclude(item) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.PlatformID), "hyperliquid") {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(item.VenueType), "cex") {
			continue
		}
		base := strings.ToUpper(strings.TrimSpace(item.BaseAsset))
		quote := strings.ToUpper(strings.TrimSpace(item.QuoteAsset))
		if base == "" || quote == "" {
			continue
		}
		assetClass := classifyGeneratedAsset(item, base)
		if assetClass != "rwa_stock" && assetClass != "rwa_commodity" {
			continue
		}
		if existing, ok := anchors[base]; ok && preferredStableQuote(existing.Quote, quote) != quote {
			continue
		}
		anchors[base] = hyperliquidHIP3AliasTarget{Base: base, Quote: quote, AssetClass: assetClass}
	}

	out := map[string]hyperliquidHIP3AliasTarget{}
	for _, item := range items {
		if !shouldInclude(item) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(item.PlatformID), "hyperliquid") {
			continue
		}
		if normalizeGeneratedMarketType(item.MarketType) != "perpetual" {
			continue
		}
		venueBase := hyperliquidHIP3Base(item.Symbol, item.BaseAsset)
		if venueBase == "" || !strings.Contains(strings.TrimSpace(item.Symbol), ":") {
			continue
		}
		if target, ok := anchors[venueBase]; ok {
			out[venueBase] = target
			continue
		}
		if target, ok := inferWrappedStockTickerTarget(venueBase, anchors); ok {
			out[venueBase] = target
		}
	}
	return out
}

func inferWrappedStockTickerTarget(venueBase string, anchors map[string]hyperliquidHIP3AliasTarget) (hyperliquidHIP3AliasTarget, bool) {
	venueBase = strings.ToUpper(strings.TrimSpace(venueBase))
	if len(venueBase) < 3 || !strings.HasSuffix(venueBase, "X") {
		return hyperliquidHIP3AliasTarget{}, false
	}
	prefix := strings.TrimSuffix(venueBase, "X")
	if len(prefix) < 3 {
		return hyperliquidHIP3AliasTarget{}, false
	}

	var match hyperliquidHIP3AliasTarget
	count := 0
	for base, target := range anchors {
		if base == venueBase || target.AssetClass != "rwa_stock" || !strings.HasPrefix(base, prefix) {
			continue
		}
		match = target
		count++
	}
	return match, count == 1
}

func hyperliquidHIP3Base(symbol string, fallback string) string {
	symbol = strings.TrimSpace(symbol)
	if _, suffix, ok := strings.Cut(symbol, ":"); ok {
		return strings.ToUpper(strings.TrimSpace(suffix))
	}
	return strings.ToUpper(strings.TrimSpace(fallback))
}

func preferredStableQuote(left string, right string) string {
	left = strings.ToUpper(strings.TrimSpace(left))
	right = strings.ToUpper(strings.TrimSpace(right))
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	order := map[string]int{"USDT": 0, "USDC": 1, "USD": 2, "FDUSD": 3}
	leftRank, leftOK := order[left]
	rightRank, rightOK := order[right]
	if leftOK && rightOK {
		if rightRank < leftRank {
			return right
		}
		return left
	}
	if rightOK && !leftOK {
		return right
	}
	return left
}

func MergeGeneratedRegistry(existing identity.Registry, generated identity.Registry, prune bool) identity.Registry {
	existing = sanitizeExistingGeneratedRegistry(existing, generated)
	if prune {
		return generated
	}
	return existing.Merge(generated)
}

func sanitizeExistingGeneratedRegistry(existing identity.Registry, current identity.Registry) identity.Registry {
	current.Normalize()
	supportedRWA := map[string]string{}
	for _, item := range current.AssetAliases {
		if item.AssetClass == "rwa_stock" || item.AssetClass == "rwa_commodity" {
			supportedRWA[item.Canonical] = item.AssetClass
		}
	}

	out := identity.Registry{
		ExchangeAliases: existing.ExchangeAliases,
		MarketOverrides: existing.MarketOverrides,
		AssetAliases:    make([]identity.AssetAliasRule, 0, len(existing.AssetAliases)),
	}
	for _, item := range existing.AssetAliases {
		if currentClass, ok := supportedRWA[item.Canonical]; ok {
			item.AssetClass = currentClass
			out.AssetAliases = append(out.AssetAliases, item)
			continue
		}
		switch item.AssetClass {
		case "rwa_stock", "rwa_commodity":
			if preciseGeneratedAssetClass(item.Canonical) == item.AssetClass || supportedRWA[item.Canonical] == item.AssetClass {
				out.AssetAliases = append(out.AssetAliases, item)
			}
		default:
			out.AssetAliases = append(out.AssetAliases, item)
		}
	}
	out.Normalize()
	return out
}

func preciseGeneratedAssetClass(asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	switch {
	case stableAssets[asset]:
		return "fiat_stable"
	case rwaStockAssets[asset]:
		return "rwa_stock"
	case rwaCommodityAssets[asset]:
		return "rwa_commodity"
	default:
		return ""
	}
}

func shouldInclude(item discovery.ImportedMarket) bool {
	venueType := strings.ToLower(strings.TrimSpace(item.VenueType))
	if venueType != "cex" && venueType != "dex" {
		return false
	}
	if discovery.IsExcludedLeveragedToken(item.PlatformID, item.BaseAsset, item.Symbol) {
		return false
	}
	quote := strings.ToUpper(strings.TrimSpace(item.QuoteAsset))
	if !stableAssets[quote] {
		return false
	}
	marketType := strings.ToLower(strings.TrimSpace(item.MarketType))
	return marketType == "spot" || marketType == "perp" || marketType == "perpetual"
}

func normalizeGeneratedMarketType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "spot":
		return "spot"
	case "perp", "perpetual", "swap", "linear":
		return "perpetual"
	case "future", "futures", "delivery":
		return "future"
	default:
		return ""
	}
}

func ensureAsset(target map[string]identity.AssetAliasRule, canonical string, assetClass string) {
	canonical = strings.ToUpper(strings.TrimSpace(canonical))
	assetClass = strings.TrimSpace(assetClass)
	if canonical == "" || assetClass == "" {
		return
	}
	if _, exists := target[canonical]; exists {
		return
	}
	target[canonical] = identity.AssetAliasRule{
		Canonical:  canonical,
		AssetClass: assetClass,
		Aliases:    []string{},
	}
}

func addAssetAlias(target map[string]identity.AssetAliasRule, canonical string, alias string) {
	canonical = strings.ToUpper(strings.TrimSpace(canonical))
	alias = strings.ToUpper(strings.TrimSpace(alias))
	if canonical == "" || alias == "" || canonical == alias {
		return
	}
	item, ok := target[canonical]
	if !ok {
		return
	}
	for _, existing := range item.Aliases {
		if existing == alias {
			return
		}
	}
	item.Aliases = append(item.Aliases, alias)
	target[canonical] = item
}

func classifyGeneratedAsset(item discovery.ImportedMarket, asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if asset == "" {
		return ""
	}
	if stableAssets[asset] {
		return "fiat_stable"
	}
	if fallback := inferAssetClassFallback(asset); fallback == "rwa_stock" || fallback == "rwa_commodity" {
		return fallback
	}
	if fromExchange := inferAssetClassFromExchangeMetadata(item); fromExchange != "" {
		return fromExchange
	}
	return inferAssetClassFallback(asset)
}

func inferAssetClassFromExchangeMetadata(item discovery.ImportedMarket) string {
	hints := []string{
		item.AssetClass,
		item.AssetClassHint,
		item.Category,
		item.UnderlyingCategory,
	}
	hints = append(hints, item.Tags...)

	seen := map[string]bool{}
	for _, hint := range hints {
		if assetClass := normalizeAssetClassHint(hint); assetClass != "" {
			seen[assetClass] = true
		}
	}
	for _, assetClass := range []string{"fiat_stable", "rwa_stock", "rwa_commodity", "crypto"} {
		if seen[assetClass] {
			return assetClass
		}
	}
	return ""
}

func normalizeAssetClassHint(value string) string {
	raw := normalizedHintText(value)

	switch {
	case raw == "":
		return ""
	case strings.Contains(raw, "stable"):
		return "fiat_stable"
	case strings.Contains(raw, "stock"),
		strings.Contains(raw, "equity"),
		strings.Contains(raw, "tradfi"),
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

func normalizedHintText(value string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	raw = strings.ReplaceAll(raw, "_", " ")
	raw = strings.ReplaceAll(raw, "-", " ")
	return strings.Join(strings.Fields(raw), " ")
}

func inferAssetClassFallback(asset string) string {
	switch {
	case stableAssets[asset]:
		return "fiat_stable"
	case rwaStockAssets[asset]:
		return "rwa_stock"
	case rwaCommodityAssets[asset]:
		return "rwa_commodity"
	default:
		return ""
	}
}

func overrideKey(item identity.MarketOverride) string {
	return item.Exchange + "|" + item.RawSymbol + "|" + item.MarketType
}
