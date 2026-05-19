package curation

import (
	"testing"

	"github.com/solobat/market-kit/discovery"
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
