package identity

import (
	"strings"
	"testing"
)

func TestNumericIndexNeverBecomesScaledCrypto(t *testing.T) {
	r := Registry{AssetAliases: []AssetAliasRule{{Canonical: "US", AssetClass: "crypto", UnitAliases: []AssetUnitAlias{{Alias: "US100", Multiplier: 100}}}, {Canonical: "US100", AssetClass: "rwa_stock"}}, MarketOverrides: []MarketOverride{{Exchange: "lighter", RawSymbol: "US100", MarketType: "perpetual", CanonicalSymbol: "US100/USDC", AssetClass: "rwa_stock"}}}
	r.Normalize()
	v := NewResolver(r).Resolve(ResolveRequest{Exchange: "lighter", Symbol: "US100", MarketTypeHint: "perpetual"})
	if v.Market == nil || v.Market.BaseAsset != "US100" || v.Market.UnitMultiplier != 0 {
		t.Fatalf("index scaled: %+v", v)
	}
}
func TestScaledIdentityConversionAndCustomRebaseSurviveRepeatedNormalization(t *testing.T) {
	r := Registry{AssetAliases: []AssetAliasRule{{Canonical: "PEPE", AssetClass: "crypto", UnitAliases: []AssetUnitAlias{{Alias: "1000PEPE", Multiplier: 1000}}}}, MarketOverrides: []MarketOverride{{Exchange: "lighter", RawSymbol: "1000PEPE", MarketType: "perpetual", CanonicalSymbol: "PEPE/USDC", AssetClass: "crypto", AssetID: "crypto:ticker:1000pepe", UnderlyingID: "crypto:ticker:1000pepe", ComparisonKey: "crypto:ticker:1000pepe/USDC"}, {Exchange: "bybit", RawSymbol: "1000PEPEPERP", MarketType: "perpetual", CanonicalSymbol: "1000PEPE/USDC", AssetClass: "crypto"}, {Exchange: "test", RawSymbol: "SPLITUSDT", MarketType: "perpetual", CanonicalSymbol: "PEPE/USDT", AssetClass: "crypto", UnitMultiplier: 0.1}}}
	r.Normalize()
	r.Normalize()
	resolver := NewResolver(r)
	for _, q := range []ResolveRequest{{Exchange: "lighter", Symbol: "1000PEPE", MarketTypeHint: "perpetual"}, {Exchange: "bybit", Symbol: "1000PEPEPERP", MarketTypeHint: "perpetual"}} {
		v := resolver.Resolve(q)
		if v.Market == nil || v.Market.AssetID != "crypto:ticker:pepe" || v.Market.UnitMultiplier != 1000 || v.Market.CanonicalPriceMultiplier != 0.001 || v.Market.CanonicalQuantityMultiplier != 1000 {
			t.Fatalf("wrong unit identity: %+v", v)
		}
	}
	v := resolver.Resolve(ResolveRequest{Exchange: "test", Symbol: "SPLITUSDT", MarketTypeHint: "perpetual"})
	if v.Market == nil || v.Market.UnitMultiplier != 0.1 {
		t.Fatal(v)
	}
}
func TestReviewedExactMarketsAllResolveWithExpectedIdentity(t *testing.T) {
	reviewed, err := LoadReviewedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	all, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(all)
	for _, want := range reviewed.MarketOverrides {
		got := resolver.Resolve(ResolveRequest{Exchange: want.Exchange, Symbol: want.RawSymbol, MarketTypeHint: want.MarketType})
		m := got.Market
		if m == nil || m.AssetClass != want.AssetClass || m.AssetID != want.AssetID || m.UnderlyingID != want.UnderlyingID || m.CanonicalSymbol != strings.ToUpper(want.CanonicalSymbol) || m.UnitMultiplier != want.UnitMultiplier {
			t.Fatalf("reviewed market changed %s/%s: want %+v got %+v", want.Exchange, want.RawSymbol, want, got)
		}
	}
}

