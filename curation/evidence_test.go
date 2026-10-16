package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestVenueStockEvidenceBeatsGenericCryptoRegardlessOfOrder(t *testing.T) {
	stock := discovery.ImportedMarket{PlatformID: "bitget", VenueType: "cex", MarketType: "spot", Symbol: "RFRESHUSDT", BaseAsset: "FRESH", QuoteAsset: "USDT", AssetClassHint: "stock", Tags: []string{"tokenized-stock"}}
	weak := stock
	weak.BaseAsset = "rFRESH"
	weak.AssetClassHint = "crypto"
	weak.Tags = []string{"crypto"}
	for _, rows := range [][]discovery.ImportedMarket{{stock, weak}, {weak, stock}} {
		registry := BuildGeneratedRegistry(rows)
		if len(registry.MarketOverrides) != 1 {
			t.Fatal(registry.MarketOverrides)
		}
		r := registry.MarketOverrides[0]
		if r.CanonicalSymbol != "FRESH/USDT" || r.AssetClass != "rwa_stock" || r.AssetID != "equity:ticker:fresh" {
			t.Fatalf("weak source won: %+v", r)
		}
	}
}
func TestEquallyStrongConflictingVenueIdentityFailsClosed(t *testing.T) {
	a := discovery.ImportedMarket{PlatformID: "test", VenueType: "cex", MarketType: "spot", Symbol: "SAMEUSDT", BaseAsset: "ONE", QuoteAsset: "USDT", AssetClassHint: "stock"}
	b := a
	b.BaseAsset = "TWO"
	r := BuildGeneratedRegistry([]discovery.ImportedMarket{a, b}).MarketOverrides[0]
	if r.ComparisonStatus != identity.ComparisonAmbiguous || r.AssetClass != "unknown" {
		t.Fatalf("conflict became comparable: %+v", r)
	}
}

func TestWeakMarketCannotInheritStockIdentityAtKnownTickerCollision(t *testing.T) {
	rows := []discovery.ImportedMarket{
		{PlatformID: "binance", VenueType: "cex", MarketType: "perp", Symbol: "DIAUSDT", BaseAsset: "DIA", QuoteAsset: "USDT", UnderlyingCategory: "coin"},
		{PlatformID: "bitget", VenueType: "cex", MarketType: "spot", Symbol: "RDIAUSDT", BaseAsset: "DIA", QuoteAsset: "USDT", AssetClassHint: "stock"},
		{PlatformID: "binance", VenueType: "cex", MarketType: "spot", Symbol: "DIAUSDT", BaseAsset: "DIA", QuoteAsset: "USDT"},
	}
	r := BuildGeneratedRegistry(rows)
	for _, v := range r.MarketOverrides {
		if v.Exchange == "binance" && v.MarketType == "spot" {
			if v.AssetClass != "unknown" || v.ComparisonStatus != identity.ComparisonAmbiguous {
				t.Fatalf("weak spot became stock: %+v", v)
			}
		}
	}
}

func TestCoinEvidenceCanDisambiguateCommodityTicker(t *testing.T) {
	rows := []discovery.ImportedMarket{
		{PlatformID: "coins", VenueType: "cex", MarketType: "perp", Symbol: "GOLDUSDT", BaseAsset: "GOLD", QuoteAsset: "USDT", UnderlyingCategory: "coin"},
		{PlatformID: "metals", VenueType: "cex", MarketType: "perp", Symbol: "GOLDUSDT", BaseAsset: "GOLD", QuoteAsset: "USDT", AssetClassHint: "commodity"},
	}
	r := BuildGeneratedRegistry(rows)
	if len(r.AssetCollisions) != 1 {
		t.Fatalf("commodity ticker hid a coin collision: %+v", r)
	}
	for _, v := range r.MarketOverrides {
		if v.Exchange == "coins" && v.AssetClass != "crypto" {
			t.Fatal(v)
		}
		if v.Exchange == "metals" && v.AssetClass != "rwa_commodity" {
			t.Fatal(v)
		}
	}
}
