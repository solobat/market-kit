package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestDatedFuturesNeverJoinPerpetualOrAnotherExpiry(t *testing.T) {
	meta := &discovery.InstrumentMetadata{Source: "bybit", ContractType: "LinearFutures", ExpiryAtMs: 1794556800000, SettlementAsset: "USDT"}
	rows := []discovery.ImportedMarket{
		{PlatformID: "bybit", MarketType: "perp", Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", AssetClassHint: "crypto"},
		{PlatformID: "bybit", MarketType: "future", Symbol: "BTCUSDT-13NOV26", BaseAsset: "BTC", QuoteAsset: "USDT", AssetClassHint: "crypto", InstrumentMetadata: meta},
		{PlatformID: "bybit", MarketType: "perp", Symbol: "BTCUSDT-13NOV26", BaseAsset: "BTC", QuoteAsset: "USDT", AssetClassHint: "crypto"},
		{PlatformID: "bybit", MarketType: "future", Symbol: "BTCUSDT-27NOV26", BaseAsset: "BTC", QuoteAsset: "USDT", AssetClassHint: "crypto"},
	}
	for i := range rows {
		rows[i].VenueType = "cex"
	}
	generated := BuildGeneratedRegistry(rows)
	merged := MergeGeneratedRegistry(identity.Registry{}, generated, false)
	resolver := identity.NewResolver(merged)
	get := func(symbol, kind string) *identity.MarketIdentity {
		v := resolver.Resolve(identity.ResolveRequest{Exchange: "bybit", Symbol: symbol, MarketTypeHint: kind})
		if v.Market == nil {
			t.Fatal(v)
		}
		return v.Market
	}
	perp := get("BTCUSDT", "perpetual")
	future := get("BTCUSDT-13NOV26", "future")
	legacy := get("BTCUSDT-13NOV26", "perpetual")
	unknown := get("BTCUSDT-27NOV26", "future")
	if future.ExpiryAtMs != meta.ExpiryAtMs || future.SettlementAsset != "USDT" || future.ComparisonStatus != identity.ComparisonAmbiguous || future.ComparisonKey == perp.ComparisonKey {
		t.Fatal(future, perp)
	}
	if legacy.ComparisonStatus != identity.ComparisonProhibited || legacy.SettlementAsset != "" || legacy.ExpiryAtMs != 0 {
		t.Fatal(legacy)
	}
	if unknown.ComparisonStatus != identity.ComparisonProhibited || unknown.ComparisonKey == future.ComparisonKey || unknown.ExpiryAtMs != 0 || unknown.SettlementAsset != "" {
		t.Fatal(unknown)
	}
}

func TestOfficialFutureEvidenceWinsAndConflictingExpiryFailsClosed(t *testing.T) {
	official := discovery.ImportedMarket{SourceID: "venue", PlatformID: "bybit", VenueType: "cex", MarketType: "future", Symbol: "BTCUSDT-13NOV26", BaseAsset: "BTC", QuoteAsset: "USDT", AssetClassHint: "crypto", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "bybit", ContractType: "LinearFutures", ExpiryAtMs: 1794556800000, SettlementAsset: "USDT"}}
	weak := official
	weak.SourceID = "downstream"
	weak.InstrumentMetadata = nil
	for _, rows := range [][]discovery.ImportedMarket{{official, weak}, {weak, official}} {
		registry := BuildGeneratedRegistry(rows)
		if len(registry.MarketOverrides) != 1 || registry.MarketOverrides[0].ExpiryAtMs != 1794556800000 {
			t.Fatal(registry.MarketOverrides)
		}
	}
	conflict := official
	conflict.SourceID = "conflicting-snapshot"
	conflict.InstrumentMetadata = &discovery.InstrumentMetadata{Source: "bybit", ContractType: "LinearFutures", ExpiryAtMs: 1795766400000, SettlementAsset: "USDT"}
	registry := MergeGeneratedRegistry(identity.Registry{}, BuildGeneratedRegistry([]discovery.ImportedMarket{official, conflict}), false)
	v := identity.NewResolver(registry).Resolve(identity.ResolveRequest{Exchange: "bybit", Symbol: official.Symbol, MarketTypeHint: "future"})
	if v.Market == nil || v.Market.ComparisonStatus != identity.ComparisonProhibited || v.Market.ExpiryAtMs != 0 || v.Market.SettlementAsset != "" {
		t.Fatal(v)
	}
}

func TestRetainedLegacyDatedOverrideFailsClosedWithoutCurrentCatalogRow(t *testing.T) {
	old := identity.Registry{MarketOverrides: []identity.MarketOverride{{Exchange: "bybit", RawSymbol: "BTCUSDT-13NOV26", MarketType: "perpetual", CanonicalSymbol: "BTC/USDT", AssetClass: "crypto", AssetID: "crypto:ticker:btc", UnderlyingID: "crypto:ticker:btc", ComparisonKey: "crypto:ticker:btc/USDT", ComparisonStatus: identity.ComparisonEligible}}}
	merged := MergeGeneratedRegistry(old, identity.Registry{}, false)
	got := identity.NewResolver(merged).Resolve(identity.ResolveRequest{Exchange: "bybit", Symbol: "BTCUSDT-13NOV26", MarketTypeHint: "perpetual"})
	if got.Market == nil || got.Market.ComparisonStatus != identity.ComparisonProhibited || got.Market.ComparisonKey == "crypto:ticker:btc/USDT" || got.Market.SettlementAsset != "" {
		t.Fatal(got)
	}
}
