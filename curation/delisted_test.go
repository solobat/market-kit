package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestOfficialDelistingWinsOverGenericLiveMetadata(t *testing.T) {
	official := discovery.ImportedMarket{PlatformID: "hyperliquid", VenueType: "dex", MarketType: "perp", Symbol: "io:SKHX", BaseAsset: "SKHX", QuoteAsset: "USDC", Status: "delisted", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "hyperliquid", IsDelisted: true}}
	weak := official
	weak.SourceID = "downstream"
	weak.AssetClassHint = "stock"
	weak.Status = "live"
	weak.InstrumentMetadata = nil
	for _, rows := range [][]discovery.ImportedMarket{{official, weak}, {weak, official}} {
		reg := MergeGeneratedRegistry(identity.Registry{}, BuildGeneratedRegistry(rows), false)
		for _, alias := range reg.AssetAliases {
			if alias.Canonical == "SKHX" {
				t.Fatalf("inactive metadata must not create a global issuer alias: %+v", alias)
			}
		}
		got := identity.NewResolver(reg).Resolve(identity.ResolveRequest{Exchange: "hyperliquid", Symbol: "io:SKHX", MarketTypeHint: "perpetual"})
		if got.Market == nil || got.Market.ComparisonStatus != identity.ComparisonProhibited || got.Market.ContractType != "delisted_perpetual" || got.Market.ComparisonKey == "equity:ticker:skhx/USDC" {
			t.Fatal(got)
		}
	}
	live := official
	live.Symbol = "BTC"
	live.BaseAsset = "BTC"
	live.Status = "live"
	live.AssetClassHint = "crypto"
	live.InstrumentMetadata = &discovery.InstrumentMetadata{Source: "hyperliquid"}
	reg := BuildGeneratedRegistry([]discovery.ImportedMarket{live})
	if len(reg.MarketOverrides) != 1 || reg.MarketOverrides[0].ComparisonStatus != identity.ComparisonEligible {
		t.Fatal(reg.MarketOverrides)
	}
}
