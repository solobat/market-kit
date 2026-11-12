package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestBinanceNonTradingRequiresNativeUSDMProof(t *testing.T) {
	for _, url := range []string{"https://dapi.binance.com/dapi/v1/exchangeInfo", "https://example.com"} {
		m := &discovery.InstrumentMetadata{Source: "binance", SourceURL: url, NativeTradingStatus: "PENDING_TRADING", ContractType: "PERPETUAL"}
		rule := identity.MarketOverride{Exchange: "binance", MarketType: "perpetual", RawSymbol: "XUSDT", ComparisonStatus: identity.ComparisonEligible}
		guardBinanceInactive(&rule, m)
		if rule.ComparisonStatus != identity.ComparisonEligible {
			t.Fatal(rule)
		}
	}
}

func TestBinanceNonTradingConflictingCollateralStaysProhibited(t *testing.T) {
	a := discovery.ImportedMarket{PlatformID: "binance", VenueType: "cex", MarketType: "perp", Symbol: "NEWUSDT", BaseAsset: "NEW", QuoteAsset: "USDT", AssetClassHint: "crypto", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "binance", SourceURL: "https://fapi.binance.com/fapi/v1/exchangeInfo", NativeTradingStatus: "PENDING_TRADING", ContractType: "PERPETUAL", SettlementAsset: "USDT"}}
	b := a
	m := *a.InstrumentMetadata
	m.SettlementAsset = "USDC"
	b.InstrumentMetadata = &m
	for _, rows := range [][]discovery.ImportedMarket{{a, b}, {b, a}} {
		result := identity.NewResolver(BuildGeneratedRegistry(rows)).Resolve(identity.ResolveRequest{Exchange: "binance", Symbol: "NEWUSDT", MarketTypeHint: "perpetual"})
		if result.Market == nil || result.Market.ComparisonStatus != identity.ComparisonProhibited || result.Market.SettlementAsset != "" {
			t.Fatal(result)
		}
	}
}
