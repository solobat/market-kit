package identity

import "testing"

func TestResolveOverride(t *testing.T) {
	registry := Registry{
		MarketOverrides: []MarketOverride{
			{Exchange: "okx", RawSymbol: "DRAM-USDT-SWAP", MarketType: "perpetual", CanonicalSymbol: "DRAM/USDT"},
		},
	}
	resolver := NewResolver(registry)

	result := resolver.Resolve(ResolveRequest{
		Exchange: "okx",
		Symbol:   "DRAM-USDT-SWAP",
	})
	if result.Status != ResolveResolved || result.Market == nil {
		t.Fatalf("expected resolved override, got %+v", result)
	}
	if result.Market.CanonicalSymbol != "DRAM/USDT" || result.Market.MarketType != MarketTypePerpetual {
		t.Fatalf("unexpected override identity: %+v", result.Market)
	}
}

func TestResolveOverridePrefersMarketTypeHint(t *testing.T) {
	registry := Registry{
		MarketOverrides: []MarketOverride{
			{Exchange: "bitget", RawSymbol: "DRAMUSDT", MarketType: "spot", CanonicalSymbol: "DRAM/USDT"},
			{Exchange: "bitget", RawSymbol: "DRAMUSDT", MarketType: "perpetual", CanonicalSymbol: "DRAM/USDT"},
		},
	}
	resolver := NewResolver(registry)

	result := resolver.Resolve(ResolveRequest{
		Exchange:       "bitget",
		Symbol:         "DRAMUSDT",
		MarketTypeHint: "perp",
	})
	if result.Status != ResolveResolved || result.Market == nil {
		t.Fatalf("expected resolved override, got %+v", result)
	}
	if result.Market.MarketType != MarketTypePerpetual {
		t.Fatalf("expected perpetual match, got %+v", result.Market)
	}
}

func TestResolveOKXSpotHeuristic(t *testing.T) {
	resolver := NewResolver(Registry{})
	result := resolver.Resolve(ResolveRequest{
		Exchange: "okx",
		Symbol:   "BTC-USDT",
	})
	if result.Status != ResolveResolved || result.Market == nil {
		t.Fatalf("expected resolved okx spot, got %+v", result)
	}
	if result.Market.MarketType != MarketTypeSpot || result.Market.CanonicalSymbol != "BTC/USDT" {
		t.Fatalf("unexpected market: %+v", result.Market)
	}
}

func TestResolveUnitAliasToCanonicalBase(t *testing.T) {
	registry := Registry{
		AssetAliases: []AssetAliasRule{
			{
				Canonical:  "PEPE",
				AssetClass: "crypto",
				UnitAliases: []AssetUnitAlias{
					{Alias: "1000PEPE", Multiplier: 1000},
				},
			},
		},
	}
	resolver := NewResolver(registry)

	result := resolver.Resolve(ResolveRequest{
		Exchange: "binance",
		Symbol:   "1000PEPEUSDT",
	})
	if result.Status != ResolveResolved || result.Market == nil {
		t.Fatalf("expected unit alias to resolve, got %+v", result)
	}
	if result.Market.CanonicalSymbol != "PEPE/USDT" || result.Market.BaseAsset != "PEPE" {
		t.Fatalf("expected canonical PEPE market, got %+v", result.Market)
	}
}

func TestResolveHyperliquidKMOverride(t *testing.T) {
	registry := Registry{
		AssetAliases: []AssetAliasRule{
			{Canonical: "USO", AssetClass: "rwa_stock", Aliases: []string{"USOIL"}},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "hyperliquid", RawSymbol: "km:USOIL", MarketType: "perpetual", CanonicalSymbol: "USO/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "USOIL/USDH", MarketType: "spot", CanonicalSymbol: "USO/USDT"},
		},
	}
	resolver := NewResolver(registry)

	result := resolver.Resolve(ResolveRequest{
		Exchange:       "hyperliquid",
		Symbol:         "USOIL/USDH",
		MarketTypeHint: "spot",
	})
	if result.Status != ResolveResolved || result.Market == nil {
		t.Fatalf("expected resolved km override, got %+v", result)
	}
	if result.Market.CanonicalSymbol != "USO/USDT" || result.Market.BaseAsset != "USO" {
		t.Fatalf("expected canonical USO market, got %+v", result.Market)
	}
	if result.Market.VenueSymbol != "USOIL/USDH" {
		t.Fatalf("expected km venue symbol to be preserved, got %+v", result.Market)
	}
}

