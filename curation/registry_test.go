package curation

import (
	"testing"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

func TestBuildGeneratedRegistryIncludesStableQuotedDEXMarkets(t *testing.T) {
	registry := BuildGeneratedRegistry([]discovery.ImportedMarket{
		{
			SourceID:       "slipstream",
			PlatformID:     "aster",
			Platform:       "Aster",
			VenueType:      "dex",
			MarketType:     "perp",
			Symbol:         "ZK_USDT_Perp",
			BaseAsset:      "ZK",
			QuoteAsset:     "USDT",
			AssetClassHint: "crypto",
			Status:         "live",
		},
	})

	if len(registry.MarketOverrides) != 1 {
		t.Fatalf("expected one market override, got %+v", registry.MarketOverrides)
	}
	override := registry.MarketOverrides[0]
	if override.Exchange != "aster" || override.RawSymbol != "ZK_USDT_Perp" || override.MarketType != "perpetual" || override.CanonicalSymbol != "ZK/USDT" {
		t.Fatalf("unexpected override: %+v", override)
	}

	var foundZK bool
	for _, asset := range registry.AssetAliases {
		if asset.Canonical == "ZK" {
			foundZK = true
			if asset.AssetClass != "crypto" {
				t.Fatalf("expected ZK to classify as crypto, got %+v", asset)
			}
		}
	}
	if !foundZK {
		t.Fatalf("expected ZK asset alias, got %+v", registry.AssetAliases)
	}
}

func TestBuildGeneratedRegistryClassifiesBinanceTradFiPerpetualAsRWAStock(t *testing.T) {
	registry := BuildGeneratedRegistry([]discovery.ImportedMarket{
		{
			SourceID:           "market-kit-bootstrap",
			PlatformID:         "binance",
			Platform:           "Binance",
			VenueType:          "cex",
			MarketType:         "perp",
			Symbol:             "KORUUSDT",
			BaseAsset:          "KORU",
			QuoteAsset:         "USDT",
			AssetClassHint:     "equity",
			Category:           "tradifi_perpetual",
			UnderlyingCategory: "equity",
			Status:             "live",
		},
	})

	for _, asset := range registry.AssetAliases {
		if asset.Canonical == "KORU" {
			if asset.AssetClass != "rwa_stock" {
				t.Fatalf("expected KORU to classify as rwa_stock, got %+v", asset)
			}
			return
		}
	}
	t.Fatalf("expected KORU asset alias, got %+v", registry.AssetAliases)
}

func TestMergeGeneratedRegistryPromotesExistingCryptoWhenCurrentRWAHasEvidence(t *testing.T) {
	existing := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "KORU", AssetClass: "crypto"},
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
	}
	current := BuildGeneratedRegistry([]discovery.ImportedMarket{
		{
			SourceID:           "market-kit-bootstrap",
			PlatformID:         "binance",
			Platform:           "Binance",
			VenueType:          "cex",
			MarketType:         "perp",
			Symbol:             "KORUUSDT",
			BaseAsset:          "KORU",
			QuoteAsset:         "USDT",
			AssetClassHint:     "equity",
			Category:           "tradifi_perpetual",
			UnderlyingCategory: "equity",
			Status:             "live",
		},
	})

	merged := MergeGeneratedRegistry(existing, current, false)
	for _, asset := range merged.AssetAliases {
		if asset.Canonical == "KORU" {
			if asset.AssetClass != "rwa_stock" {
				t.Fatalf("expected current RWA evidence to correct KORU, got %+v", asset)
			}
			return
		}
	}
	t.Fatalf("expected KORU asset alias, got %+v", merged.AssetAliases)
}

func TestBuildGeneratedRegistryInfersHyperliquidHIP3RWAStockAlias(t *testing.T) {
	registry := BuildGeneratedRegistry([]discovery.ImportedMarket{
		{
			SourceID:   "market-kit-bootstrap",
			PlatformID: "binance",
			Platform:   "Binance",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "SKHYNIXUSDT",
			BaseAsset:  "SKHYNIX",
			QuoteAsset: "USDT",
			Status:     "live",
		},
		{
			SourceID:   "market-kit-bootstrap",
			PlatformID: "hyperliquid",
			Platform:   "Hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "xyz:SKHX",
			BaseAsset:  "SKHX",
			QuoteAsset: "USDC",
			Status:     "live",
		},
	})

	var foundAlias bool
	for _, item := range registry.AssetAliases {
		if item.Canonical != "SKHYNIX" {
			continue
		}
		if item.AssetClass != "rwa_stock" {
			t.Fatalf("expected SKHYNIX to classify as rwa_stock, got %+v", item)
		}
		for _, alias := range item.Aliases {
			if alias == "SKHX" {
				foundAlias = true
			}
		}
	}
	if !foundAlias {
		t.Fatalf("expected SKHYNIX alias SKHX, got %+v", registry.AssetAliases)
	}

	for _, override := range registry.MarketOverrides {
		if override.Exchange == "hyperliquid" && override.RawSymbol == "xyz:SKHX" {
			if override.CanonicalSymbol != "SKHYNIX/USDT" {
				t.Fatalf("expected xyz:SKHX to canonicalize to SKHYNIX/USDT, got %+v", override)
			}
			return
		}
	}
	t.Fatalf("expected hyperliquid xyz:SKHX override, got %+v", registry.MarketOverrides)
}
