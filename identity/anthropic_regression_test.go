package identity

import "testing"

func TestAnthropicPreIPOHasSharedUnderlyingAndDistinctLifecycle(t *testing.T) {
	reg, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver(reg)
	var binance, entropy *MarketIdentity
	for _, q := range []ResolveRequest{
		{Exchange: "binance", Symbol: "ANTHROPICUSDT", MarketTypeHint: "perpetual"},
		{Exchange: "binance", Symbol: "ANTHROPIC/USDT", MarketTypeHint: "perpetual"},
		{Exchange: "hyperliquid", Symbol: "io:ANTH", MarketTypeHint: "perpetual"},
		{Exchange: "hyperliquid", Symbol: "io:ANTH/USD", MarketTypeHint: "perpetual"},
	} {
		result := r.Resolve(q)
		m := result.Market
		if result.Status != ResolveResolved || m == nil || m.BaseAsset != "ANTHROPIC" || m.AssetClass != "rwa_stock" || m.UnderlyingID != "equity:private:anthropic" || m.ComparisonStatus != ComparisonAmbiguous || m.UnitMultiplier != 1 {
			t.Fatalf("incorrect Anthropic identity: %+v market=%+v", result, m)
		}
		if q.Exchange == "binance" {
			binance = m
		} else {
			entropy = m
		}
	}
	if binance.AssetID == entropy.AssetID || binance.ComparisonKey == entropy.ComparisonKey || entropy.SettlementAsset != "USDC" || binance.SettlementAsset != "USDT" {
		t.Fatal("distinct lifecycle or settlement boundary lost", binance, entropy)
	}
	for _, q := range []ResolveRequest{
		{Exchange: "binance", Symbol: "ANTHUSDT", MarketTypeHint: "spot"},
		{Exchange: "hyperliquid", Symbol: "para:ANTH", MarketTypeHint: "perpetual"},
		{Exchange: "gate", Symbol: "ANTHROPIC3L_USDT", MarketTypeHint: "spot"},
	} {
		result := r.Resolve(q)
		if result.Market != nil && result.Market.UnderlyingID == entropy.UnderlyingID {
			t.Fatalf("unverified product merged: %+v", result.Market)
		}
	}
}

func TestEquivalentSymbolOverridesDoNotHideUnitConflicts(t *testing.T) {
	for _, multiplier := range []float64{1, 1000} {
		r := NewResolver(Registry{MarketOverrides: []MarketOverride{
			{Exchange: "binance", RawSymbol: "ABCUSDT", MarketType: "perpetual", CanonicalSymbol: "ABC/USDT", AssetClass: "crypto", UnitMultiplier: 1},
			{Exchange: "binance", RawSymbol: "ABC/USDT", MarketType: "perpetual", CanonicalSymbol: "ABC/USDT", AssetClass: "crypto", UnitMultiplier: multiplier},
		}})
		got := r.Resolve(ResolveRequest{Exchange: "binance", Symbol: "ABCUSDT", MarketTypeHint: "perpetual"})
		if multiplier == 1 && got.Status != ResolveResolved {
			t.Fatalf("equivalent variants ambiguous: %+v", got)
		}
		if multiplier != 1 && (got.Status != ResolveAmbiguous || len(got.Candidates) != 2) {
			t.Fatalf("unit conflict hidden: %+v", got)
		}
	}
}
