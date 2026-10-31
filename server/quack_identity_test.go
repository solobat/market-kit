package server

import (
	"testing"

	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

func TestReviewedQuackIdentitySurvivesWeakSyncAndStockCollision(t *testing.T) {
	app, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	rows := []discovery.ImportedMarket{
		{PlatformID: "variational", VenueType: "dex", MarketType: "perp", Symbol: "Q", BaseAsset: "Q", QuoteAsset: "USDC"},
		{PlatformID: "bybit", VenueType: "cex", MarketType: "perp", Symbol: "QUSDT", BaseAsset: "Q", QuoteAsset: "USDT"},
		{PlatformID: "binance", VenueType: "cex", MarketType: "perp", Symbol: "QUSDT", BaseAsset: "Q", QuoteAsset: "USDT", UnderlyingCategory: "coin"},
		{PlatformID: "bitget", VenueType: "cex", MarketType: "spot", Symbol: "RQUSDT", BaseAsset: "Q", QuoteAsset: "USDT", AssetClassHint: "stock"},
	}
	for i := 0; i < 2; i++ {
		generated := curation.MergeGeneratedRegistry(app.generatedRegistrySnapshot(), curation.BuildGeneratedRegistry(rows), false)
		app.applyGeneratedRegistry(generated)
		var underlying string
		for _, req := range []identity.ResolveRequest{
			{Exchange: "variational", Symbol: "Q", MarketTypeHint: "perpetual"},
			{Exchange: "variational", Symbol: "Q/USDC", MarketTypeHint: "perpetual"},
			{Exchange: "bybit", Symbol: "QUSDT", MarketTypeHint: "perpetual"},
		} {
			got := app.runtimeResolver().Resolve(req)
			if got.Market == nil || got.Market.AssetClass != "crypto" || got.Market.UnderlyingID != "crypto:ticker:q" || got.Market.ComparisonStatus != identity.ComparisonEligible {
				t.Fatalf("sync %d: %+v", i, got)
			}
			underlying = got.Market.UnderlyingID
		}
		stock := app.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: "bitget", Symbol: "RQUSDT", MarketTypeHint: "spot"})
		if stock.Market == nil || stock.Market.AssetClass != "rwa_stock" || stock.Market.UnderlyingID == underlying {
			t.Fatal(stock)
		}
		unknown := app.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: "unverified", Symbol: "QUSDT", MarketTypeHint: "perpetual"})
		if unknown.Status == identity.ResolveResolved {
			t.Fatalf("unverified ticker joined Q: %+v", unknown)
		}
	}
}
