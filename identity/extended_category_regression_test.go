package identity

import "testing"

func TestReviewedExtendedCategoryVariantsAndMetaplexCollision(t *testing.T) {
	registry, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(registry)
	for _, pair := range []struct{ base, class string }{{"ANTHROP", "rwa_stock"}, {"JIO", "rwa_stock"}, {"NOW", "rwa_stock"}, {"LYTE", "rwa_stock"}, {"BILL", "crypto"}, {"SUI", "crypto"}} {
		var first *MarketIdentity
		for _, separator := range []string{"-", "/"} {
			result := resolver.Resolve(ResolveRequest{Exchange: "extended", Symbol: pair.base + separator + "USD", MarketTypeHint: "perpetual"})
			market := result.Market
			if market == nil || market.AssetClass != pair.class || market.ComparisonStatus != ComparisonAmbiguous {
				t.Fatalf("%s: %+v", pair.base, result)
			}
			if first != nil && (first.AssetID != market.AssetID || first.ComparisonKey != market.ComparisonKey) {
				t.Fatal(first, market)
			}
			first = market
		}
	}
	result := resolver.Resolve(ResolveRequest{Exchange: "gate", Symbol: "MPLX_USDT", MarketTypeHint: "perpetual"})
	if result.Market == nil || result.Market.AssetClass != "crypto" || result.Market.AssetID != "crypto:gate:metaplex" || result.Market.UnitMultiplier != 1 {
		t.Fatal(result)
	}
}
