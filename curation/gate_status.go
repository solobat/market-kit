package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"strings"
)

// Native exclusion evidence is interpreted at generation cadence, not Resolve.
func gateInactivePhase(exchange, marketType string, m *discovery.InstrumentMetadata) string {
	if exchange != "gate" || m == nil || m.Source != "gate" {
		return ""
	}
	url := strings.SplitN(m.SourceURL, "?", 2)[0]
	if marketType == "spot" && url == "https://api.gateio.ws/api/v4/spot/currency_pairs" && m.NativeTradingStatus == "UNTRADABLE" {
		return "untradable"
	}
	if marketType == "perpetual" && url == "https://api.gateio.ws/api/v4/futures/usdt/contracts" && m.NativeTradingStatus == "DELISTING" {
		return "delisting"
	}
	return ""
}

func guardGateInactive(rule *identity.MarketOverride, m *discovery.InstrumentMetadata, conflict bool) {
	phase := gateInactivePhase(rule.Exchange, rule.MarketType, m)
	if phase == "" {
		return
	}
	prefix := strings.SplitN(rule.AssetID, ":", 2)[0]
	if prefix == "" {
		prefix = "unknown"
	}
	rule.AssetID = prefix + ":gate:" + phase + ":" + strings.ToLower(rule.RawSymbol)
	rule.UnderlyingID = rule.AssetID
	_, quote, _ := strings.Cut(rule.CanonicalSymbol, "/")
	rule.ComparisonKey = rule.AssetID + "/" + quote
	rule.ComparisonStatus = identity.ComparisonProhibited
	rule.ContractType = phase + "_" + rule.MarketType
	rule.SettlementAsset = m.SettlementAsset
	if conflict {
		rule.ContractType = "gate_" + rule.ContractType + "_unverified"
		rule.SettlementAsset = ""
	}
}

func sameGateStateEvidence(a, b discovery.ImportedMarket) bool {
	if gateInactivePhase(a.PlatformID, normalizeGeneratedMarketType(a.MarketType), a.InstrumentMetadata) == "" && gateInactivePhase(b.PlatformID, normalizeGeneratedMarketType(b.MarketType), b.InstrumentMetadata) == "" {
		return true
	}
	x, y := a.InstrumentMetadata, b.InstrumentMetadata
	return x != nil && y != nil && x.NativeTradingStatus == y.NativeTradingStatus && x.ContractType == y.ContractType && x.SettlementAsset == y.SettlementAsset
}
