package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestBinanceNativeExpiryAndCollateralConflictFailsClosed(t *testing.T) {
	a := discovery.ImportedMarket{SourceID: "a", PlatformID: "binance", VenueType: "cex", MarketType: "future", Symbol: "BTCUSDT_261225", BaseAsset: "BTC", QuoteAsset: "USDT", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "binance", SourceURL: "https://fapi.binance.com/fapi/v1/exchangeInfo", ContractType: "CURRENT_QUARTER", SettlementAsset: "USDT", ExpiryAtMs: 1798185600000}}
	b := a
	m := *a.InstrumentMetadata
	b.InstrumentMetadata = &m
	b.InstrumentMetadata.ExpiryAtMs++
	b.SourceID = "b"
	for _, input := range [][]discovery.ImportedMarket{{a, b}, {b, a}} {
		got := identity.NewResolver(BuildGeneratedRegistry(input)).Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: a.Symbol, MarketTypeHint: "future"}).Market
		if got == nil || got.ComparisonStatus != identity.ComparisonProhibited || got.ExpiryAtMs != 0 || got.SettlementAsset != "" {
			t.Fatal(got)
		}
	}
	existing := identity.Registry{MarketOverrides: []identity.MarketOverride{{Exchange: "binance", RawSymbol: a.Symbol, MarketType: "perpetual", CanonicalSymbol: "BTC/USDT", AssetClass: "crypto", AssetID: "crypto:ticker:btc", ComparisonKey: "crypto:ticker:btc/USDT", ComparisonStatus: identity.ComparisonEligible}}}
	merged := MergeGeneratedRegistry(existing, identity.Registry{}, false)
	got := identity.NewResolver(merged).Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: a.Symbol, MarketTypeHint: "perpetual"}).Market
	if got == nil || got.ComparisonStatus != identity.ComparisonProhibited {
		t.Fatal(got)
	}
}

func TestCoinMQuarterEnumDoesNotProveLinearFuture(t *testing.T) {
	row := discovery.ImportedMarket{PlatformID: "binance", VenueType: "cex", MarketType: "future", Symbol: "BTCUSD_261225", BaseAsset: "BTC", QuoteAsset: "USD", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "binance", SourceURL: "https://dapi.binance.com/dapi/v1/exchangeInfo", ContractType: "CURRENT_QUARTER", SettlementAsset: "BTC", ExpiryAtMs: 1798185600000}}
	got := identity.NewResolver(BuildGeneratedRegistry([]discovery.ImportedMarket{row})).Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: row.Symbol, MarketTypeHint: "future"}).Market
	if got == nil || got.ComparisonStatus != identity.ComparisonProhibited || got.ContractType != "dated_future_unverified" || got.SettlementAsset != "" {
		t.Fatal(got)
	}
}
