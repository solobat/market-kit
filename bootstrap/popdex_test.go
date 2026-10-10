package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestPopdexDiscoveryPreservesCommodityAndETF(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != popdexSymbolsURL {
			t.Fatal(r.URL)
		}
		return jsonResponse(`{"code":"200","data":[{"symbol":"BZUSDT","baseToken":"BZ","quoteToken":"USDT","category":"Futures","assetClass":"Rwa","status":"Trading"},{"symbol":"QQQUSDT","baseToken":"QQQ","quoteToken":"USDT","category":"Futures","assetClass":"Rwa","status":"PreTrading"},{"symbol":"BTCUSDT","baseToken":"BTC","quoteToken":"USDT","category":"Spot","assetClass":"Crypto","status":"Trading"}]}`), nil
	})}
	env, err := Fetch(context.Background(), client, []string{"popdex"})
	if err != nil {
		t.Fatal(err)
	}
	if len(env.Items) != 2 {
		t.Fatalf("bad inventory %+v", env.Items)
	}
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	rows := discovery.NewAggregator(registry).NormalizeImportedMarkets(env.Items)
	for _, row := range rows {
		if row.Exchange != "popdex" || row.MarketType != identity.MarketTypePerpetual {
			t.Fatalf("bad venue/type %+v", row)
		}
		if row.BaseAsset == "BZ" && row.AssetClass != "rwa_commodity" {
			t.Fatalf("Brent misclassified %+v", row)
		}
		if row.BaseAsset == "QQQ" && (row.AssetClass != "rwa_stock" || row.Status != "paused") {
			t.Fatalf("ETF/status lost %+v", row)
		}
	}
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { return jsonResponse(`{"code":"429","data":[]}`), nil })
	if _, err = Fetch(context.Background(), client, []string{"popdex"}); err == nil {
		t.Fatal("accepted API error")
	}
}
