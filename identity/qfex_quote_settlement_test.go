package identity

import "testing"

func TestQFEXYubicoPreservesNativeQuoteAndMarginCurrency(t *testing.T) {
	registry, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	result := NewResolver(registry).Resolve(ResolveRequest{Exchange: "qfex", Symbol: "YUBICO-SEK", MarketTypeHint: "perpetual"})
	m := result.Market
	if m == nil || m.AssetClass != "rwa_stock" || m.QuoteAsset != "SEK" || m.SettlementAsset != "USDC" || m.ComparisonStatus != ComparisonAmbiguous || m.ComparisonKey != "equity:product:qfex:yubico-sek/SEK" {
		t.Fatalf("native quote or product boundary lost: %+v", result)
	}
}
