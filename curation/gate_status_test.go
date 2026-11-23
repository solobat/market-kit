package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestGateExclusionRequiresMatchingNativeEndpoint(t *testing.T) {
	for _, url := range []string{"https://example.com", "https://api.gateio.ws/api/v4/spot/currency_pairs"} {
		m := &discovery.InstrumentMetadata{Source: "gate", SourceURL: url, NativeTradingStatus: "DELISTING", SettlementAsset: "USDT"}
		rule := identity.MarketOverride{Exchange: "gate", RawSymbol: "ACE_USDT", MarketType: "perpetual", ComparisonStatus: identity.ComparisonEligible}
		guardGateInactive(&rule, m, false)
		if rule.ComparisonStatus != identity.ComparisonEligible {
			t.Fatal(rule)
		}
	}
}

func TestGateNativeExclusionOutranksConsumerAndPreservesConflicts(t *testing.T) {
	for _, mt := range []string{"spot", "perpetual"} {
		url, status := "https://api.gateio.ws/api/v4/spot/currency_pairs", "UNTRADABLE"
		if mt == "perpetual" {
			url, status = "https://api.gateio.ws/api/v4/futures/usdt/contracts", "DELISTING"
		}
		native := discovery.ImportedMarket{PlatformID: "gate", VenueType: "cex", MarketType: mt, Symbol: "ACE_USDT", BaseAsset: "ACE", QuoteAsset: "USDT", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "gate", SourceURL: url, NativeTradingStatus: status, SettlementAsset: "USDT"}}
		weak := native
		weak.InstrumentMetadata = nil
		weak.AssetClassHint = "stock"
		conflict := native
		m := *native.InstrumentMetadata
		m.SettlementAsset = "USDC"
		conflict.InstrumentMetadata = &m
		for _, rows := range [][]discovery.ImportedMarket{{weak, native}, {native, weak}, {native, conflict}, {conflict, native}} {
			got := identity.NewResolver(BuildGeneratedRegistry(rows)).Resolve(identity.ResolveRequest{Exchange: "gate", Symbol: "ACE_USDT", MarketTypeHint: mt})
			if got.Market == nil || got.Market.ComparisonStatus != identity.ComparisonProhibited {
				t.Fatalf("weak evidence reenabled product: %+v", got)
			}
			if rows[0].InstrumentMetadata != nil && rows[1].InstrumentMetadata != nil && got.Market.SettlementAsset != "" {
				t.Fatalf("conflicting native collateral filled from quote: %+v", got.Market)
			}
		}
	}
}
