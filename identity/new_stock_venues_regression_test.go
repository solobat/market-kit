package identity

import "testing"

func TestNewStockVenuesRetainIndependentProductBoundaries(t *testing.T) {
	registry, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(registry)
	get := func(exchange, symbol string) *MarketIdentity {
		result := resolver.Resolve(ResolveRequest{Exchange: exchange, Symbol: symbol, MarketTypeHint: "perpetual"})
		if result.Market == nil || result.Market.AssetClass != "rwa_stock" || result.Market.ComparisonStatus != ComparisonAmbiguous {
			t.Fatalf("%s %s: %+v", exchange, symbol, result)
		}
		return result.Market
	}
	gate := get("gate", "IBKR_USDT")
	bitget := get("bitget", "IBKRUSDT")
	if gate.CanonicalSymbol != bitget.CanonicalSymbol || gate.AssetID == bitget.AssetID || gate.ComparisonKey == bitget.ComparisonKey {
		t.Fatal(gate, bitget)
	}
	elice := get("bitget", "ELICEUSDT")
	if elice.AssetID != "equity:bitget:perpetual:elice" {
		t.Fatal(elice)
	}
	japan := get("gate", "FASTRETAILING_USDT")
	if japan.UnitMultiplier != 1 || japan.BaseAsset != "FASTRETAILING" {
		t.Fatal(japan)
	}
}
