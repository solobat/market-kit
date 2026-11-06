package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestBitgetRWAFlagAndCollateralReachGeneratedIdentity(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v3/market/instruments" {
			return jsonResponse(`{"data":[]}`), nil
		}
		return jsonResponse(`{"data":[{"symbol":"LTUSDT","baseCoin":"LT","quoteCoin":"USDT","isRwa":"YES","symbolType":"perpetual","supportMarginCoins":["USDT"]},{"symbol":"MULTIUSDT","baseCoin":"MULTI","quoteCoin":"USDT","isRwa":"YES","symbolType":"perpetual","supportMarginCoins":["USDT","USDC"]}]}`), nil
	})}
	rows, err := fetchBitget(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].InstrumentMetadata == nil || !rows[0].InstrumentMetadata.IsRWA {
		t.Fatal(rows)
	}
	resolver := identity.NewResolver(curation.BuildGeneratedRegistry(rows))
	for _, sym := range []string{"LTUSDT", "MULTIUSDT"} {
		got := resolver.Resolve(identity.ResolveRequest{Exchange: "bitget", Symbol: sym, MarketTypeHint: "perpetual"}).Market
		if got == nil || got.AssetClass != "unknown" || got.ComparisonStatus != identity.ComparisonAmbiguous {
			t.Fatal(got)
		}
		if sym == "MULTIUSDT" && got.SettlementAsset != "" {
			t.Fatal("multi-collateral inferred settlement", got)
		}
	}
}
