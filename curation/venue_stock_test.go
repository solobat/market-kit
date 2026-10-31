package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestNativeStocksNeverJoinOnUnverifiedTickerUnits(t *testing.T) {
	rows := []discovery.ImportedMarket{
		{PlatformID: "gate", VenueType: "cex", MarketType: "perpetual", Symbol: "EWZS_USDT", BaseAsset: "EWZS", QuoteAsset: "USDT", Category: "stocks"},
		{PlatformID: "bybit", VenueType: "cex", MarketType: "perpetual", Symbol: "EWZSUSDT", BaseAsset: "EWZS", QuoteAsset: "USDT", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "bybit", ContractType: "LinearPerpetual", SymbolType: "ETF"}},
	}
	reg := BuildGeneratedRegistry(rows)
	if len(reg.MarketOverrides) != 2 {
		t.Fatal(reg.MarketOverrides)
	}
	for _, m := range reg.MarketOverrides {
		if m.AssetClass != "rwa_stock" || m.ComparisonStatus != identity.ComparisonAmbiguous || m.AssetID == "equity:ticker:ewzs" {
			t.Fatal(m)
		}
	}
	if reg.MarketOverrides[0].ComparisonKey == reg.MarketOverrides[1].ComparisonKey {
		t.Fatal("unverified venue stock units merged")
	}
	rows[1].InstrumentMetadata.Source = "untrusted"
	if isVenueStockPerpetual(rows[1]) {
		t.Fatal("untrusted metadata accepted")
	}
	rows[1].InstrumentMetadata.Source = "bybit"
	rows[1].InstrumentMetadata.ContractType = "LinearFutures"
	if isVenueStockPerpetual(rows[1]) {
		t.Fatal("dated future treated as perpetual")
	}
}
