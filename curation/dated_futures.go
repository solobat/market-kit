package curation

import (
	"regexp"
	"strings"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

var bybitDatedSuffix = regexp.MustCompile(`-[0-9]{2}[A-Z]{3}[0-9]{2}$`)

// Runs once per inventory row at discovery cadence. A name can only quarantine
// an unverified derivative; it cannot supply expiry, settlement or eligibility.
func guardBybitDatedFuture(rule *identity.MarketOverride, metadata *discovery.InstrumentMetadata) {
	if rule.Exchange != "bybit" || rule.MarketType == "spot" {
		return
	}
	verified := isBybitFutureMetadata(metadata)
	if !verified && rule.MarketType != "future" && !bybitDatedSuffix.MatchString(strings.ToUpper(rule.RawSymbol)) {
		return
	}
	prefix := strings.SplitN(rule.AssetID, ":", 2)[0]
	if prefix == "" {
		prefix = "unknown"
	}
	rule.AssetID = prefix + ":bybit:future:" + strings.ToLower(rule.RawSymbol)
	_, quote, _ := strings.Cut(rule.CanonicalSymbol, "/")
	rule.ComparisonKey = rule.AssetID + "/" + quote
	rule.InstrumentKind = "future"
	rule.ContractType = "dated_future_unverified"
	rule.SettlementAsset = ""
	rule.ExpiryAtMs = 0
	rule.ComparisonStatus = identity.ComparisonProhibited
	if verified {
		rule.ContractType = strings.ToLower(strings.TrimSuffix(metadata.ContractType, "Futures")) + "_futures"
		rule.SettlementAsset = metadata.SettlementAsset
		rule.ExpiryAtMs = metadata.ExpiryAtMs
		if rule.MarketType == "future" && rule.ExpiryAtMs > 0 && rule.SettlementAsset != "" {
			rule.ComparisonStatus = identity.ComparisonAmbiguous
		}
	}
}

func isBybitFutureMetadata(metadata *discovery.InstrumentMetadata) bool {
	return metadata != nil && metadata.Source == "bybit" && (strings.EqualFold(metadata.ContractType, "LinearFutures") || strings.EqualFold(metadata.ContractType, "InverseFutures"))
}

func sameDatedProductEvidence(a, b discovery.ImportedMarket) bool {
	if !strings.EqualFold(a.PlatformID, "bybit") || !isBybitFutureMetadata(a.InstrumentMetadata) || !isBybitFutureMetadata(b.InstrumentMetadata) {
		return true
	}
	x, y := a.InstrumentMetadata, b.InstrumentMetadata
	return strings.EqualFold(x.ContractType, y.ContractType) && x.ExpiryAtMs == y.ExpiryAtMs && strings.EqualFold(x.SettlementAsset, y.SettlementAsset)
}
