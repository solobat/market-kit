package identity

import "testing"

func TestGTradeDARDoesNotJoinHomonymousStock(t *testing.T) {
	registry, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	// The same ticker is an equity in the official stock-token venue.
	registry.MarketOverrides = append(registry.MarketOverrides, MarketOverride{Exchange: "bitget", RawSymbol: "RDARUSDT", MarketType: "spot", CanonicalSymbol: "DAR/USDT", AssetClass: "rwa_stock", AssetID: "equity:fixture:darling-ingredients", UnderlyingID: "equity:fixture:darling-ingredients", ComparisonKey: "equity:fixture:darling-ingredients/USDT", ComparisonStatus: ComparisonAmbiguous})
	resolver := NewResolver(registry)
	crypto := resolver.Resolve(ResolveRequest{Exchange: "gtrade", Symbol: "DAR/USD", MarketTypeHint: "perpetual"})
	stock := resolver.Resolve(ResolveRequest{Exchange: "bitget", Symbol: "RDARUSDT", MarketTypeHint: "spot"})
	if crypto.Market == nil || crypto.Market.AssetClass != "crypto" || crypto.Market.ComparisonStatus != ComparisonAmbiguous || crypto.Market.ContractType != "multi_collateral_cfd" || crypto.Market.SettlementAsset != "" {
		t.Fatal(crypto)
	}
	if stock.Market == nil || stock.Market.AssetClass != "rwa_stock" || stock.Market.UnderlyingID == crypto.Market.UnderlyingID || stock.Market.ComparisonKey == crypto.Market.ComparisonKey {
		t.Fatal(stock, crypto)
	}
}
