package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestGateNativeExclusionsSurviveMissingLegacyPairFields(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v4/spot/currency_pairs" {
			return jsonResponse(`[{"id":"ACE_USDT","base":"ACE","quote":"USDT","trade_status":"untradable"},{"id":"BTC_USDT","base":"BTC","quote":"USDT","trade_status":"tradable"}]`), nil
		}
		return jsonResponse(`[{"name":"ACE_USDT","type":"direct","quanto_multiplier":"0.1","in_delisting":true}]`), nil
	})}
	rows, err := fetchGate(context.Background(), client)
	if err != nil || len(rows) != 3 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if rows[2].BaseAsset != "ACE" || rows[2].QuoteAsset != "USDT" || rows[2].AssetClassHint != "" {
		t.Fatalf("pair recovery invented a class: %+v", rows[2])
	}
	if rows[1].InstrumentMetadata != nil {
		t.Fatal("ordinary tradable spot changed")
	}
	r := identity.NewResolver(curation.BuildGeneratedRegistry(rows))
	for _, mt := range []string{"spot", "perpetual"} {
		got := r.Resolve(identity.ResolveRequest{Exchange: "gate", Symbol: "ACE_USDT", MarketTypeHint: mt})
		if got.Market == nil || got.Market.ComparisonStatus != identity.ComparisonProhibited || got.Market.SettlementAsset != "USDT" || got.Market.UnitMultiplier == 0.1 {
			t.Fatalf("exclusion/contract lot boundary lost: %+v", got)
		}
	}
}
