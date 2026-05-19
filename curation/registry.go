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
		"AAPL": true, "AAPLX": true, "AAOI": true, "AMAT": true, "AMD": true, "AMZN": true,
		"APLD": true, "APP": true, "ARM": true, "ASML": true, "AVGO": true, "BA": true,
		"BABA": true, "BILL": true, "COHR": true, "COIN": true, "COST": true, "CRCL": true,
		"CRWV": true, "CSCO": true, "D": true, "DAR": true, "DIA": true,
		"DIS": true, "DRAM": true, "EAT": true, "EWH": true, "EWJ": true, "EWT": true,
		"EWY": true, "F": true, "FIS": true, "FUTU": true, "GE": true, "GOOGL": true,
		"HD": true, "HEI": true, "HIMS": true, "HOOD": true, "IAG": true, "IAU": true,
		"INDA": true, "INTC": true, "IONQ": true, "ITOT": true, "IVV": true, "IWM": true,
		"JD": true, "KLAC": true, "KOPN": true, "KWEB": true, "LLY": true, "LWLG": true,
		"MAS": true, "MCD": true, "META": true, "MP": true, "MPLX": true, "MRVL": true,
		"MSFT": true, "MSTR": true, "MU": true, "NBIS": true, "NFLX": true, "NIO": true,
		"NVDA": true, "OKLO": true, "ORCL": true, "OXY": true, "PAYP": true, "PLTR": true,
		"QCOM": true, "QQQ": true, "RKLB": true, "RTX": true, "SLV": true, "SNDK": true,
		"SOXL": true, "SOXS": true, "SPACEX": true, "SPCX": true, "SPY": true, "SQQQ": true,
		"STXSTOCK": true, "TCOM": true, "TSLA": true, "TSM": true, "UBER": true,
		"UNH": true, "USO": true, "WDC": true, "XLE": true, "XOM": true,
	}
	rwaCommodityAssets = map[string]bool{
		"BRENT": true, "BRENTOIL": true, "CL": true, "GOLD": true, "NATGAS": true, "NG": true,
		"PAXG": true, "SILVER": true, "WTI": true, "WTIOIL": true, "XAG": true, "XAGT": true,
		"XAU": true, "XAUT": true,
	}
)

func BuildGeneratedRegistry(items []discovery.ImportedMarket) identity.Registry {
	assets := map[string]identity.AssetAliasRule{}
	overrides := map[string]identity.MarketOverride{}

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

func classifyGeneratedAsset(item discovery.ImportedMarket, asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if asset == "" {
		return ""
	}
	if stableAssets[asset] {
		return "fiat_stable"
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

	for _, hint := range hints {
		if assetClass := normalizeAssetClassHint(hint); assetClass != "" {
			return assetClass
		}
	}
	return ""
}

func normalizeAssetClassHint(value string) string {
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
