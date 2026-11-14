package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"strings"
)

func nativePopdex(exchange string, m *discovery.InstrumentMetadata) bool {
	return exchange == "popdex" && m != nil && m.Source == "popdex" && m.SourceURL == "https://api.popdex.xyz/api/v1/config/symbols?category=Futures" && m.ContractType == "perpetual"
}
func popdexPhase(exchange string, m *discovery.InstrumentMetadata) string {
	if !nativePopdex(exchange, m) {
		return ""
	}
	switch m.NativeTradingStatus {
	case "PreTrading":
		return "prelaunch"
	case "Delisted":
		return "delisted"
	}
	return ""
}
func popdexUnclassified(exchange string, m *discovery.InstrumentMetadata) bool {
	return nativePopdex(exchange, m) && m.IsRWA && m.SymbolType == "unclassified_rwa"
}
func guardPopdexProduct(rule *identity.MarketOverride, m *discovery.InstrumentMetadata, conflict bool) {
	if rule.MarketType != "perpetual" {
		return
	}
	phase := popdexPhase(rule.Exchange, m)
	unknown := popdexUnclassified(rule.Exchange, m)
	if phase == "" && !unknown {
		return
	}
	prefix := strings.SplitN(rule.AssetID, ":", 2)[0]
	if unknown {
		rule.AssetClass = "unknown"
		rule.InstrumentKind = "perpetual"
		prefix = "rwa"
	}
	scope := phase
	if scope == "" {
		scope = "perpetual"
	}
	rule.AssetID = prefix + ":popdex:" + scope + ":" + strings.ToLower(rule.RawSymbol)
	rule.UnderlyingID = rule.AssetID
	_, quote, _ := strings.Cut(rule.CanonicalSymbol, "/")
	rule.ComparisonKey = rule.AssetID + "/" + quote
	rule.ComparisonStatus = identity.ComparisonAmbiguous
	rule.ContractType = "perpetual"
	if phase != "" {
		rule.ComparisonStatus = identity.ComparisonProhibited
		rule.ContractType = phase + "_perpetual"
	}
	rule.SettlementAsset = m.SettlementAsset
	if conflict || rule.SettlementAsset == "" {
		rule.SettlementAsset = ""
		if phase == "prelaunch" {
			rule.ContractType = "prelaunch_perpetual_unverified"
		} else {
			rule.ContractType = "rwa_perpetual_unverified"
		}
	}
}
func samePopdexEvidence(a, b discovery.ImportedMarket) bool {
	if !nativePopdex(a.PlatformID, a.InstrumentMetadata) || !nativePopdex(b.PlatformID, b.InstrumentMetadata) {
		return true
	}
	x, y := a.InstrumentMetadata, b.InstrumentMetadata
	return x.NativeTradingStatus == y.NativeTradingStatus && x.SettlementAsset == y.SettlementAsset && x.IsRWA == y.IsRWA && x.SymbolType == y.SymbolType
}
