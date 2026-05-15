package identity

import "testing"

func TestRegistryMergePrefersExistingEntries(t *testing.T) {
	base := Registry{
		ExchangeAliases: map[string]string{"okex": "okx"},
		AssetAliases: []AssetAliasRule{
			{Canonical: "DRAM", AssetClass: "rwa_stock"},
		},
		MarketOverrides: []MarketOverride{
			{Exchange: "okx", RawSymbol: "DRAM-USDT-SWAP", MarketType: "perpetual", CanonicalSymbol: "DRAM/USDT"},
		},
	}
	other := Registry{
		ExchangeAliases: map[string]string{"okex": "okx-new", "okx-alt": "okx"},
		AssetAliases: []AssetAliasRule{
			{Canonical: "DRAM", AssetClass: "crypto"},
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
	if len(merged.MarketOverrides) != 2 {
		t.Fatalf("expected 2 unique market overrides, got %d", len(merged.MarketOverrides))
	}
}
