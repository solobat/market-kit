package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestGateOfficialContractTypesPreserveProductIdentity(t *testing.T) {
	c := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v4/spot/currency_pairs" {
			return jsonResponse(`[]`), nil
		}
		return jsonResponse(`[{"name":"BOTZ_USDT","contract_type":"stocks","quanto_multiplier":"0.1"},{"name":"USDC_USDT","contract_type":"forex"},{"name":"BVOL_USDT","contract_type":"indices"},{"name":"IAU_USDT","contract_type":"metals"},{"name":"XCU_USDT","contract_type":"metals"},{"name":"NEW_USDT","contract_type":""}]`), nil
	})}
	rows, err := fetchGate(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatal(rows)
	}
	if rows[0].BaseAsset != "BOTZ" || rows[0].QuoteAsset != "USDT" || rows[0].AssetClassHint != "stock" || rows[3].AssetClassHint != "stock" || rows[4].AssetClassHint != "commodity" || rows[5].AssetClassHint != "" {
		t.Fatal(rows)
	}
	reg := curation.BuildGeneratedRegistry(rows)
	resolver := identity.NewResolver(reg)
	for _, q := range []struct{ symbol, class, kind string }{{"BOTZ_USDT", "rwa_stock", "equity_perpetual"}, {"USDC_USDT", "unknown", "fx_perpetual"}, {"BVOL_USDT", "unknown", "index_perpetual"}, {"IAU_USDT", "rwa_stock", "equity_perpetual"}, {"XCU_USDT", "rwa_commodity", "commodity_perpetual"}} {
		got := resolver.Resolve(identity.ResolveRequest{Exchange: "gate", Symbol: q.symbol, MarketTypeHint: "perpetual"}).Market
		if got == nil || got.AssetClass != q.class || got.InstrumentKind != q.kind || got.UnitMultiplier != 0 {
			t.Fatalf("contract type/lot confused: %+v", got)
		}
		if q.symbol == "USDC_USDT" && (got.ComparisonStatus != identity.ComparisonProhibited || got.AssetID == "fiat:ticker:usdc") {
			t.Fatal(got)
		}
	}
}
