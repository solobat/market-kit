package identity

import "testing"

func TestReviewedGateProductTypesNeverBorrowCryptoOrStableIDs(t *testing.T) {
	reg, e := LoadDefaultRegistry()
	if e != nil {
		t.Fatal(e)
	}
	r := NewResolver(reg)
	for _, q := range []struct{ symbol, class, kind string }{{"BOTZ_USDT", "rwa_stock", "equity_perpetual"}, {"CAT_USDT", "rwa_stock", "equity_perpetual"}, {"USDC_USDT", "unknown", "fx_perpetual"}, {"XCU_USDT", "rwa_commodity", "commodity_perpetual"}, {"H100_USDT", "unknown", "index_perpetual"}, {"BEN_USDT", "unknown", "perpetual"}} {
		v := r.Resolve(ResolveRequest{Exchange: "gate", Symbol: q.symbol, MarketTypeHint: "perpetual"})
		m := v.Market
		if v.Status != ResolveResolved || m == nil || m.AssetClass != q.class || m.InstrumentKind != q.kind || m.ComparisonStatus == ComparisonEligible || m.UnitMultiplier != 1 {
			t.Fatalf("wrong product boundary: %+v %+v", v, m)
		}
		if q.symbol == "USDC_USDT" && (m.AssetID == "fiat:ticker:usdc" || m.ComparisonStatus != ComparisonProhibited) {
			t.Fatal(m)
		}
	}
}
