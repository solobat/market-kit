package discovery

import (
	"github.com/solobat/market-kit/identity"
	"testing"
)

func TestDirectoryEvidenceOverridesStaleCryptoIdentity(t *testing.T) {
	reg := identity.Registry{MarketOverrides: []identity.MarketOverride{{Exchange: "hyperliquid", RawSymbol: "xyz:FUTUREAI", MarketType: "perpetual", CanonicalSymbol: "FUTUREAI/USDC", AssetClass: "crypto", AssetID: "crypto:ticker:futureai"}}}
	rows := NewAggregator(reg).NormalizeImportedMarkets([]ImportedMarket{{PlatformID: "hyperliquid", Symbol: "xyz:FUTUREAI", BaseAsset: "FUTUREAI", QuoteAsset: "USDC", MarketType: "perp", InstrumentMetadata: &InstrumentMetadata{PreIPOIssuer: "OPENAI", PreIPOEvidenceURL: XYZPreIPOURL}}})
	if len(rows) != 1 || rows[0].BaseAsset != "OPENAI" || rows[0].AssetClass != "rwa_stock" || rows[0].ComparisonStatus != identity.ComparisonAmbiguous {
		t.Fatalf("%+v", rows)
	}
	if _, ok := OfficialPreIPOIssuer(ImportedMarket{PlatformID: "binance", Symbol: "xyz:FUTUREAI", InstrumentMetadata: &InstrumentMetadata{PreIPOIssuer: "OPENAI", PreIPOEvidenceURL: XYZPreIPOURL}}); ok {
		t.Fatal("cross-venue leak")
	}
}
