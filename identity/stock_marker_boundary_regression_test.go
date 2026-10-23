package identity

import "testing"

func TestATAndTAndRT1RetainNativeNamesAndSeparateClasses(t *testing.T) {
	registry, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(registry)
	stock := resolver.Resolve(ResolveRequest{Exchange: "bitget", Symbol: "TSTOCKUSDT", MarketTypeHint: "perpetual"})
	token := resolver.Resolve(ResolveRequest{Exchange: "bitget", Symbol: "RT1USDT", MarketTypeHint: "spot"})
	if stock.Market == nil || stock.Market.AssetClass != "rwa_stock" || stock.Market.CanonicalSymbol != "TSTOCK/USDT" || stock.Market.ComparisonStatus != ComparisonAmbiguous {
		t.Fatal(stock)
	}
	if token.Market == nil || token.Market.AssetClass != "crypto" || token.Market.CanonicalSymbol != "RT1/USDT" || token.Market.ComparisonStatus != ComparisonAmbiguous {
		t.Fatal(token)
	}
	if stock.Market.AssetID == token.Market.AssetID || stock.Market.ComparisonKey == token.Market.ComparisonKey || stock.Market.UnitMultiplier != 1 || token.Market.UnitMultiplier != 1 {
		t.Fatal(stock, token)
	}
}
