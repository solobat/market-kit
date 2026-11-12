package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestBinanceNativeNonTradingStateSurvivesWeakLiveSource(t *testing.T) {
	for _, status := range []string{"PENDING_TRADING", "PRE_TRADING", "SETTLING"} {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "api.binance.com" {
				return jsonResponse(`{"symbols":[]}`), nil
			}
			return jsonResponse(`{"symbols":[{"symbol":"NEWUSDC","baseAsset":"NEW","quoteAsset":"USDC","marginAsset":"USDC","status":"` + status + `","contractType":"PERPETUAL","underlyingType":"COIN","deliveryDate":4133404800000},{"symbol":"LIVEUSDT","baseAsset":"LIVE","quoteAsset":"USDT","marginAsset":"USDT","status":"TRADING","contractType":"PERPETUAL","underlyingType":"COIN"}]}`), nil
		})}
		rows, err := fetchBinance(context.Background(), client)
		if err != nil {
			t.Fatal(err)
		}
		if rows[0].InstrumentMetadata.NativeTradingStatus != status || rows[0].InstrumentMetadata.SettlementAsset != "USDC" || rows[0].InstrumentMetadata.ExpiryAtMs != 0 {
			t.Fatal(rows[0])
		}
		weak := rows[0]
		weak.SourceID = "consumer"
		weak.Status = "live"
		weak.InstrumentMetadata = nil
		for _, input := range [][]discovery.ImportedMarket{{rows[0], weak, rows[1]}, {weak, rows[0], rows[1]}} {
			reg := curation.BuildGeneratedRegistry(input)
			resolver := identity.NewResolver(reg)
			got := resolver.Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: "NEWUSDC", MarketTypeHint: "perpetual"})
			if got.Market == nil || got.Market.ComparisonStatus != identity.ComparisonProhibited || got.Market.SettlementAsset != "USDC" || got.Market.ExpiryAtMs != 0 || got.Market.AssetID == "crypto:ticker:new" {
				t.Fatal(got)
			}
			live := resolver.Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: "LIVEUSDT", MarketTypeHint: "perpetual"})
			if live.Market == nil || live.Market.ComparisonStatus != identity.ComparisonEligible {
				t.Fatal(live)
			}
		}
	}
}