func TestResolveHyperliquidWTIOverrides(t *testing.T) {
	registry := Registry{
		AssetAliases: []AssetAliasRule{
			{Canonical: "CL", AssetClass: "rwa_commodity", Aliases: []string{"WTI", "WTIOIL"}},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "hyperliquid", RawSymbol: "xyz:CL", MarketType: "perpetual", CanonicalSymbol: "CL/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "cash:WTI", MarketType: "perpetual", CanonicalSymbol: "CL/USDT"},
		},
	}
	resolver := NewResolver(registry)

	for _, tc := range []struct {
		name           string
		symbol         string
		marketTypeHint string
		venueSymbol    string
	}{
		{name: "xyz perp", symbol: "xyz:CL", venueSymbol: "XYZ:CL"},
		{name: "cash perp", symbol: "cash:WTI", venueSymbol: "CASH:WTI"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := resolver.Resolve(ResolveRequest{
				Exchange:       "hyperliquid",
				Symbol:         tc.symbol,
				MarketTypeHint: tc.marketTypeHint,
			})
			if result.Status != ResolveResolved || result.Market == nil {
				t.Fatalf("expected resolved WTI override, got %+v", result)
			}
			if result.Market.CanonicalSymbol != "CL/USDT" || result.Market.BaseAsset != "CL" {
				t.Fatalf("expected canonical CL market, got %+v", result.Market)
			}
			if result.Market.VenueSymbol != tc.venueSymbol {
				t.Fatalf("expected preserved venue symbol %q, got %+v", tc.venueSymbol, result.Market)
			}
		})
	}
}

func TestResolveHyperliquidCanonicalSymbolIsAmbiguousAcrossHIP3Dexs(t *testing.T) {
	registry := Registry{
		AssetAliases: []AssetAliasRule{
			{Canonical: "CL", AssetClass: "rwa_commodity", Aliases: []string{"WTI", "WTIOIL"}},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "hyperliquid", RawSymbol: "xyz:CL", MarketType: "perpetual", CanonicalSymbol: "CL/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "cash:WTI", MarketType: "perpetual", CanonicalSymbol: "CL/USDT"},
		},
	}
	resolver := NewResolver(registry)

	result := resolver.Resolve(ResolveRequest{
		Exchange:       "hyperliquid",
		Symbol:         "CL/USDT",
		MarketTypeHint: "perpetual",
	})
	if result.Status != ResolveAmbiguous {
		t.Fatalf("expected ambiguous canonical CL market, got %+v", result)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected two HIP-3 candidates, got %+v", result.Candidates)
	}
}

func TestResolveHyperliquidCanonicalSymbolCanMapToUniqueHIP3Dex(t *testing.T) {
	registry := Registry{
		AssetAliases: []AssetAliasRule{
			{Canonical: "USO", AssetClass: "rwa_stock", Aliases: []string{"USOIL"}},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "hyperliquid", RawSymbol: "km:USOIL", MarketType: "perpetual", CanonicalSymbol: "USO/USDT"},
		},
	}
	resolver := NewResolver(registry)

	result := resolver.Resolve(ResolveRequest{
		Exchange:       "hyperliquid",
		Symbol:         "USO/USDT",
		MarketTypeHint: "perpetual",
	})
	if result.Status != ResolveResolved || result.Market == nil {
		t.Fatalf("expected unique HIP-3 mapping, got %+v", result)
	}
	if result.Market.RawSymbol != "km:USOIL" || result.Market.VenueSymbol != "KM:USOIL" {
		t.Fatalf("expected km USOIL venue mapping, got %+v", result.Market)
	}
}

