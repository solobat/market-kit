package discovery

import (
	"encoding/json"
	"strings"
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

func TestBuildAssetGroupsSurfacesCrossClassTickerCollisionForReview(t *testing.T) {
	registry := identity.Registry{
		AssetCollisions: []identity.AssetCollision{{Symbol: "ON", AssetClasses: []string{"crypto", "rwa_stock"}}},
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "binance", RawSymbol: "ONUSDT", MarketType: "perpetual", CanonicalSymbol: "ON/USDT", AssetClass: "crypto", ComparisonKey: "crypto:ticker:on/USDT", ComparisonStatus: identity.ComparisonEligible},
			{Exchange: "bybit", RawSymbol: "ONUSDT", MarketType: "perpetual", CanonicalSymbol: "ON/USDT", AssetClass: "rwa_stock", ComparisonKey: "equity:us:on/USDT", ComparisonStatus: identity.ComparisonEligible},
		},
	}
	groups := NewAggregator(registry).BuildAssetGroups([]ImportedMarket{
		{PlatformID: "binance", VenueType: "cex", MarketType: "perpetual", Symbol: "ONUSDT", BaseAsset: "ON", QuoteAsset: "USDT"},
		{PlatformID: "bybit", VenueType: "cex", MarketType: "perpetual", Symbol: "ONUSDT", BaseAsset: "ON", QuoteAsset: "USDT"},
	})
	if len(groups) != 2 {
		t.Fatalf("cross-class ticker collision must produce separate groups: %+v", groups)
	}
	if groups[0].GroupKey == groups[1].GroupKey || groups[0].AssetClass == groups[1].AssetClass {
		t.Fatalf("different asset classes must have distinct group identities: %+v", groups)
	}
	for _, group := range groups {
		if len(group.Markets) != 1 || group.ComparisonKey == "" {
			t.Fatalf("each explicit identity must retain its comparison key: %+v", group)
		}
	}
}

func TestImportedMarketUnmarshalCollectsStatusFlags(t *testing.T) {
	var market ImportedMarket
	if err := json.Unmarshal([]byte(`{
		"sourceId":"slipstream",
		"platformId":"bitget",
		"marketType":"spot",
		"symbol":"FOOUSDT",
		"baseAsset":"FOO",
		"quoteAsset":"USDT",
		"st":"1",
		"pre_delisting":"true",
		"flags":["Special Treatment"]
	}`), &market); err != nil {
		t.Fatalf("unmarshal market: %v", err)
	}

	if !market.ST || !market.PreDelisting {
		t.Fatalf("expected st and pre-delisting flags, got %+v", market)
	}
	if !containsString(market.Flags, MarketFlagST) || !containsString(market.Flags, MarketFlagPreDelisting) {
		t.Fatalf("expected normalized flags, got %+v", market.Flags)
	}
}

func TestNormalizeImportedMarketsPreservesStatusFlags(t *testing.T) {
	aggregator := NewAggregator(identity.Registry{})
	markets := aggregator.NormalizeImportedMarkets([]ImportedMarket{
		{
			SourceID:     "slipstream",
			PlatformID:   "bitget",
			Platform:     "Bitget",
			VenueType:    "cex",
			MarketType:   "spot",
			Symbol:       "FOOUSDT",
			BaseAsset:    "FOO",
			QuoteAsset:   "USDT",
			ST:           true,
			PreDelisting: true,
		},
	})

	if len(markets) != 1 {
		t.Fatalf("expected one market, got %d", len(markets))
	}
	if !markets[0].ST || !markets[0].PreDelisting {
		t.Fatalf("expected status flags on candidate, got %+v", markets[0])
	}
	if !containsString(markets[0].Flags, MarketFlagST) || !containsString(markets[0].Flags, MarketFlagPreDelisting) {
		t.Fatalf("expected normalized candidate flags, got %+v", markets[0].Flags)
	}

	groups := aggregator.BuildAssetGroups([]ImportedMarket{
		{
			SourceID:     "slipstream",
			PlatformID:   "bitget",
			Platform:     "Bitget",
			VenueType:    "cex",
			MarketType:   "spot",
			Symbol:       "FOOUSDT",
			BaseAsset:    "FOO",
			QuoteAsset:   "USDT",
			ST:           true,
			PreDelisting: true,
		},
	})
	if len(groups) != 1 || !containsString(groups[0].Flags, MarketFlagST) || !containsString(groups[0].Flags, MarketFlagPreDelisting) {
		t.Fatalf("expected grouped status flags, got %+v", groups)
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

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
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

func TestNormalizeImportedMarketsPrefersRegistryRWAOverGenericCryptoHint(t *testing.T) {
	registry := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "WMT", AssetClass: "rwa_stock"},
		},
	}
	aggregator := NewAggregator(registry)
	markets := aggregator.NormalizeImportedMarkets([]ImportedMarket{
		{
			SourceID:       "market-kit-bootstrap",
			PlatformID:     "bitget",
			Platform:       "Bitget",
			VenueType:      "cex",
			MarketType:     "perp",
			Symbol:         "WMTUSDT",
			BaseAsset:      "WMT",
			QuoteAsset:     "USDT",
			AssetClassHint: "crypto",
			Category:       "token",
			Tags:           []string{"coin"},
		},
	})

	if len(markets) != 1 {
		t.Fatalf("expected 1 normalized market, got %d", len(markets))
	}
	if markets[0].AssetClass != "rwa_stock" {
		t.Fatalf("expected registry RWA class to beat generic crypto hint, got %+v", markets[0])
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

	if len(groups) != 2 {
		t.Fatalf("USO ETF and USOIL/USDH must stay separate: %+v", groups)
	}
	for _, group := range groups {
		if group.CanonicalAsset == "USOIL" && !group.NeedsReview {
			t.Fatalf("unverified oil token needs review: %+v", group)
		}
	}

}

