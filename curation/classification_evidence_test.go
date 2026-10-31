package curation

import (
	"encoding/json"
	"testing"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

func TestExplicitCoinEvidenceSurvivesMissingMetadataAndTickerCollision(t *testing.T) {
	coin := discovery.ImportedMarket{PlatformID: "coins", VenueType: "cex", MarketType: "perp", Symbol: "FRESHUSDT", BaseAsset: "FRESH", QuoteAsset: "USDT", UnderlyingCategory: "coin"}
	stock := discovery.ImportedMarket{PlatformID: "stocks", VenueType: "cex", MarketType: "spot", Symbol: "RFRESHUSDT", BaseAsset: "FRESH", QuoteAsset: "USDT", AssetClassHint: "stock"}
	existing := BuildGeneratedRegistry([]discovery.ImportedMarket{coin})
	// A persisted registry must retain the evidence, too.
	encoded, err := json.Marshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(encoded, &existing); err != nil {
		t.Fatal(err)
	}
	coin.UnderlyingCategory = ""
	for i := 0; i < 2; i++ {
		existing = MergeGeneratedRegistry(existing, BuildGeneratedRegistry([]discovery.ImportedMarket{coin, stock}), false)
		got := identity.NewResolver(existing).Resolve(identity.ResolveRequest{Exchange: "coins", Symbol: "FRESHUSDT", MarketTypeHint: "perpetual"})
		if got.Market == nil || got.Market.AssetClass != "crypto" || got.Market.ComparisonStatus != identity.ComparisonEligible {
			t.Fatalf("explicit market evidence lost on sync %d: %+v", i, got)
		}
		other := identity.NewResolver(existing).Resolve(identity.ResolveRequest{Exchange: "stocks", Symbol: "RFRESHUSDT", MarketTypeHint: "spot"})
		if other.Market == nil || other.Market.UnderlyingID == got.Market.UnderlyingID || len(existing.AssetCollisions) != 1 {
			t.Fatal(other, existing.AssetCollisions)
		}
	}
}

func TestClassificationRetentionDoesNotOverrideContraryOrUnverifiedEvidence(t *testing.T) {
	coin := discovery.ImportedMarket{PlatformID: "coins", VenueType: "cex", MarketType: "perp", Symbol: "FRESHUSDT", BaseAsset: "FRESH", QuoteAsset: "USDT", UnderlyingCategory: "coin"}
	existing := BuildGeneratedRegistry([]discovery.ImportedMarket{coin})
	existing.AssetCollisions = []identity.AssetCollision{{Symbol: "FRESH", AssetClasses: []string{"crypto", "rwa_stock"}}}
	weak := coin
	weak.UnderlyingCategory = ""
	stock := weak
	stock.PlatformID = "stocks"
	stock.Symbol = "RFRESHUSDT"
	stock.AssetClassHint = "stock"
	for _, tc := range []struct {
		name      string
		old       identity.Registry
		rows      []discovery.ImportedMarket
		prune     bool
		wantClass string
	}{
		{"legacy lacks provenance", func() identity.Registry {
			r := existing.Clone()
			r.MarketOverrides[0].ClassificationEvidence = ""
			return r
		}(), []discovery.ImportedMarket{weak, stock}, false, "unknown"},
		{"generic crypto lacks provenance", func() identity.Registry {
			generic := weak
			generic.AssetClassHint = "crypto"
			r := BuildGeneratedRegistry([]discovery.ImportedMarket{generic})
			r.AssetCollisions = existing.AssetCollisions
			return r
		}(), []discovery.ImportedMarket{weak, stock}, false, "unknown"},
		{"positive stock reclassification", existing, []discovery.ImportedMarket{func() discovery.ImportedMarket { r := weak; r.AssetClassHint = "stock"; return r }()}, false, "rwa_stock"},
		{"conflicting market evidence", existing, []discovery.ImportedMarket{weak, func() discovery.ImportedMarket { r := weak; r.BaseAsset = "OTHER"; return r }()}, false, "unknown"},
		{"pruning discards history", existing, []discovery.ImportedMarket{weak, stock, func() discovery.ImportedMarket { r := coin; r.PlatformID = "anchor"; return r }()}, true, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := MergeGeneratedRegistry(tc.old, BuildGeneratedRegistry(tc.rows), tc.prune)
			found := false
			for _, got := range r.MarketOverrides {
				if got.Exchange != "coins" {
					continue
				}
				found = true
				if got.AssetClass != tc.wantClass {
					t.Fatalf("got %+v, want %s", got, tc.wantClass)
				}
			}
			if !found {
				t.Fatal("target market disappeared")
			}
		})
	}
	changed := BuildGeneratedRegistry([]discovery.ImportedMarket{weak, stock})
	changed.MarketOverrides[0].CanonicalSymbol = "OTHER/USDT"
	retained := retainExplicitMarketClassification(existing, changed)
	if retained.MarketOverrides[0].ClassificationEvidence == "explicit_crypto" {
		t.Fatal("changed native product inherited old identity")
	}
}
