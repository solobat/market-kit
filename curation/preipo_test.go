package curation

import (
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestGeneratedPreIPOIdentityPreservesUnitsBoundary(t *testing.T) {
	r := BuildGeneratedRegistry([]discovery.ImportedMarket{{PlatformID: "hyperliquid", VenueType: "dex", Symbol: "xyz:FUTUREAI", BaseAsset: "FUTUREAI", QuoteAsset: "USDC", MarketType: "perp", Status: "live", InstrumentMetadata: &discovery.InstrumentMetadata{PreIPOIssuer: "OPENAI", PreIPOEvidenceURL: discovery.XYZPreIPOURL}}})
	if len(r.MarketOverrides) != 1 {
		t.Fatalf("%+v", r)
	}
	m := r.MarketOverrides[0]
	if m.CanonicalSymbol != "OPENAI/USDC" || m.AssetClass != "rwa_stock" || m.UnderlyingID != "equity:private:openai" || m.ComparisonStatus != identity.ComparisonAmbiguous {
		t.Fatalf("%+v", m)
	}
}
