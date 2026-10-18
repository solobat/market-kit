package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"net/http"
	"testing"
)

func TestBitgetIPOStockMetadataDoesNotRequireRealityOrStripPrefix(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v3/market/instruments" {
			return jsonResponse(`{"data":[{"symbol":"PRESPCXUSDT","baseCoin":"preSPCX","quoteCoin":"USDT","symbolType":"stock","isReality":"no","status":"online"},{"symbol":"RAAPLUSDT","baseCoin":"rAAPL","quoteCoin":"USDT","symbolType":"stock","isReality":"yes","status":"online"}]}`), nil
		}
		return jsonResponse(`{"data":[]}`), nil
	})}
	rows, err := fetchBitget(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].BaseAsset != "preSPCX" || rows[0].AssetClassHint != "stock" || rows[1].BaseAsset != "AAPL" {
		t.Fatal(rows)
	}
	reg := curation.BuildGeneratedRegistry(rows)
	for _, v := range reg.MarketOverrides {
		if v.AssetClass != "rwa_stock" {
			t.Fatal(v)
		}
	}
}
