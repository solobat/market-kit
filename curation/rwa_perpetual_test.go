package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestNativeBitgetRWARequiresProductReview(t *testing.T) {
	native := discovery.ImportedMarket{SourceID: "native", PlatformID: "bitget", VenueType: "cex", MarketType: "perp", Symbol: "LTUSDT", BaseAsset: "LT", QuoteAsset: "USDT", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "bitget", ContractType: "perpetual", IsRWA: true, SettlementAsset: "USDT"}}
	generic := native
	generic.SourceID = "consumer"
	generic.InstrumentMetadata = nil
	generic.AssetClassHint = "crypto"
	for _, rows := range [][]discovery.ImportedMarket{{native, generic}, {generic, native}} {
		reg := BuildGeneratedRegistry(rows)
		got := identity.NewResolver(reg).Resolve(identity.ResolveRequest{Exchange: "bitget", Symbol: "LTUSDT", MarketTypeHint: "perpetual"}).Market
		if got == nil || got.AssetClass != "unknown" || got.AssetID != "rwa:bitget:perpetual:ltusdt" || got.ComparisonStatus != identity.ComparisonAmbiguous || got.SettlementAsset != "USDT" {
			t.Fatalf("RWA boundary: %+v", got)
		}
		if len(reg.AssetAliases) != 0 {
			t.Fatal("RWA emitted global token identity", reg.AssetAliases)
		}
	}
	for _, mutate := range []func(*discovery.ImportedMarket){func(r *discovery.ImportedMarket) { r.InstrumentMetadata.Source = "consumer" }, func(r *discovery.ImportedMarket) { r.InstrumentMetadata.IsRWA = false }, func(r *discovery.ImportedMarket) { r.MarketType = "spot" }, func(r *discovery.ImportedMarket) { r.InstrumentMetadata.ContractType = "future" }} {
		row := native
		m := *native.InstrumentMetadata
		row.InstrumentMetadata = &m
		mutate(&row)
		if isBitgetRWAPerpetual(row) {
			t.Fatal("unsupported RWA evidence", row)
		}
	}
}

func TestConflictingRWACollateralFailsClosed(t *testing.T) {
	a := discovery.ImportedMarket{SourceID: "a", PlatformID: "bitget", VenueType: "cex", MarketType: "perp", Symbol: "LTUSDT", BaseAsset: "LT", QuoteAsset: "USDT", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "bitget", ContractType: "perpetual", IsRWA: true, SettlementAsset: "USDT"}}
	b := a
	m := *a.InstrumentMetadata
	b.InstrumentMetadata = &m
	b.InstrumentMetadata.SettlementAsset = "USDC"
	b.SourceID = "b"
	for _, rows := range [][]discovery.ImportedMarket{{a, b}, {b, a}} {
		got := identity.NewResolver(BuildGeneratedRegistry(rows)).Resolve(identity.ResolveRequest{Exchange: "bitget", Symbol: "LTUSDT", MarketTypeHint: "perpetual"}).Market
		if got == nil || got.ComparisonStatus != identity.ComparisonAmbiguous || got.SettlementAsset != "" || got.ContractType != "rwa_perpetual_unverified" {
			t.Fatal(got)
		}
	}
}
