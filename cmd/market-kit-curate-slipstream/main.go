package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/solobat/market-kit/identity"
)

type discoveryEnvelope struct {
	Source string          `json:"source"`
	Items  []discoveryItem `json:"items"`
}

type discoveryItem struct {
	PlatformID string `json:"platformId"`
	VenueType  string `json:"venueType"`
	MarketType string `json:"marketType"`
	Symbol     string `json:"symbol"`
	BaseAsset  string `json:"baseAsset"`
	QuoteAsset string `json:"quoteAsset"`
	Status     string `json:"status"`
}

var (
	stableAssets = map[string]bool{
		"USDT": true, "USDC": true, "USD": true, "FDUSD": true, "USDE": true, "USDS": true,
		"DAI": true, "BUSD": true, "USDB": true, "USD1": true, "USDY": true, "RLUSD": true,
		"EURC": true, "EURT": true,
	}
	rwaStockAssets = map[string]bool{
		"AAPL": true, "AAPLX": true, "AMZN": true, "GOOGL": true, "META": true, "MSFT": true,
		"NVDA": true, "TSLA": true, "QQQ": true, "SPY": true, "MSTR": true, "COIN": true,
		"DRAM": true,
	}
	rwaCommodityAssets = map[string]bool{
		"XAUT": true, "PAXG": true, "XAGT": true,
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

		ensureAsset(assets, base)
		ensureAsset(assets, quote)

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

func ensureAsset(target map[string]identity.AssetAliasRule, canonical string) {
	canonical = strings.ToUpper(strings.TrimSpace(canonical))
	if canonical == "" {
		return
	}
	if _, exists := target[canonical]; exists {
		return
	}
	target[canonical] = identity.AssetAliasRule{
		Canonical:  canonical,
		AssetClass: inferAssetClass(canonical),
		Aliases:    []string{},
	}
}

func inferAssetClass(asset string) string {
	switch {
	case stableAssets[asset]:
		return "fiat_stable"
	case rwaStockAssets[asset]:
		return "rwa_stock"
	case rwaCommodityAssets[asset]:
		return "rwa_commodity"
	default:
		return "crypto"
	}
}

func overrideKey(item identity.MarketOverride) string {
	return item.Exchange + "|" + item.RawSymbol + "|" + item.MarketType
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
