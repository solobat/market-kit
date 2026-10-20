package identity

import "testing"

func TestNHPConversionAndTaiwanStockProductsRetainBoundaries(t *testing.T) {
	reg, e := LoadDefaultRegistry()
	if e != nil {
		t.Fatal(e)
	}
	r := NewResolver(reg)
	get := func(sym, kind string) *MarketIdentity {
		v := r.Resolve(ResolveRequest{Exchange: "bitget", Symbol: sym, MarketTypeHint: kind})
		if v.Market == nil {
			t.Fatal(v)
		}
		return v.Market
	}
	nhp := get("RNHPUSDT", "spot")
	if nhp.AssetID != "equity:bitget:rtoken:nhp" || nhp.UnderlyingID != "equity:isin:us42226b6002" || nhp.InstrumentKind != "tokenized_stock" || nhp.ComparisonStatus != ComparisonAmbiguous || nhp.UnitMultiplier != 1 || nhp.CanonicalQuantityMultiplier != 1 {
		t.Fatal(nhp)
	}
	tw := get("TWTSMUSDT", "perpetual")
	adr := get("RTSMUSDT", "spot")
	if tw.BaseAsset != "TWTSM" || tw.AssetClass != "rwa_stock" || tw.ComparisonStatus != ComparisonAmbiguous || tw.UnderlyingID == adr.UnderlyingID || tw.ComparisonKey == adr.ComparisonKey {
		t.Fatal(tw, adr)
	}
	mdt := get("MDTUSDT", "perpetual")
	if mdt.AssetID != "equity:bitget:perpetual:mdt" || mdt.AssetClass != "rwa_stock" || mdt.ComparisonStatus != ComparisonAmbiguous {
		t.Fatal(mdt)
	}
}
