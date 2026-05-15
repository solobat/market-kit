package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

type discoveryEnvelope struct {
	Source string          `json:"source"`
	Items  []discoveryItem `json:"items"`
}

type discoveryItem struct {
	PlatformID         string   `json:"platformId"`
	VenueType          string   `json:"venueType"`
	MarketType         string   `json:"marketType"`
	Symbol             string   `json:"symbol"`
	BaseAsset          string   `json:"baseAsset"`
	QuoteAsset         string   `json:"quoteAsset"`
	Status             string   `json:"status"`
	AssetClass         string   `json:"assetClass"`
	AssetClassHint     string   `json:"assetClassHint"`
	Category           string   `json:"category"`
	UnderlyingCategory string   `json:"underlyingCategory"`
	Sector             string   `json:"sector"`
	Tags               []string `json:"tags"`
}

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
		"SOXL": true, "SOXS": true, "SPACEX": true, "SPY": true, "SQQQ": true,
		"STXSTOCK": true, "TCOM": true, "TSLA": true, "TSM": true, "UBER": true,
		"UNH": true, "USO": true, "WDC": true, "XLE": true, "XOM": true,
	}
	rwaCommodityAssets = map[string]bool{
		"BRENT": true, "BRENTOIL": true, "CL": true, "GOLD": true, "NATGAS": true, "NG": true,
		"PAXG": true, "SILVER": true, "WTI": true, "WTIOIL": true, "XAG": true, "XAGT": true,
		"XAU": true, "XAUT": true,
	}
)

func main() {
	inputPath := flag.String("input", "", "Path to slipstream discovery export JSON")
	outputPath := flag.String("output", filepath.Join("identity", "generated_registry.json"), "Path to write generated registry JSON")
	flag.Parse()

	if strings.TrimSpace(*inputPath) == "" {
		fatalf("missing required --input")
	}

	payload, err := os.ReadFile(*inputPath)
	if err != nil {
		fatalf("read input: %v", err)
	}

	var envelope discoveryEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		fatalf("decode input: %v", err)
	}

	registry := buildGeneratedRegistry(envelope.Items)
	encoded, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		fatalf("encode registry: %v", err)
	}
	encoded = append(encoded, '\n')

	if err := os.WriteFile(*outputPath, encoded, 0o644); err != nil {
		fatalf("write output: %v", err)
	}

	fmt.Printf("wrote %s with %d asset aliases and %d market overrides\n", *outputPath, len(registry.AssetAliases), len(registry.MarketOverrides))
}

func buildGeneratedRegistry(items []discoveryItem) identity.Registry {
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

func shouldInclude(item discoveryItem) bool {
	if !strings.EqualFold(strings.TrimSpace(item.VenueType), "cex") {
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

func classifyGeneratedAsset(item discoveryItem, asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if asset == "" {
		return ""
	}
	if stableAssets[asset] {
		return "fiat_stable"
	}
	if fromExchange := inferAssetClassFromExchangeMetadata(item, asset); fromExchange != "" {
		return fromExchange
	}
	return inferAssetClassFallback(asset)
}

func inferAssetClassFromExchangeMetadata(item discoveryItem, asset string) string {
	hints := []string{
		item.AssetClass,
		item.AssetClassHint,
		item.Category,
		item.UnderlyingCategory,
		item.Sector,
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
	case strings.HasSuffix(asset, "ON"):
		return inferAssetClassFallback(strings.TrimSuffix(asset, "ON"))
	case strings.HasSuffix(asset, "X") && len(asset) > 1:
		return inferAssetClassFallback(strings.TrimSuffix(asset, "X"))
	default:
		return ""
	}
}

func overrideKey(item identity.MarketOverride) string {
	return item.Exchange + "|" + item.RawSymbol + "|" + item.MarketType
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
