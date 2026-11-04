package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestGTradeForexNeverBorrowsCryptoOrStableIdentity(t *testing.T) {
	for _, pair := range [][2]string{{"EUR", "USD"}, {"USD", "JPY"}, {"GBP", "CHF"}} {
		official := discovery.ImportedMarket{SourceID: "market-kit-bootstrap", PlatformID: "gtrade", VenueType: "dex", MarketType: "perp", Symbol: pair[0] + "/" + pair[1], BaseAsset: pair[0], QuoteAsset: pair[1], Category: "forex-minor"}
		generic := official
		generic.SourceID = "slipstream"
		generic.Category = "crypto"
		generic.AssetClass = "crypto"
		for _, rows := range [][]discovery.ImportedMarket{{official, generic}, {generic, official}} {
			reg := BuildGeneratedRegistry(rows)
			got := identity.NewResolver(reg).Resolve(identity.ResolveRequest{Exchange: "gtrade", Symbol: official.Symbol, MarketTypeHint: "perpetual"}).Market
			if got == nil || got.AssetClass != "unknown" || got.ComparisonStatus != identity.ComparisonProhibited || got.InstrumentKind != "fx_cfd" || got.ContractType != "multi_collateral_cfd" || got.SettlementAsset != "" || got.AssetID == identity.CanonicalAssetID("crypto", pair[0]) || got.AssetID == identity.CanonicalAssetID("fiat_stable", pair[0]) {
				t.Fatalf("FX misidentified: %+v", got)
			}
			if len(reg.AssetAliases) != 0 {
				t.Fatal("FX emitted global token aliases", reg.AssetAliases)
			}
		}
	}
}

func TestExtendedFXRetainsReferenceAndCollateralBoundaries(t *testing.T) {
	reg := BuildGeneratedRegistry([]discovery.ImportedMarket{{PlatformID: "extended", VenueType: "dex", MarketType: "perp", Symbol: "USDJPY-USD", BaseAsset: "USDJPY", QuoteAsset: "USD", Category: "FX"}})
	got := identity.NewResolver(reg).Resolve(identity.ResolveRequest{Exchange: "extended", Symbol: "USDJPY-USD", MarketTypeHint: "perpetual"}).Market
	if got == nil || got.AssetClass != "unknown" || got.ComparisonStatus != identity.ComparisonProhibited || got.InstrumentKind != "fx_perpetual" || got.ContractType != "perpetual" || got.SettlementAsset != "USD" || got.UnderlyingID != "fx:extended:usdjpy-usd" {
		t.Fatal(got)
	}
}

func TestBybitNativeForexOutranksGenericStock(t *testing.T) {
	official := discovery.ImportedMarket{SourceID: "native", PlatformID: "bybit", VenueType: "cex", MarketType: "perpetual", Symbol: "USDBRLUSDT", BaseAsset: "USDBRL", QuoteAsset: "USDT", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "bybit", SymbolType: "forex", ContractType: "LinearPerpetual", SettlementAsset: "USDT"}}
	generic := official
	generic.SourceID = "consumer"
	generic.InstrumentMetadata = nil
	generic.AssetClassHint = "stock"
	for _, rows := range [][]discovery.ImportedMarket{{official, generic}, {generic, official}} {
		reg := BuildGeneratedRegistry(rows)
		got := identity.NewResolver(reg).Resolve(identity.ResolveRequest{Exchange: "bybit", Symbol: "USDBRLUSDT", MarketTypeHint: "perpetual"}).Market
		if got == nil || got.AssetClass != "unknown" || got.InstrumentKind != "fx_perpetual" || got.ComparisonStatus != identity.ComparisonProhibited || got.SettlementAsset != "USDT" {
			t.Fatalf("FX identity: %+v", got)
		}
		if len(reg.AssetAliases) != 0 {
			t.Fatal("FX emitted global aliases", reg.AssetAliases)
		}
	}
	for _, mutate := range []func(*discovery.ImportedMarket){func(r *discovery.ImportedMarket) { r.InstrumentMetadata.Source = "consumer" }, func(r *discovery.ImportedMarket) { r.InstrumentMetadata.ContractType = "LinearFutures" }, func(r *discovery.ImportedMarket) { r.MarketType = "spot" }} {
		row := official
		meta := *official.InstrumentMetadata
		row.InstrumentMetadata = &meta
		mutate(&row)
		if isForexMarket(row) {
			t.Fatal("unverified metadata treated as native FX", row)
		}
	}
}
