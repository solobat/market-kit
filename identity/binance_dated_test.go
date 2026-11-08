package identity

import "testing"

func TestUnregisteredBinanceDateCannotBecomeQuoteCurrency(t *testing.T) {
	r := NewResolver(Registry{})
	for _, kind := range []string{"perpetual", "future", "spot"} {
		v := r.Resolve(ResolveRequest{Exchange: "binance", Symbol: "BTCUSDT_261225", MarketTypeHint: kind})
		if v.Status != ResolveUnresolved || v.Market != nil {
			t.Fatal(kind, v)
		}
	}
	if binanceDatedSymbol("BTCUSDT_26X225") || binanceDatedSymbol("1000SATSUSDT") {
		t.Fatal("non-native syntax quarantined")
	}
}
