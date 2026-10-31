package identity

import "testing"

func TestBybitDatedContractNeverFallsBackToDateQuote(t *testing.T) {
	reg := Registry{MarketOverrides: []MarketOverride{{Exchange: "bybit", RawSymbol: "BTCUSDT-20NOV26", MarketType: "future", CanonicalSymbol: "BTC/USDT", AssetClass: "crypto", AssetID: "crypto:bybit:future:btc-20nov26", UnderlyingID: "crypto:ticker:btc", ComparisonStatus: ComparisonAmbiguous, ContractType: "linear_futures", SettlementAsset: "USDT", ExpiryAtMs: 1795161600000}}}
	reg.Normalize()
	r := NewResolver(reg)
	good := r.Resolve(ResolveRequest{Exchange: "bybit", Symbol: "BTCUSDT-20NOV26", MarketTypeHint: "future"})
	if good.Market == nil || good.Market.QuoteAsset != "USDT" || good.Market.ExpiryAtMs != 1795161600000 {
		t.Fatal(good)
	}
	for _, q := range []ResolveRequest{{Exchange: "bybit", Symbol: "BTCUSDT-20NOV26", MarketTypeHint: "perpetual"}, {Exchange: "bybit", Symbol: "BTCUSDT-27nov26", MarketTypeHint: "future"}} {
		v := r.Resolve(q)
		if v.Status != ResolveUnresolved || v.Market != nil {
			t.Fatalf("date token inferred as currency: %+v", v)
		}
	}
	normal := r.Resolve(ResolveRequest{Exchange: "bybit", Symbol: "BTCUSDT", MarketTypeHint: "perpetual"})
	if normal.Market == nil || normal.Market.QuoteAsset != "USDT" {
		t.Fatal(normal)
	}
}