func TestResolveHyperliquidCanonicalSymbolCanMapToUniqueKMBMNRDex(t *testing.T) {
	registry := Registry{
		MarketOverrides: []MarketOverride{
			{Exchange: "hyperliquid", RawSymbol: "km:BMNR", MarketType: "perpetual", CanonicalSymbol: "BMNR/USDT"},
		},
	}
	resolver := NewResolver(registry)

	result := resolver.Resolve(ResolveRequest{
		Exchange:       "hyperliquid",
		Symbol:         "BMNR/USDT",
		MarketTypeHint: "perpetual",
	})
	if result.Status != ResolveResolved || result.Market == nil {
		t.Fatalf("expected unique BMNR HIP-3 mapping, got %+v", result)
	}
	if result.Market.RawSymbol != "km:BMNR" || result.Market.VenueSymbol != "KM:BMNR" {
		t.Fatalf("expected km BMNR venue mapping, got %+v", result.Market)
	}
}

func TestResolveHyperliquidCommodityAliasOverrides(t *testing.T) {
	registry := Registry{
		AssetAliases: []AssetAliasRule{
			{Canonical: "BZ", AssetClass: "rwa_commodity", Aliases: []string{"BRENT", "BRENTOIL"}},
			{Canonical: "XAU", AssetClass: "rwa_commodity", Aliases: []string{"GOLD"}},
			{Canonical: "XAG", AssetClass: "rwa_commodity", Aliases: []string{"SILVER"}},
			{Canonical: "XPD", AssetClass: "rwa_commodity", Aliases: []string{"PALLADIUM"}},
			{Canonical: "XPT", AssetClass: "rwa_commodity", Aliases: []string{"PLATINUM"}},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "hyperliquid", RawSymbol: "xyz:BRENTOIL", MarketType: "perpetual", CanonicalSymbol: "BZ/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "xyz:GOLD", MarketType: "perpetual", CanonicalSymbol: "XAU/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "xyz:SILVER", MarketType: "perpetual", CanonicalSymbol: "XAG/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "xyz:PALLADIUM", MarketType: "perpetual", CanonicalSymbol: "XPD/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "xyz:PLATINUM", MarketType: "perpetual", CanonicalSymbol: "XPT/USDT"},
		},
	}
	resolver := NewResolver(registry)

	for _, tc := range []struct {
		name           string
		symbol         string
		expectedBase   string
		expectedSymbol string
		expectedVenue  string
	}{
		{name: "brent", symbol: "xyz:BRENTOIL", expectedBase: "BZ", expectedSymbol: "BZ/USDT", expectedVenue: "XYZ:BRENTOIL"},
		{name: "gold", symbol: "xyz:GOLD", expectedBase: "XAU", expectedSymbol: "XAU/USDT", expectedVenue: "XYZ:GOLD"},
		{name: "silver", symbol: "xyz:SILVER", expectedBase: "XAG", expectedSymbol: "XAG/USDT", expectedVenue: "XYZ:SILVER"},
		{name: "palladium", symbol: "xyz:PALLADIUM", expectedBase: "XPD", expectedSymbol: "XPD/USDT", expectedVenue: "XYZ:PALLADIUM"},
		{name: "platinum", symbol: "xyz:PLATINUM", expectedBase: "XPT", expectedSymbol: "XPT/USDT", expectedVenue: "XYZ:PLATINUM"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := resolver.Resolve(ResolveRequest{
				Exchange: "hyperliquid",
				Symbol:   tc.symbol,
			})
			if result.Status != ResolveResolved || result.Market == nil {
				t.Fatalf("expected resolved commodity override, got %+v", result)
			}
			if result.Market.BaseAsset != tc.expectedBase || result.Market.CanonicalSymbol != tc.expectedSymbol {
				t.Fatalf("expected %s -> %s, got %+v", tc.expectedBase, tc.expectedSymbol, result.Market)
			}
			if result.Market.VenueSymbol != tc.expectedVenue {
				t.Fatalf("expected preserved venue symbol %q, got %+v", tc.expectedVenue, result.Market)
			}
		})
	}
}

func TestResolveUnresolvedWithoutSymbol(t *testing.T) {
	resolver := NewResolver(Registry{})
	result := resolver.Resolve(ResolveRequest{Exchange: "okx"})
	if result.Status != ResolveUnresolved {
		t.Fatalf("expected unresolved result, got %+v", result)
	}
}
