package curation

import (
	"regexp"
	"strings"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

var binanceDatedSuffix = regexp.MustCompile(`_[0-9]{6}$`)

var bybitDatedSuffix = regexp.MustCompile(`-[0-9]{2}[A-Z]{3}[0-9]{2}$`)

// Runs once per inventory row at discovery cadence. A name can only quarantine
// an unverified derivative; it cannot supply expiry, settlement or eligibility.
func guardDatedFuture(rule *identity.MarketOverride, metadata *discovery.InstrumentMetadata) {
	if (rule.Exchange != "bybit" && rule.Exchange != "binance") || rule.MarketType == "spot" {
		return
	}
	verified := isNativeFutureMetadata(rule.Exchange, metadata)
	if !verified && rule.MarketType != "future" && !datedSymbolForExchange(rule.Exchange, rule.RawSymbol) {
		return
	}
	prefix := strings.SplitN(rule.AssetID, ":", 2)[0]
	if prefix == "" {
		prefix = "unknown"
	}
	rule.AssetID = prefix + ":" + rule.Exchange + ":future:" + strings.ToLower(rule.RawSymbol)
	_, quote, _ := strings.Cut(rule.CanonicalSymbol, "/")
	rule.ComparisonKey = rule.AssetID + "/" + quote
	rule.InstrumentKind = "future"
	rule.ContractType = "dated_future_unverified"
	rule.SettlementAsset = ""
	rule.ExpiryAtMs = 0
	rule.ComparisonStatus = identity.ComparisonProhibited
	if verified {
		rule.ContractType = strings.ToLower(strings.TrimSuffix(metadata.ContractType, "Futures")) + "_futures"
		if rule.Exchange == "binance" {
			rule.ContractType = "linear_futures"
		}
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
	if !isNativeFutureMetadata(a.PlatformID, a.InstrumentMetadata) || !isNativeFutureMetadata(b.PlatformID, b.InstrumentMetadata) {
		return true
	}
	x, y := a.InstrumentMetadata, b.InstrumentMetadata
	return strings.EqualFold(x.ContractType, y.ContractType) && x.ExpiryAtMs == y.ExpiryAtMs && strings.EqualFold(x.SettlementAsset, y.SettlementAsset)
}

func datedSymbolForExchange(exchange, symbol string) bool {
	switch exchange {
	case "bybit":
		return bybitDatedSuffix.MatchString(strings.ToUpper(symbol))
	case "binance":
		return binanceDatedSuffix.MatchString(symbol)
	}
	return false
}
func isNativeFutureMetadata(exchange string, m *discovery.InstrumentMetadata) bool {
	if strings.EqualFold(exchange, "bybit") {
		return isBybitFutureMetadata(m)
	}
	if !strings.EqualFold(exchange, "binance") || m == nil || m.Source != "binance" {
		return false
	}
	// The quarter enum also exists on COIN-M; only the USD-M endpoint
	// proves this linear-future normalization. Other producers fail closed.
	if strings.SplitN(m.SourceURL, "?", 2)[0] != "https://fapi.binance.com/fapi/v1/exchangeInfo" {
		return false
	}
	kind := strings.TrimSuffix(strings.ToUpper(m.ContractType), "_DELIVERING")
	return kind == "CURRENT_QUARTER" || kind == "NEXT_QUARTER"
}
