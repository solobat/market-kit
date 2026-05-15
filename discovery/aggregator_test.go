package discovery

import (
	"testing"

	"github.com/solobat/market-kit/identity"
)

func TestBuildAssetGroupsGroupsCrossVenueMarkets(t *testing.T) {
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatalf("load default registry: %v", err)
	}

	aggregator := NewAggregator(registry)
	groups := aggregator.BuildAssetGroups([]ImportedMarket{
		{
			SourceID:   "slipstream",
			PlatformID: "okx",
			Platform:   "OKX",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "DRAM-USDT-SWAP",
			BaseAsset:  "DRAM",
			QuoteAsset: "USDT",
		},
		{
			SourceID:   "slipstream",
			PlatformID: "bitget",
			Platform:   "Bitget",
			VenueType:  "cex",
			MarketType: "perpetual",
			Symbol:     "DRAMUSDT",
			BaseAsset:  "DRAM",
			QuoteAsset: "USDT",
		},
	})

	if len(groups) != 1 {
		t.Fatalf("expected one grouped asset family, got %d", len(groups))
	}

	group := groups[0]
	if group.GroupKey != "DRAM/USDT" {
		t.Fatalf("unexpected group key: %s", group.GroupKey)
	}
	if len(group.Markets) != 2 {
		t.Fatalf("expected 2 markets in group, got %d", len(group.Markets))
	}
	if group.NeedsReview {
		t.Fatalf("expected explicit base/quote grouping to avoid review flag")
	}
}

func TestNormalizeImportedMarketsUsesResolverFallback(t *testing.T) {
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatalf("load default registry: %v", err)
	}

	aggregator := NewAggregator(registry)
	markets := aggregator.NormalizeImportedMarkets([]ImportedMarket{
		{
			SourceID:   "slipstream",
			PlatformID: "hyperliquid",
			Platform:   "Hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "DRAM",
		},
	})

	if len(markets) != 1 {
		t.Fatalf("expected 1 normalized market, got %d", len(markets))
	}
	if markets[0].CanonicalSymbol != "DRAM/USDT" {
		t.Fatalf("expected canonical symbol DRAM/USDT, got %s", markets[0].CanonicalSymbol)
	}
	if markets[0].VenueSymbol == "" {
		t.Fatalf("expected venue symbol to be normalized")
	}
}
