package identity

import "testing"

func TestIPOPrimeTokensKeepProductAndNativeUnits(t *testing.T) {
	reg, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver(reg)
	for _, base := range []string{"PREOPAI", "PRESPCX"} {
		got := r.Resolve(ResolveRequest{Exchange: "bitget", Symbol: base + "USDT", MarketTypeHint: "spot"}).Market
		if got == nil || got.BaseAsset != base || got.AssetClass != "rwa_stock" || got.ComparisonStatus != ComparisonAmbiguous || got.InstrumentKind != "pre_ipo_token" || got.UnitMultiplier != 1 || got.AssetID == got.UnderlyingID {
			t.Fatal(got)
		}
	}
	old := r.Resolve(ResolveRequest{Exchange: "bitget", Symbol: "PRESPAXUSDT", MarketTypeHint: "spot"}).Market
	current := r.Resolve(ResolveRequest{Exchange: "bitget", Symbol: "PRESPCXUSDT", MarketTypeHint: "spot"}).Market
	if old != nil && old.ComparisonKey == current.ComparisonKey {
		t.Fatal("historical split unit collapsed", old, current)
	}
}
