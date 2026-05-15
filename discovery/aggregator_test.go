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

func TestNormalizeImportedMarketsSkipsGateLeveragedTokens(t *testing.T) {
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatalf("load default registry: %v", err)
	}

	aggregator := NewAggregator(registry)
	markets := aggregator.NormalizeImportedMarkets([]ImportedMarket{
		{
			SourceID:   "bootstrap",
			PlatformID: "gate",
			Platform:   "Gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC3L_USDT",
			BaseAsset:  "BTC3L",
			QuoteAsset: "USDT",
		},
		{
			SourceID:   "bootstrap",
			PlatformID: "gate",
			Platform:   "Gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC_USDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
		},
	})

	if len(markets) != 1 {
		t.Fatalf("expected only plain gate market to remain, got %d", len(markets))
	}
	if markets[0].RawSymbol != "BTC_USDT" {
		t.Fatalf("unexpected remaining market: %+v", markets[0])
	}
}

func TestBuildAssetGroupsUsesExplicitOverrideForHyperliquidKMMarket(t *testing.T) {
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
			Symbol:     "USO-USDT-SWAP",
			BaseAsset:  "USO",
			QuoteAsset: "USDT",
		},
		{
			SourceID:   "slipstream",
			PlatformID: "hyperliquid",
			Platform:   "Hyperliquid",
			VenueType:  "dex",
			MarketType: "spot",
			Symbol:     "USOIL/USDH",
			BaseAsset:  "USOIL",
			QuoteAsset: "USDH",
		},
	})

	if len(groups) != 1 {
		t.Fatalf("expected km market and okx market to group together, got %d groups: %+v", len(groups), groups)
	}
	if groups[0].GroupKey != "USO/USDT" {
		t.Fatalf("expected canonical USO/USDT group, got %s", groups[0].GroupKey)
	}
	if len(groups[0].Markets) != 2 {
		t.Fatalf("expected 2 grouped markets, got %d", len(groups[0].Markets))
	}
}

func TestBuildAssetGroupsUsesExplicitOverrideForHyperliquidWTIMarkets(t *testing.T) {
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatalf("load default registry: %v", err)
	}

	aggregator := NewAggregator(registry)
	for _, market := range []ImportedMarket{
		{
			SourceID:   "bootstrap",
			PlatformID: "hyperliquid",
			Platform:   "Hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "xyz:CL",
			BaseAsset:  "xyz:CL",
			QuoteAsset: "USDC",
		},
		{
			SourceID:   "bootstrap",
			PlatformID: "hyperliquid",
			Platform:   "Hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "cash:WTI",
			BaseAsset:  "cash:WTI",
			QuoteAsset: "USDC",
		},
		{
			SourceID:   "bootstrap",
			PlatformID: "hyperliquid",
			Platform:   "Hyperliquid",
			VenueType:  "dex",
			MarketType: "spot",
			Symbol:     "WTIOIL/USDH",
			BaseAsset:  "WTIOIL",
			QuoteAsset: "USDH",
		},
		{
			SourceID:   "bootstrap",
			PlatformID: "okx",
			Platform:   "OKX",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "CL-USDT-SWAP",
			BaseAsset:  "CL",
			QuoteAsset: "USDT",
		},
	} {
		groups := aggregator.BuildAssetGroups([]ImportedMarket{market, {
			SourceID:   "bootstrap",
			PlatformID: "okx",
			Platform:   "OKX",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "CL-USDT-SWAP",
			BaseAsset:  "CL",
			QuoteAsset: "USDT",
		}})
		if len(groups) != 1 {
			t.Fatalf("expected %s to group with okx CL, got %d groups: %+v", market.Symbol, len(groups), groups)
		}
		if groups[0].GroupKey != "CL/USDT" {
			t.Fatalf("expected canonical CL/USDT group for %s, got %s", market.Symbol, groups[0].GroupKey)
		}
		if len(groups[0].Markets) != 2 {
			t.Fatalf("expected 2 grouped markets for %s, got %d", market.Symbol, len(groups[0].Markets))
		}
	}
}

func TestBuildAssetGroupsUsesExplicitOverrideForHyperliquidCommodityAliasMarkets(t *testing.T) {
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatalf("load default registry: %v", err)
	}

	aggregator := NewAggregator(registry)
	for _, tc := range []struct {
		symbol           string
		baseAsset        string
		okxSymbol        string
		okxBaseAsset     string
		expectedGroupKey string
	}{
		{symbol: "xyz:BRENTOIL", baseAsset: "xyz:BRENTOIL", okxSymbol: "BZ-USDT-SWAP", okxBaseAsset: "BZ", expectedGroupKey: "BZ/USDT"},
		{symbol: "xyz:GOLD", baseAsset: "xyz:GOLD", okxSymbol: "XAU-USDT-SWAP", okxBaseAsset: "XAU", expectedGroupKey: "XAU/USDT"},
		{symbol: "xyz:SILVER", baseAsset: "xyz:SILVER", okxSymbol: "XAG-USDT-SWAP", okxBaseAsset: "XAG", expectedGroupKey: "XAG/USDT"},
		{symbol: "xyz:PALLADIUM", baseAsset: "xyz:PALLADIUM", okxSymbol: "XPD-USDT-SWAP", okxBaseAsset: "XPD", expectedGroupKey: "XPD/USDT"},
		{symbol: "xyz:PLATINUM", baseAsset: "xyz:PLATINUM", okxSymbol: "XPT-USDT-SWAP", okxBaseAsset: "XPT", expectedGroupKey: "XPT/USDT"},
	} {
		t.Run(tc.symbol, func(t *testing.T) {
			groups := aggregator.BuildAssetGroups([]ImportedMarket{
				{
					SourceID:   "bootstrap",
					PlatformID: "hyperliquid",
					Platform:   "Hyperliquid",
					VenueType:  "dex",
					MarketType: "perp",
					Symbol:     tc.symbol,
					BaseAsset:  tc.baseAsset,
					QuoteAsset: "USDC",
				},
				{
					SourceID:   "bootstrap",
					PlatformID: "okx",
					Platform:   "OKX",
					VenueType:  "cex",
					MarketType: "perp",
					Symbol:     tc.okxSymbol,
					BaseAsset:  tc.okxBaseAsset,
					QuoteAsset: "USDT",
				},
			})
			if len(groups) != 1 {
				t.Fatalf("expected %s to group with %s, got %d groups: %+v", tc.symbol, tc.okxSymbol, len(groups), groups)
			}
			if groups[0].GroupKey != tc.expectedGroupKey {
				t.Fatalf("expected canonical group %s for %s, got %s", tc.expectedGroupKey, tc.symbol, groups[0].GroupKey)
			}
			if len(groups[0].Markets) != 2 {
				t.Fatalf("expected 2 grouped markets for %s, got %d", tc.symbol, len(groups[0].Markets))
			}
		})
	}
}
