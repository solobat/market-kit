package identity

import "testing"

func TestMultiCollateralNeverInfersSettlementFromQuote(t *testing.T) {
	for _, settlement := range []string{"", "USDC"} {
		r := NewResolver(Registry{MarketOverrides: []MarketOverride{{Exchange: "gtrade", RawSymbol: "USD/JPY", MarketType: "perpetual", CanonicalSymbol: "USD/JPY", AssetClass: "unknown", AssetID: "fx:gtrade:usd-jpy", UnderlyingID: "fx:usd-jpy", ComparisonStatus: ComparisonProhibited, ContractType: "multi_collateral_cfd", InstrumentKind: "fx_cfd", SettlementAsset: settlement}}})
		got := r.Resolve(ResolveRequest{Exchange: "gtrade", Symbol: "USD/JPY", MarketTypeHint: "perpetual"}).Market
		if got == nil || got.SettlementAsset != settlement {
			t.Fatal("quote invented as collateral", got)
		}
	}
}
