package identity

import "testing"

func TestAnthropicRebaseKeepsRetiredAndLiveProductsDistinct(t *testing.T) {
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
		if result.Status != ResolveResolved || m == nil || m.BaseAsset != "ANTHROPIC" || m.AssetClass != "rwa_stock" || m.UnitMultiplier != 1 {
			t.Fatalf("incorrect Anthropic identity: %+v market=%+v", result, m)
		}
		if q.Exchange == "binance" {
			if m.ComparisonStatus != ComparisonProhibited || m.ContractType != "settling_perpetual" {
				t.Fatalf("retired pre-rebase contract is not quarantined: %+v", m)
			}
			binance = m
		} else {
			if m.UnderlyingID != "equity:private:anthropic" || m.ComparisonStatus != ComparisonAmbiguous {
				t.Fatalf("Entropy identity or unit-review boundary changed: %+v", m)
			}
			entropy = m
		}
	}
	if binance.AssetID == entropy.AssetID || binance.ComparisonKey == entropy.ComparisonKey || entropy.SettlementAsset != "USDC" || binance.SettlementAsset != "USDT" {
		t.Fatal("distinct lifecycle or settlement boundary lost", binance, entropy)
	}
	for _, exchange := range []string{"binance", "bitget"} {
		result := r.Resolve(ResolveRequest{Exchange: exchange, Symbol: "ANTUSDT", MarketTypeHint: "perpetual"})
		m := result.Market
		if result.Status != ResolveResolved || m == nil || m.AssetClass != "rwa_stock" || m.ComparisonStatus != ComparisonAmbiguous {
			t.Fatalf("post-rebase product lost independent unit review: %+v", result)
		}
		if m.AssetID == binance.AssetID || m.ComparisonKey == binance.ComparisonKey || m.ComparisonKey == entropy.ComparisonKey {
			t.Fatalf("post-rebase product merged with old units: %+v", m)
		}
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
