package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"strings"
)

// Native non-trading states are exclusion evidence, not identity or unit proof.
// Run only at discovery generation cadence; never parse status on Resolve.
func binanceInactivePhase(exchange string, m *discovery.InstrumentMetadata) string {
	if exchange != "binance" || m == nil || m.Source != "binance" || strings.SplitN(m.SourceURL, "?", 2)[0] != "https://fapi.binance.com/fapi/v1/exchangeInfo" || (m.ContractType != "PERPETUAL" && m.ContractType != "TRADIFI_PERPETUAL") {
		return ""
	}
	switch strings.ToUpper(strings.TrimSpace(m.NativeTradingStatus)) {
	case "PRE_TRADING", "PENDING_TRADING":
		return "prelaunch"
	case "SETTLING":
		return "settling"
	}
	return ""
}

func guardBinanceInactive(rule *identity.MarketOverride, m *discovery.InstrumentMetadata) {
	phase := binanceInactivePhase(rule.Exchange, m)
	if phase == "" || rule.MarketType != "perpetual" {
		return
	}
	prefix := strings.SplitN(rule.AssetID, ":", 2)[0]
	if prefix == "" {
		prefix = "unknown"
	}
	rule.AssetID = prefix + ":binance:" + phase + ":" + strings.ToLower(rule.RawSymbol)
	rule.UnderlyingID = rule.AssetID
	_, quote, _ := strings.Cut(rule.CanonicalSymbol, "/")
	rule.ComparisonKey = rule.AssetID + "/" + quote
	rule.ComparisonStatus = identity.ComparisonProhibited
	rule.ContractType = phase + "_perpetual"
	rule.SettlementAsset = m.SettlementAsset
}

func sameBinanceStateEvidence(a, b discovery.ImportedMarket) bool {
	if binanceInactivePhase(a.PlatformID, a.InstrumentMetadata) == "" && binanceInactivePhase(b.PlatformID, b.InstrumentMetadata) == "" {
		return true
	}
	x, y := a.InstrumentMetadata, b.InstrumentMetadata
	return x != nil && y != nil && x.NativeTradingStatus == y.NativeTradingStatus && x.ContractType == y.ContractType && x.SettlementAsset == y.SettlementAsset
}
