package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"strings"
	"testing"
)

func TestNewPerpInventoryAndETFIdentity(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case arcusMarketsURL:
			return jsonResponse(`{"markets":[{"marketDisplayName":"USO-USD","baseAsset":"USO","quoteAsset":"USD","type":"PERPETUAL","category":"COMMODITIES","status":"ONLINE"},{"marketDisplayName":"GLD-USD","baseAsset":"GLD","quoteAsset":"USD","type":"PERPETUAL","category":"COMMODITIES","status":"OFFLINE"},{"marketDisplayName":"BTC-USD","baseAsset":"BTC","quoteAsset":"USD","type":"SPOT"}]}`), nil
		case qfexRefdataURL:
			return jsonResponse(`{"data":[{"symbol":"CL-USD","status":"ACTIVE","product_category":"COMMODITY"},{"symbol":"AAPL-EUR","status":"INACTIVE","product_category":"EQUITY"}]}`), nil
		case qfexContractsURL:
			return jsonResponse(`{"data":[{"ticker_id":"CL-USD","base_currency":"CL","target_currency":"USD","product_type":"Perpetual"},{"ticker_id":"AAPL-EUR","base_currency":"AAPL","target_currency":"EUR","product_type":"Perpetual"}]}`), nil
		default:
			t.Fatalf("unexpected URL %s", req.URL)
			return nil, nil
		}
	})}
	envelope, err := Fetch(context.Background(), client, []string{"arcus", "qfex"})
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Items) != 4 {
		t.Fatalf("unexpected inventory %+v", envelope.Items)
	}
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	normalized := discovery.NewAggregator(registry).NormalizeImportedMarkets(envelope.Items)
	for _, row := range envelope.Items {
		if row.PlatformID == "arcus" && row.AssetClassHint != "etf" {
			t.Fatalf("ETF treated as commodity: %+v", row)
		}
		if row.Symbol == "GLD-USD" && row.Status != "paused" {
			t.Fatal("offline market marked live")
		}
		if row.Symbol == "AAPL-EUR" && row.QuoteAsset != "EUR" {
			t.Fatal("native quote lost")
		}
	}
	for _, row := range normalized {
		if row.Exchange == "arcus" && (row.AssetClass != "rwa_stock" || strings.Contains(row.CanonicalSymbol, "XAU") || strings.Contains(row.CanonicalSymbol, "CL/")) {
			t.Fatalf("ETF identity corrupted: %+v", row)
		}
	}
}

func TestNewPerpMalformedResponsesFail(t *testing.T) {
	for _, venue := range []string{"arcus", "qfex"} {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) { return jsonResponse(`{}`), nil })}
		if _, err := Fetch(context.Background(), client, []string{venue}); err == nil {
			t.Fatalf("%s accepted malformed response", venue)
		}
	}
}
