package identity

import "testing"

func TestRegistryMergePrefersExistingEntries(t *testing.T) {
	base := Registry{
		ExchangeAliases: map[string]string{"okex": "okx"},
		AssetAliases: []AssetAliasRule{
			{Canonical: "DRAM", AssetClass: "rwa_stock", Aliases: []string{"DRAMX"}},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "okx", RawSymbol: "DRAM-USDT-SWAP", MarketType: "perpetual", CanonicalSymbol: "DRAM/USDT"},
		},
	}
	other := Registry{
		ExchangeAliases: map[string]string{"okex": "okx-new", "okx-alt": "okx"},
		AssetAliases: []AssetAliasRule{
			{Canonical: "DRAM", AssetClass: "crypto", UnitAliases: []AssetUnitAlias{{Alias: "1000DRAM", Multiplier: 1000}}},
			{Canonical: "BTC", AssetClass: "crypto"},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "okx", RawSymbol: "DRAM-USDT-SWAP", MarketType: "perpetual", CanonicalSymbol: "DRAM/USDT"},
			{Exchange: "binance", RawSymbol: "BTCUSDT", MarketType: "perpetual", CanonicalSymbol: "BTC/USDT"},
		},
	}

	merged := base.Merge(other)
	if merged.ExchangeAliases["okex"] != "okx" {
		t.Fatalf("expected existing alias to win, got %q", merged.ExchangeAliases["okex"])
	}
	if merged.ExchangeAliases["okx-alt"] != "okx" {
		t.Fatalf("expected new alias to be added")
	}
	if len(merged.AssetAliases) != 2 {
		t.Fatalf("expected 2 unique asset aliases, got %d", len(merged.AssetAliases))
	}
	if merged.AssetAliases[0].Canonical != "BTC" || merged.AssetAliases[1].Canonical != "DRAM" {
		t.Fatalf("unexpected asset ordering: %+v", merged.AssetAliases)
	}
	dram := merged.AssetAliases[1]
	if len(dram.Aliases) != 1 || dram.Aliases[0] != "DRAMX" {
		t.Fatalf("expected existing aliases to be preserved, got %+v", dram.Aliases)
	}
	if len(dram.UnitAliases) != 1 || dram.UnitAliases[0].Alias != "1000DRAM" || dram.UnitAliases[0].Multiplier != 1000 {
		t.Fatalf("expected unit aliases to merge, got %+v", dram.UnitAliases)
	}
	if len(merged.MarketOverrides) != 2 {
		t.Fatalf("expected 2 unique market overrides, got %d", len(merged.MarketOverrides))
	}
}

func TestRegistryNormalizeCollapsesScaledUnitAliases(t *testing.T) {
	registry := Registry{
		AssetAliases: []AssetAliasRule{
			{Canonical: "PEPE", AssetClass: "crypto"},
			{Canonical: "1000PEPE", AssetClass: "crypto"},
			{Canonical: "SHIB", AssetClass: "crypto"},
			{Canonical: "SHIB1000", AssetClass: "crypto"},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "binance", RawSymbol: "1000PEPEUSDT", MarketType: "perpetual", CanonicalSymbol: "1000PEPE/USDT"},
			{Exchange: "bybit", RawSymbol: "SHIB1000USDT", MarketType: "perpetual", CanonicalSymbol: "SHIB1000/USDT"},
		},
	}

	registry.Normalize()

	if len(registry.AssetAliases) != 2 {
		t.Fatalf("expected scaled aliases to collapse into 2 assets, got %d", len(registry.AssetAliases))
	}

	assets := map[string]AssetAliasRule{}
	for _, item := range registry.AssetAliases {
		assets[item.Canonical] = item
	}
	pepe, ok := assets["PEPE"]
	if !ok {
		t.Fatalf("expected PEPE asset to remain after collapse")
	}
	if len(pepe.UnitAliases) != 1 || pepe.UnitAliases[0].Alias != "1000PEPE" || pepe.UnitAliases[0].Multiplier != 1000 {
		t.Fatalf("expected 1000PEPE unit alias, got %+v", pepe.UnitAliases)
	}
	shib, ok := assets["SHIB"]
	if !ok {
		t.Fatalf("expected SHIB asset to remain after collapse")
	}
	if len(shib.UnitAliases) != 1 || shib.UnitAliases[0].Alias != "SHIB1000" || shib.UnitAliases[0].Multiplier != 1000 {
		t.Fatalf("expected SHIB1000 unit alias, got %+v", shib.UnitAliases)
	}

	if registry.MarketOverrides[0].CanonicalSymbol != "PEPE/USDT" || registry.MarketOverrides[1].CanonicalSymbol != "SHIB/USDT" {
		t.Fatalf("expected market overrides to be rewritten, got %+v", registry.MarketOverrides)
	}
}
