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
		"USDT":   "fiat_stable",
		"AEUR":   "fiat_stable",
		"PYUSD":  "fiat_stable",
		"AAPL":   "rwa_stock",
		"AAPLON": "rwa_stock",
		"AMZNX":  "rwa_stock",
		"TSM":    "rwa_stock",
		"SOXL":   "rwa_stock",
		"XLE":    "rwa_stock",
		"CL":     "rwa_commodity",
		"NATGAS": "rwa_commodity",
		"XAUT":   "rwa_commodity",
		"CVX":    "",
		"BTC":    "",
	}

	for asset, expected := range cases {
		if actual := inferAssetClassFallback(asset); actual != expected {
			t.Fatalf("asset %s expected %s got %s", asset, expected, actual)
		}
	}
}

func TestClassifyGeneratedAssetPrefersExchangeMetadata(t *testing.T) {
	item := discoveryItem{
		BaseAsset:      "CL",
		AssetClassHint: "commodity",
		Category:       "energy",
		Tags:           []string{"oil"},
	}

	if actual := classifyGeneratedAsset(item, "CL"); actual != "rwa_commodity" {
		t.Fatalf("expected exchange metadata to classify CL as rwa_commodity, got %s", actual)
	}
}

func TestBuildGeneratedRegistrySkipsUnknownAutoAliases(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "binance",
			VenueType:  "cex",
			MarketType: "perpetual",
			Symbol:     "FOOUSDT",
			BaseAsset:  "FOO",
			QuoteAsset: "USDT",
			Status:     "live",
		},
	})

	if len(registry.MarketOverrides) != 1 {
		t.Fatalf("expected generated override for FOOUSDT, got %d", len(registry.MarketOverrides))
	}
	if len(registry.AssetAliases) != 1 {
		t.Fatalf("expected only stable quote alias to be generated, got %d", len(registry.AssetAliases))
	}
	if registry.AssetAliases[0].Canonical != "USDT" || registry.AssetAliases[0].AssetClass != "fiat_stable" {
		t.Fatalf("unexpected generated alias: %+v", registry.AssetAliases[0])
	}
}

func TestBuildGeneratedRegistryCollapsesScaledUnitAssets(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "binance",
			VenueType:  "cex",
			MarketType: "perpetual",
			Symbol:     "1000PEPEUSDT",
			BaseAsset:  "1000PEPE",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
		{
			PlatformID: "okx",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "PEPE-USDT",
			BaseAsset:  "PEPE",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
	})

	assets := map[string]identity.AssetAliasRule{}
	for _, item := range registry.AssetAliases {
		assets[item.Canonical] = item
	}
	pepe, ok := assets["PEPE"]
	if !ok {
		t.Fatalf("expected PEPE asset alias to exist after collapse")
	}
	if len(pepe.UnitAliases) != 1 || pepe.UnitAliases[0].Alias != "1000PEPE" || pepe.UnitAliases[0].Multiplier != 1000 {
		t.Fatalf("expected 1000PEPE unit alias, got %+v", pepe.UnitAliases)
	}
	if _, exists := assets["1000PEPE"]; exists {
		t.Fatalf("expected scaled asset canonical to be collapsed away")
	}

	for _, item := range registry.MarketOverrides {
		if item.RawSymbol == "1000PEPEUSDT" && item.CanonicalSymbol != "PEPE/USDT" {
			t.Fatalf("expected scaled market override to rewrite to PEPE/USDT, got %+v", item)
		}
	}
}

func TestBuildGeneratedRegistrySkipsGateLeveragedTokens(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC3L_USDT",
			BaseAsset:  "BTC3L",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
		{
			PlatformID: "gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC_USDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
	})

	if len(registry.MarketOverrides) != 1 {
		t.Fatalf("expected leveraged gate token to be filtered, got %d overrides", len(registry.MarketOverrides))
	}
	if registry.MarketOverrides[0].RawSymbol != "BTC_USDT" {
		t.Fatalf("unexpected override retained: %+v", registry.MarketOverrides[0])
	}
}
