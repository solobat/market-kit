package main

import (
	"testing"

	"github.com/solobat/market-kit/identity"
)

func TestBuildGeneratedRegistryFiltersToStableCEXMarkets(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "okx",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "DRAM-USDT-SWAP",
			BaseAsset:  "DRAM",
			QuoteAsset: "USDT",
			Status:     "live",
		},
		{
			PlatformID: "hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "DRAM",
			BaseAsset:  "DRAM",
			QuoteAsset: "USDT",
			Status:     "live",
		},
		{
			PlatformID: "okx",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC-EUR",
			BaseAsset:  "BTC",
			QuoteAsset: "EUR",
			Status:     "live",
		},
	})

	if len(registry.MarketOverrides) != 1 {
		t.Fatalf("expected 1 generated override, got %d", len(registry.MarketOverrides))
	}
	if registry.MarketOverrides[0] != (identity.MarketOverride{
		Exchange:        "okx",
		RawSymbol:       "DRAM-USDT-SWAP",
		MarketType:      "perpetual",
		CanonicalSymbol: "DRAM/USDT",
	}) {
		t.Fatalf("unexpected generated override: %+v", registry.MarketOverrides[0])
	}
	if len(registry.AssetAliases) != 2 {
		t.Fatalf("expected DRAM and USDT asset aliases, got %d", len(registry.AssetAliases))
	}
}

func TestInferAssetClass(t *testing.T) {
	cases := map[string]string{
		"USDT": "fiat_stable",
		"AAPL": "rwa_stock",
		"XAUT": "rwa_commodity",
		"BTC":  "crypto",
	}

	for asset, expected := range cases {
		if actual := inferAssetClass(asset); actual != expected {
			t.Fatalf("asset %s expected %s got %s", asset, expected, actual)
		}
	}
}
