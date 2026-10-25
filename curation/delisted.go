package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"strings"
)

// Keep inactive products in the registry for historical identity lookups, but
// never treat an explicit venue delisting as an eligible comparison market.
func guardHyperliquidDelisted(rule *identity.MarketOverride, metadata *discovery.InstrumentMetadata) {
	if rule.Exchange != "hyperliquid" || metadata == nil || metadata.Source != "hyperliquid" || !metadata.IsDelisted {
		return
	}
	prefix := strings.SplitN(rule.AssetID, ":", 2)[0]
	if prefix == "" {
		prefix = "unknown"
	}
	rule.AssetID = prefix + ":hyperliquid:delisted:" + strings.ToLower(rule.RawSymbol)
	_, quote, _ := strings.Cut(rule.CanonicalSymbol, "/")
	rule.ComparisonKey = rule.AssetID + "/" + quote
	rule.ComparisonStatus = identity.ComparisonProhibited
	rule.ContractType = "delisted_perpetual"
}