func TestHynixOrdinaryShareADRAndLeveragedETPNeverMerge(t *testing.T) {
	reg, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver(reg)
	get := func(ex, sym, kind string) *MarketIdentity {
		t.Helper()
		v := r.Resolve(ResolveRequest{Exchange: ex, Symbol: sym, MarketTypeHint: kind})
		if v.Market == nil {
			t.Fatalf("missing identity %s %s %+v", ex, sym, v)
		}
		return v.Market
	}
	xyz := get("hyperliquid", "xyz:SKHX", "perpetual")
	lighter := get("lighter", "SKHYNIXUSD", "perpetual")
	adr := get("lighter", "SKHY", "perpetual")
	etp := get("bitget", "RSKHXUSDT", "spot")
	if xyz.AssetID != lighter.AssetID || xyz.ComparisonKey != lighter.ComparisonKey {
		t.Fatal("same verified ordinary-share unit not normalized", xyz, lighter)
	}
	if adr.UnderlyingID == xyz.UnderlyingID || etp.UnderlyingID == xyz.UnderlyingID || etp.UnderlyingID == adr.UnderlyingID || etp.BaseAsset != "SKHX" {
		t.Fatal("different products merged", xyz, adr, etp)
	}
}

func TestScaledCryptoDoesNotInheritCollidingStockIdentity(t *testing.T) {
	reg := Registry{AssetAliases: []AssetAliasRule{{Canonical: "CAT", AssetClass: "rwa_stock", AssetID: "equity:ticker:cat", UnderlyingID: "equity:ticker:cat", UnitAliases: []AssetUnitAlias{{Alias: "1000CAT", Multiplier: 1000}}}}, MarketOverrides: []MarketOverride{{Exchange: "binance", RawSymbol: "1000CATUSDT", MarketType: "perpetual", CanonicalSymbol: "1000CAT/USDT", AssetClass: "crypto", AssetID: "crypto:ticker:1000cat", UnderlyingID: "crypto:ticker:1000cat", ComparisonKey: "crypto:ticker:1000cat/USDT"}, {Exchange: "test", RawSymbol: "1000CATUSDT", MarketType: "perpetual", CanonicalSymbol: "1000CAT/USDT", AssetClass: "crypto", AssetID: "crypto:custom:cat-product", UnderlyingID: "crypto:custom:cat-underlying", ComparisonKey: "custom-cat"}}}
	reg.Normalize()
	reg.Normalize()
	r := NewResolver(reg)
	v := r.Resolve(ResolveRequest{Exchange: "binance", Symbol: "1000CATUSDT", MarketTypeHint: "perpetual"}).Market
	if v == nil || v.BaseAsset != "CAT" || v.AssetID != "crypto:ticker:cat" || v.UnderlyingID != v.AssetID || v.ComparisonKey != "crypto:ticker:cat/USDT" || v.UnitMultiplier != 1000 {
		t.Fatal(v)
	}
	custom := r.Resolve(ResolveRequest{Exchange: "test", Symbol: "1000CATUSDT", MarketTypeHint: "perpetual"}).Market
	if custom == nil || custom.AssetID != "crypto:custom:cat-product" || custom.UnderlyingID != "crypto:custom:cat-underlying" || custom.ComparisonKey != "custom-cat" {
		t.Fatal("custom product overwritten", custom)
	}
}

func TestXYZ100IndexRetainsNativeNameAndOneIndexUnit(t *testing.T) {
	reg, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	reg.AssetAliases = append(reg.AssetAliases, AssetAliasRule{Canonical: "XYZ", AssetClass: "rwa_stock", UnitAliases: []AssetUnitAlias{{Alias: "XYZ100", Multiplier: 100}}})
	reg.Normalize()
	reg.Normalize()
	m := NewResolver(reg).Resolve(ResolveRequest{Exchange: "hyperliquid", Symbol: "xyz:XYZ100", MarketTypeHint: "perpetual"}).Market
	if m == nil || m.BaseAsset != "XYZ100" || m.AssetID != "equity:index:trade-xyz:xyz100" || m.UnitMultiplier != 1 || m.CanonicalPriceMultiplier != 1 || m.ComparisonStatus != ComparisonAmbiguous {
		t.Fatal(m)
	}
}

func TestBitgetBZStockNeverInheritsBrentCommodityIdentity(t *testing.T) {
	reg, err := LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver(reg)
	stock := r.Resolve(ResolveRequest{Exchange: "bitget", Symbol: "RBZUSDT", MarketTypeHint: "spot"}).Market
	oil := r.Resolve(ResolveRequest{Exchange: "hyperliquid", Symbol: "xyz:BRENTOIL", MarketTypeHint: "perpetual"}).Market
	if stock == nil || oil == nil || stock.AssetClass != "rwa_stock" || stock.AssetID != "equity:isin:us48553t1060" || stock.UnderlyingID == oil.UnderlyingID || stock.ComparisonKey == oil.ComparisonKey || stock.ComparisonStatus != ComparisonAmbiguous {
		t.Fatal(stock, oil)
	}
}
