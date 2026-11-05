package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestRealityWrapperKeepsProductBoundary(t *testing.T) {
	native := discovery.ImportedMarket{SourceID: "native", PlatformID: "bitget", VenueType: "cex", MarketType: "spot", Symbol: "RCATUSDT", BaseAsset: "CAT", QuoteAsset: "USDT", AssetClassHint: "stock", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "bitget", SymbolType: "stock", IsReality: true}}
	generic := native
	generic.SourceID = "consumer"
	generic.InstrumentMetadata = nil
	for _, rows := range [][]discovery.ImportedMarket{{native, generic}, {generic, native}} {
		got := identity.NewResolver(BuildGeneratedRegistry(rows)).Resolve(identity.ResolveRequest{Exchange: "bitget", Symbol: "RCATUSDT", MarketTypeHint: "spot"}).Market
		if got == nil || got.AssetID != "equity:bitget:reality:rcatusdt" || got.UnderlyingID != got.AssetID || got.ComparisonStatus != identity.ComparisonAmbiguous || got.InstrumentKind != "tokenized_equity" || got.ContractType != "reality_stock_token" || got.SettlementAsset != "USDT" {
			t.Fatalf("wrapper identity: %+v", got)
		}
	}
	for _, mutate := range []func(*discovery.ImportedMarket){func(r *discovery.ImportedMarket) { r.InstrumentMetadata.Source = "consumer" }, func(r *discovery.ImportedMarket) { r.InstrumentMetadata.IsReality = false }, func(r *discovery.ImportedMarket) { r.MarketType = "perpetual" }, func(r *discovery.ImportedMarket) { r.InstrumentMetadata.SymbolType = "crypto" }} {
		row := native
		m := *native.InstrumentMetadata
		row.InstrumentMetadata = &m
		mutate(&row)
		if isBitgetRealitySpot(row) {
			t.Fatal("unsupported wrapper evidence", row)
		}
	}
}
