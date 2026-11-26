package identity

import "testing"

func TestVariationalBabylCanonicalIdentity(t *testing.T) {
	registry, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(registry)
	v := resolver.Resolve(ResolveRequest{Exchange: "variational", Symbol: "BABYL", MarketTypeHint: "perpetual"}).Market
	b := resolver.Resolve(ResolveRequest{Exchange: "bybit", Symbol: "BABYUSDT", MarketTypeHint: "perpetual"}).Market
	if v == nil || b == nil {
		t.Fatal(v, b)
	}
	if v.BaseAsset != "BABY" || v.AssetID != b.AssetID || v.UnderlyingID != b.UnderlyingID || v.ComparisonStatus != ComparisonEligible {
		t.Fatal(v, b)
	}
	if v.VenueSymbol != "BABYL" || v.RawSymbol != "BABYL" || v.QuoteAsset != "USDC" || v.SettlementAsset != "USDC" || v.ComparisonKey != "crypto:ticker:baby/USDC" {
		t.Fatal(v)
	}
	other := resolver.Resolve(ResolveRequest{Exchange: "bybit", Symbol: "BABYLUSDT", MarketTypeHint: "perpetual"}).Market
	if other != nil && other.AssetID == b.AssetID {
		t.Fatal("VAR alias leaked to another venue", other)
	}
}
