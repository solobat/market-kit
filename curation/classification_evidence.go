package curation

import (
	"strings"

	"github.com/solobat/market-kit/identity"
)

// Own normalization state once for both reconciliation steps.
func reconcileGeneratedClassification(existing, current identity.Registry) identity.Registry {
	existing = existing.Clone()
	current = current.Clone()
	existing.Normalize()
	current.Normalize()
	current = quarantineHistoricalTickerCollisions(existing, current)
	return retainExplicitMarketClassification(existing, current)
}

// An incomplete refresh must not erase a known ticker collision and then borrow
// the other asset's class. Combine explicit market evidence across syncs before
// deciding whether a ticker-wide fallback is safe.
func quarantineHistoricalTickerCollisions(existing, current identity.Registry) identity.Registry {
	classes := map[string]map[string]bool{}
	for _, registry := range []identity.Registry{existing, current} {
		for _, collision := range registry.AssetCollisions {
			if classes[collision.Symbol] == nil {
				classes[collision.Symbol] = map[string]bool{}
			}
			for _, class := range collision.AssetClasses {
				classes[collision.Symbol][class] = true
			}
		}
		for _, rule := range registry.MarketOverrides {
			if rule.ClassificationEvidence != "explicit_crypto" && rule.ClassificationEvidence != "explicit_rwa" {
				continue
			}
			base, _ := splitGeneratedCanonicalSymbol(rule.CanonicalSymbol)
			if base == "" || rule.AssetClass == "" || rule.AssetClass == "unknown" {
				continue
			}
			if classes[base] == nil {
				classes[base] = map[string]bool{}
			}
			classes[base][rule.AssetClass] = true
		}
	}
	current.AssetCollisions = generatedAssetCollisions(classes)
	collisions := map[string]bool{}
	for _, collision := range current.AssetCollisions {
		collisions[collision.Symbol] = true
	}
	assets := current.AssetAliases[:0]
	for _, asset := range current.AssetAliases {
		if !collisions[asset.Canonical] {
			assets = append(assets, asset)
		}
	}
	current.AssetAliases = assets
	for i := range current.MarketOverrides {
		rule := &current.MarketOverrides[i]
		base, _ := splitGeneratedCanonicalSymbol(rule.CanonicalSymbol)
		if !collisions[base] || rule.ClassificationEvidence != "" || (rule.AssetClass == "unknown" || rule.AssetClass == "") ||
			rule.ContractType != "" || rule.ExpiryAtMs != 0 || rule.ComparisonStatus == identity.ComparisonProhibited {
			continue
		}
		rule.AssetClass = "unknown"
		rule.AssetID = identity.CanonicalAssetID("unknown", base)
		rule.UnderlyingID = rule.AssetID
		rule.ComparisonKey = ""
		rule.ComparisonStatus = identity.ComparisonAmbiguous
		rule.InstrumentKind = identity.DefaultInstrumentKind(identity.MarketType(rule.MarketType), "unknown")
	}
	return current
}

// Missing metadata is not contrary identity evidence. Retain only an exact
// market's explicit coin evidence, never a ticker-wide or heuristic crypto label.
// This runs once per discovery sync, not during Resolve or ticker processing.
func retainExplicitMarketClassification(existing, current identity.Registry) identity.Registry {
	known := make(map[string]identity.MarketOverride)
	for _, old := range existing.MarketOverrides {
		if old.ClassificationEvidence == "explicit_crypto" && old.AssetClass == "crypto" &&
			old.ComparisonStatus == identity.ComparisonEligible && strings.HasPrefix(old.AssetID, "crypto:") &&
			old.UnderlyingID != "" && old.ComparisonKey != "" &&
			(old.MarketType == "spot" || old.MarketType == "perpetual") {
			known[overrideKey(old)] = old
		}
	}
	for i, next := range current.MarketOverrides {
		if next.ClassificationEvidence != "" || (next.AssetClass != "" && next.AssetClass != "unknown") ||
			next.ComparisonStatus != identity.ComparisonAmbiguous || !generatedIdentityMissing(next.AssetID) ||
			!generatedIdentityMissing(next.UnderlyingID) || next.ComparisonKey != "" ||
			next.ContractType != "" || next.ExpiryAtMs != 0 || next.UnitAlias != "" || next.UnitMultiplier != 0 {
			continue
		}
		if old, ok := known[overrideKey(next)]; ok && old.CanonicalSymbol == next.CanonicalSymbol &&
			old.InstrumentKind == next.InstrumentKind && old.SettlementAsset == next.SettlementAsset {
			current.MarketOverrides[i] = old
		}
	}
	return current
}
