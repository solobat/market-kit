package server

import "testing"

func TestKnownClassRequestDoesNotGuessContractSettlementFromQuote(t *testing.T) {
	for _, test := range []struct{ contract, explicit, want string }{
		{"multi_collateral_cfd", "", ""},
		{" MULTI_COLLATERAL_CFD ", "", ""},
		{"linear_futures", "", ""},
		{"inverse_futures", "", ""},
		{"dated_future_unverified", "", ""},
		{"multi_collateral_cfd", "USDC", "USDC"},
		{"perpetual", "", "USD"},
	} {
		got, _, err := normalizeRegistryOverrideRequest(registryOverrideRequest{Exchange: "gtrade", RawSymbol: "DAR/USD", MarketType: "perpetual", CanonicalSymbol: "DAR/USD", AssetClass: "crypto", ContractType: test.contract, SettlementAsset: test.explicit, ComparisonStatus: "ambiguous"})
		if err != nil || got.SettlementAsset != test.want || got.AssetClass != "crypto" {
			t.Fatalf("%+v -> %+v, %v", test, got, err)
		}
	}
}
