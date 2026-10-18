package curation

import (
	"strings"

	"github.com/solobat/market-kit/discovery"
)

// Select once at discovery cadence; row ordering must not choose an identity.
// Generic downstream classifications are weaker than explicit venue metadata.
func selectMarketIdentityEvidence(items []discovery.ImportedMarket) ([]discovery.ImportedMarket, map[string]bool) {
	selected := map[string]discovery.ImportedMarket{}
	conflicts := map[string]bool{}
	order := make([]string, 0, len(items))
	for _, row := range items {
		if !shouldInclude(row) {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(row.PlatformID)) + "|" + strings.TrimSpace(row.Symbol) + "|" + normalizeGeneratedMarketType(row.MarketType)
		old, exists := selected[key]
		if !exists {
			selected[key] = row
			order = append(order, key)
			continue
		}
		oldRank, newRank := marketEvidenceRank(old), marketEvidenceRank(row)
		if newRank > oldRank {
			selected[key] = row
			delete(conflicts, key)
			continue
		}
		if newRank < oldRank {
			continue
		}
		same := strings.EqualFold(strings.TrimSpace(old.BaseAsset), strings.TrimSpace(row.BaseAsset)) && strings.EqualFold(strings.TrimSpace(old.QuoteAsset), strings.TrimSpace(row.QuoteAsset)) && inferAssetClassFromExchangeMetadata(old) == inferAssetClassFromExchangeMetadata(row)
		if !same {
			conflicts[key] = true
		}
		// Stable source tie-break only applies to equally supported identities.
		if same && strings.Compare(row.SourceID, old.SourceID) < 0 {
			selected[key] = row
		}
	}
	out := make([]discovery.ImportedMarket, 0, len(order))
	for _, key := range order {
		out = append(out, selected[key])
	}
	return out, conflicts
}
func marketEvidenceRank(row discovery.ImportedMarket) int {
	if isForexMarket(row) || hasExplicitRWAHint(row) {
		return 5
	}
	if hasExplicitCryptoIdentityHint(row) {
		return 4
	}
	if inferAssetClassFromExchangeMetadata(row) != "" {
		return 1
	}
	return 0
}

// These are currency-pair CFDs, not crypto tokens or stablecoin balances.
// Keep this venue/metadata scoped; an EUR ticker alone is not sufficient.
func isForexMarket(row discovery.ImportedMarket) bool {
	group := strings.ToLower(strings.TrimSpace(row.Category))
	switch strings.ToLower(strings.TrimSpace(row.PlatformID)) {
	case "gtrade":
		return group == "forex" || group == "forex-minor" || group == "forex-exotic"
	case "extended":
		return group == "fx"
	default:
		return false
	}
}