func TestBuildAssetGroupsKeepsGenericHyperliquidHIP3Namespaced(t *testing.T) {
	registry := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "CBRS", AssetClass: "crypto"},
		},
	}
	aggregator := NewAggregator(registry)
	groups := aggregator.BuildAssetGroups([]ImportedMarket{
		{
			SourceID:   "market-kit-bootstrap",
			PlatformID: "hyperliquid",
			Platform:   "Hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "xyz:CBRS",
			BaseAsset:  "xyz:CBRS",
			QuoteAsset: "USDC",
			Status:     "live",
		},
		{
			SourceID:   "market-kit-bootstrap",
			PlatformID: "bitget",
			Platform:   "Bitget",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "CBRSUSDT",
			BaseAsset:  "CBRS",
			QuoteAsset: "USDT",
			Status:     "live",
		},
	})

	if len(groups) != 2 {
		t.Fatalf("expected namespaced HIP-3 market to stay separate, got %+v", groups)
	}
	foundCEX := false
	foundHL := false
	for _, market := range groups[0].Markets {
		if market.Exchange == "bitget" && market.CanonicalSymbol == "CBRS/USDT" {
			foundCEX = true
		}
		if market.Exchange == "hyperliquid" && market.RawSymbol == "xyz:CBRS" && market.CanonicalSymbol == "XYZ:CBRS/USDT" {
			foundHL = true
		}
	}
	for _, market := range groups[1].Markets {
		if market.Exchange == "bitget" && market.CanonicalSymbol == "CBRS/USDT" {
			foundCEX = true
		}
		if market.Exchange == "hyperliquid" && market.RawSymbol == "xyz:CBRS" && market.CanonicalSymbol == "XYZ:CBRS/USDT" {
			foundHL = true
		}
	}
	if !foundCEX || !foundHL {
		t.Fatalf("expected separate CEX and namespaced HIP-3 markets, got %+v", groups)
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
		if market.Symbol == "cash:WTI" {
			if len(groups) != 2 {
				t.Fatalf("officially inactive cash WTI must remain separate: %+v", groups)
			}
			for _, group := range groups {
				for _, row := range group.Markets {
					if row.RawSymbol == "cash:WTI" && row.ComparisonStatus != identity.ComparisonProhibited {
						t.Fatalf("inactive contract remains comparable: %+v", row)
					}
				}
			}
			continue
		}
		if market.Symbol == "WTIOIL/USDH" {
			if len(groups) != 2 {
				t.Fatalf("USDH must remain distinct from USDT: %+v", groups)
			}
			continue
		}
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
			if len(groups) != 2 {
				t.Fatalf("expected %s to group with %s, got %d groups: %+v", tc.symbol, tc.okxSymbol, len(groups), groups)
			}
			found := map[string]bool{}
			for _, group := range groups {
				found[group.GroupKey] = true
				if len(group.Markets) != 1 {
					t.Fatalf("quote groups must remain separate: %+v", groups)
				}
			}
			usdc := strings.TrimSuffix(tc.expectedGroupKey, "USDT") + "USDC"
			if !found[tc.expectedGroupKey] || !found[usdc] {
				t.Fatalf("expected both quote currencies: %+v", groups)
			}

		})
	}
}

func TestCATStockAnd1000CATCryptoStayInSeparateGroups(t *testing.T) {
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	groups := NewAggregator(registry).BuildAssetGroups([]ImportedMarket{
		{PlatformID: "binance", MarketType: "perpetual", Symbol: "1000CATUSDT", BaseAsset: "1000CAT", QuoteAsset: "USDT"},
		{PlatformID: "qfex", MarketType: "perpetual", Symbol: "CAT-USD", BaseAsset: "CAT", QuoteAsset: "USD"},
		{PlatformID: "variational", MarketType: "perpetual", Symbol: "CAT", BaseAsset: "CAT", QuoteAsset: "USDC"},
	})
	resolved := identity.NewResolver(registry).Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: "1000CATUSDT", MarketTypeHint: "perpetual"})
	if resolved.Market == nil || resolved.Market.UnitMultiplier != 1000 {
		t.Fatalf("lost scaled crypto unit: %+v", resolved)
	}
	stock, crypto := 0, 0
	for _, group := range groups {
		for _, market := range group.Markets {
			if market.AssetClass != group.AssetClass {
				t.Fatalf("mixed asset classes: %+v", group)
			}
			if market.RawSymbol == "1000CATUSDT" {
				crypto++
				if group.AssetClass != "crypto" || market.AssetID != "crypto:ticker:cat" {
					t.Fatalf("lost crypto identity or scaled unit: %+v", market)
				}
			} else {
				stock++
				if group.AssetClass != "rwa_stock" || !strings.HasPrefix(market.AssetID, "equity:") {
					t.Fatalf("lost stock identity: %+v", market)
				}
			}
		}
	}
	if stock != 2 || crypto != 1 {
		t.Fatalf("missing CAT markets: %+v", groups)
	}
}
