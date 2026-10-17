package bootstrap

import (
	"context"
	"net/http"
	"testing"

	"github.com/solobat/market-kit/curation"
)

func TestOfficialReferenceSourcesIncludeEquitiesCommoditiesAndForexWithoutCryptoGuess(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case extendedReferenceURL:
			return jsonResponse(`{"status":"OK","data":[{"name":"SKHY-USD","type":"PERPETUAL","assetName":"SKHY","collateralAssetName":"USD","category":"RWA","subCategory":"Equity","active":true},{"name":"XPT-USD","type":"PERPETUAL","assetName":"XPT","collateralAssetName":"USD","category":"RWA","subCategory":"Commodity","active":true},{"name":"NVDA_24_5-USD","type":"PERPETUAL","assetName":"NVDA_24_5","collateralAssetName":"USD","category":"RWA","subCategory":"Equity","active":false}]}`), nil
		case gtradeReferenceURL:
			return jsonResponse(`{"groups":[{"name":"crypto"},{"name":"stocks-1"},{"name":"commodities-2"},{"name":"forex"}],"pairs":[{"from":"BTC","to":"USD","groupIndex":"0"},{"from":"FB","to":"USD","groupIndex":"1"},{"from":"XPD","to":"USD","groupIndex":"2"},{"from":"EUR","to":"USD","groupIndex":"3"}]}`), nil
		default:
			t.Fatal(req.URL)
			return nil, nil
		}
	})}
	rows, err := fetchExtendedReference(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].BaseAsset != "SKHY" || rows[2].BaseAsset != "NVDA_24_5" || rows[2].Status != "paused" {
		t.Fatal(rows)
	}
	pairs, err := fetchGTradeReference(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 4 || pairs[1].AssetClassHint != "stock" || pairs[2].AssetClassHint != "commodity" || pairs[3].AssetClassHint != "" || pairs[3].Status != "unknown" {
		t.Fatal(pairs)
	}
	registry := curation.BuildGeneratedRegistry(append(rows, pairs...))
	for _, v := range registry.MarketOverrides {
		if v.Exchange == "gtrade" && v.RawSymbol == "EUR/USD" && v.AssetClass == "crypto" {
			t.Fatal("forex became crypto", v)
		}
	}
}

func TestGRVTOfficialCategoriesBeatGenericCryptoAndKeepRawContract(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != grvtReferenceURL || req.Method != http.MethodPost {
			t.Fatal(req.URL, req.Method)
		}
		return jsonResponse(`{"result":[{"instrument":"ASTS_USDT_Perp","base":"ASTS","quote":"USDT","kind":"PERPETUAL","asset_class":"EQUITY","status":"ACTIVE"},{"instrument":"COPPER_USDT_Perp","base":"COPPER","quote":"USDT","kind":"PERPETUAL","asset_class":"COMMODITY","status":"ACTIVE"},{"instrument":"BTC_USDT_Perp","base":"BTC","quote":"USDT","kind":"PERPETUAL","asset_class":"CRYPTO","status":"ACTIVE"},{"instrument":"FUTURE","base":"X","quote":"USDT","kind":"FUTURE","asset_class":"EQUITY"}]}`), nil
	})}
	rows, err := fetchGRVTReference(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Symbol != "ASTS_USDT_Perp" || rows[0].AssetClassHint != "stock" || rows[1].AssetClassHint != "commodity" || rows[2].UnderlyingCategory != "coin" {
		t.Fatal(rows)
	}
	generic := rows[0]
	generic.AssetClassHint = "crypto"
	generic.Category = "crypto"
	generic.SourceID = "slipstream"
	registry := curation.BuildGeneratedRegistry(append(rows, generic))
	for _, v := range registry.MarketOverrides {
		if v.RawSymbol == "ASTS_USDT_Perp" && v.AssetClass != "rwa_stock" {
			t.Fatal("generic crypto overrode official equity", v)
		}
	}
}
