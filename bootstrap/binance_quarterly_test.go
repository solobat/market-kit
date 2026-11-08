package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestBinanceQuarterlyMetadataNeverBecomesPerpetual(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.binance.com" {
			return jsonResponse(`{"symbols":[]}`), nil
		}
		return jsonResponse(`{"symbols":[{"symbol":"BTCUSDT","baseAsset":"BTC","quoteAsset":"USDT","status":"TRADING","contractType":"PERPETUAL"},{"symbol":"BTCUSDT_261225","baseAsset":"BTC","quoteAsset":"USDT","marginAsset":"USDT","deliveryDate":1798185600000,"status":"TRADING","contractType":"CURRENT_QUARTER"},{"symbol":"BTCUSDT_270326","baseAsset":"BTC","quoteAsset":"USDT","marginAsset":"USDT","deliveryDate":1806048000000,"status":"TRADING","contractType":"NEXT_QUARTER"}]}`), nil
	})}
	rows, err := fetchBinance(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].MarketType != "perp" || rows[1].MarketType != "future" || rows[2].InstrumentMetadata.ExpiryAtMs != 1806048000000 {
		t.Fatal(rows)
	}
	legacy := rows[1]
	legacy.SourceID = "consumer"
	legacy.MarketType = "perp"
	legacy.InstrumentMetadata = nil
	for _, input := range [][]discovery.ImportedMarket{append(append([]discovery.ImportedMarket{}, rows...), legacy), append([]discovery.ImportedMarket{legacy}, rows...)} {
		resolver := identity.NewResolver(curation.MergeGeneratedRegistry(identity.Registry{}, curation.BuildGeneratedRegistry(input), false))
		get := func(s, k string) *identity.MarketIdentity {
			v := resolver.Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: s, MarketTypeHint: k})
			if v.Market == nil {
				t.Fatal(v)
			}
			return v.Market
		}
		perp := get("BTCUSDT", "perpetual")
		future := get("BTCUSDT_261225", "future")
		next := get("BTCUSDT_270326", "future")
		wrong := get("BTCUSDT_261225", "perpetual")
		if future.ExpiryAtMs != 1798185600000 || future.QuoteAsset != "USDT" || future.SettlementAsset != "USDT" || future.ComparisonStatus != identity.ComparisonAmbiguous || future.InstrumentKind != "future" || future.ComparisonKey == perp.ComparisonKey || future.ComparisonKey == next.ComparisonKey {
			t.Fatal(perp, future, next)
		}
		if wrong.ComparisonStatus != identity.ComparisonProhibited || wrong.SettlementAsset != "" || wrong.ExpiryAtMs != 0 {
			t.Fatal(wrong)
		}
	}
}
