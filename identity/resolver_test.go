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

func TestResolveUnresolvedWithoutSymbol(t *testing.T) {
	resolver := NewResolver(Registry{})
	result := resolver.Resolve(ResolveRequest{Exchange: "okx"})
	if result.Status != ResolveUnresolved {
		t.Fatalf("expected unresolved result, got %+v", result)
	}
}
