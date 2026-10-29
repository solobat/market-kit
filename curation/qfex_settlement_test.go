package curation

import (
	"github.com/solobat/market-kit/discovery"
	"testing"
)

func TestQFEXReferenceQuoteDoesNotReplaceOfficialMargin(t *testing.T) {
	for _, tc := range []struct{ source, margin, want string }{{"qfex", "USDC", "USDC"}, {"qfex", "", "USD"}, {"other", "USDC", "USD"}} {
		reg := BuildGeneratedRegistry([]discovery.ImportedMarket{{PlatformID: "qfex", VenueType: "cex", MarketType: "perpetual", Symbol: "FIXTURE-USD", BaseAsset: "FIXTURE", QuoteAsset: "USD", AssetClassHint: "stock", InstrumentMetadata: &discovery.InstrumentMetadata{Source: tc.source, SettlementAsset: tc.margin}}})
		if len(reg.MarketOverrides) != 1 || reg.MarketOverrides[0].SettlementAsset != tc.want {
			t.Fatalf("source=%s margin=%s: %+v", tc.source, tc.margin, reg.MarketOverrides)
		}
	}
}
