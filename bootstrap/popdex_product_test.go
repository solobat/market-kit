package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestPopdexUnknownRWAAndPrelaunchDoNotBreakCatalogOrBecomeEligible(t *testing.T) {
	c := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"code":"200","data":[{"symbol":"FRESHUSDT","baseToken":"FRESH","quoteToken":"USDT","settlementToken":"USDT","category":"Futures","assetClass":"Rwa","status":"PreTrading"},{"symbol":"OTHERUSDT","baseToken":"OTHER","quoteToken":"USDT","settlementToken":"USDT","category":"Futures","assetClass":"Rwa","status":"Trading"},{"symbol":"BTCUSDT","baseToken":"BTC","quoteToken":"USDT","settlementToken":"USDT","category":"Futures","assetClass":"Crypto","status":"Trading"}]}`), nil
	})}
	rows, e := fetchPopdex(context.Background(), c)
	if e != nil || len(rows) != 3 {
		t.Fatal(e, rows)
	}
	weak := rows[0]
	weak.InstrumentMetadata = nil
	weak.SourceID = "consumer"
	weak.AssetClassHint = "stock"
	weak.Status = "live"
	for _, input := range [][]discovery.ImportedMarket{append(append([]discovery.ImportedMarket{}, rows...), weak), append([]discovery.ImportedMarket{weak}, rows...)} {
		resolver := identity.NewResolver(curation.BuildGeneratedRegistry(input))
		for _, s := range []string{"FRESHUSDT", "OTHERUSDT", "BTCUSDT"} {
			got := resolver.Resolve(identity.ResolveRequest{Exchange: "popdex", Symbol: s, MarketTypeHint: "perpetual"})
			if got.Market == nil {
				t.Fatal(got)
			}
			m := got.Market
			if s == "FRESHUSDT" && (m.ComparisonStatus != identity.ComparisonProhibited || m.AssetClass != "unknown" || m.SettlementAsset != "USDT") {
				t.Fatalf("prelaunch %+v", m)
			}
			if s == "OTHERUSDT" && (m.ComparisonStatus != identity.ComparisonAmbiguous || m.AssetClass != "unknown") {
				t.Fatalf("unknown RWA %+v", m)
			}
			if s == "BTCUSDT" && m.ComparisonStatus != identity.ComparisonEligible {
				t.Fatalf("live crypto %+v", m)
			}
		}
	}
}
