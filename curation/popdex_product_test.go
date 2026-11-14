package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestPopdexNativeCollateralConflictNeverGuessesQuote(t *testing.T) {
	a := discovery.ImportedMarket{PlatformID: "popdex", VenueType: "dex", MarketType: "perp", Symbol: "NEWUSDT", BaseAsset: "NEW", QuoteAsset: "USDT", AssetClassHint: "stock", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "popdex", SourceURL: "https://api.popdex.xyz/api/v1/config/symbols?category=Futures", ContractType: "perpetual", NativeTradingStatus: "PreTrading", SettlementAsset: "USDT", IsRWA: true, SymbolType: "unclassified_rwa"}}
	b := a
	m := *a.InstrumentMetadata
	m.SettlementAsset = "USDC"
	b.InstrumentMetadata = &m
	for _, rows := range [][]discovery.ImportedMarket{{a, b}, {b, a}} {
		r := identity.NewResolver(BuildGeneratedRegistry(rows)).Resolve(identity.ResolveRequest{Exchange: "popdex", Symbol: "NEWUSDT", MarketTypeHint: "perpetual"})
		if r.Market == nil || r.Market.ComparisonStatus != identity.ComparisonProhibited || r.Market.SettlementAsset != "" {
			t.Fatalf("%+v", r.Market)
		}
	}
	wrong := *a.InstrumentMetadata
	wrong.SourceURL = "https://example.com"
	if popdexPhase("popdex", &wrong) != "" || popdexUnclassified("popdex", &wrong) {
		t.Fatal("accepted non-native proof")
	}
}
