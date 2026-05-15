package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/solobat/market-kit/bootstrap"
)

func TestHandleDiscoverySyncBuiltInBootstrap(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				payloads := map[string]string{
					"GET https://api.binance.com/api/v3/exchangeInfo":                                 `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT"}]}`,
					"GET https://fapi.binance.com/fapi/v1/exchangeInfo":                               `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT"}]}`,
					"GET https://api.bybit.com/v5/market/instruments-info?category=spot&limit=1000":   `{"result":{"list":[]}}`,
					"GET https://api.bybit.com/v5/market/instruments-info?category=linear&limit=1000": `{"result":{"list":[]}}`,
					"GET https://www.okx.com/api/v5/public/instruments?instType=SPOT":                 `{"data":[]}`,
					"GET https://www.okx.com/api/v5/public/instruments?instType=SWAP":                 `{"data":[]}`,
					"GET https://api.bitget.com/api/v2/spot/public/symbols":                           `{"data":[]}`,
					"GET https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES": `{"data":[]}`,
					"GET https://api.gateio.ws/api/v4/spot/currency_pairs":                            `[]`,
					"GET https://api.gateio.ws/api/v4/futures/usdt/contracts":                         `[]`,
					"POST https://api.hyperliquid.xyz/info":                                           `{"universe":[]}`,
				}
				key := req.Method + " " + req.URL.String()
				body, ok := payloads[key]
				if !ok {
					t.Fatalf("unexpected request: %s", key)
				}
				return jsonResponse(body), nil
			}),
		},
		sources: []SyncSource{
			{
				ID:      bootstrap.BuiltInSourceID,
				Label:   bootstrap.BuiltInSourceLabel,
				Project: "market-kit",
				Kind:    "discovery",
				URL:     bootstrap.BuiltInSourceURL,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/sync?source="+bootstrap.BuiltInSourceID, nil)
	rec := httptest.NewRecorder()
	app.handleDiscoverySync(rec, req.WithContext(context.Background()))

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Source struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"source"`
		Payload struct {
			Source string `json:"source"`
			Items  []any  `json:"items"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Source.ID != bootstrap.BuiltInSourceID {
		t.Fatalf("unexpected source id: %s", payload.Source.ID)
	}
	if payload.Source.Kind != "discovery" {
		t.Fatalf("unexpected source kind: %s", payload.Source.Kind)
	}
	if payload.Payload.Source != "market-kit-bootstrap" {
		t.Fatalf("unexpected payload source: %s", payload.Payload.Source)
	}
	if len(payload.Payload.Items) != 2 {
		t.Fatalf("unexpected payload items: %d", len(payload.Payload.Items))
	}
}

func TestLoadSyncSourcesAlwaysIncludesBuiltInDiscoverySource(t *testing.T) {
	sources, err := loadSyncSources(Config{})
	if err != nil {
		t.Fatalf("loadSyncSources returned error: %v", err)
	}

	var found *SyncSource
	for i := range sources {
		if sources[i].ID == bootstrap.BuiltInSourceID {
			found = &sources[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("built-in discovery source missing")
	}
	if found.Kind != "discovery" {
		t.Fatalf("unexpected kind: %s", found.Kind)
	}
}

func TestHandleDiscoveryLookupReturnsMatchedGroups(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://example.com/discovery.json" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				}
				return jsonResponse(`{
				  "source":"slipstream",
				  "generatedAt":"2026-05-15T00:00:00Z",
				  "items":[
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"DRAM-USDT-SWAP","baseAsset":"DRAM","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"bitget","platform":"Bitget","venueType":"cex","marketType":"spot","symbol":"DRAMUSDT","baseAsset":"DRAM","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"binance","platform":"Binance","venueType":"cex","marketType":"spot","symbol":"BTCUSDT","baseAsset":"BTC","quoteAsset":"USDT","status":"live"}
				  ]
				}`), nil
			}),
		},
		sources: []SyncSource{
			{
				ID:      "slipstream-test",
				Label:   "Slipstream Test",
				Project: "slipstream",
				Kind:    "discovery",
				URL:     "https://example.com/discovery.json",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/lookup?source=slipstream-test&symbol=DRAM", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoveryLookup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Query   string `json:"query"`
		Summary struct {
			GroupCount  int `json:"groupCount"`
			MarketCount int `json:"marketCount"`
		} `json:"summary"`
		Groups []struct {
			GroupKey  string   `json:"groupKey"`
			Exchanges []string `json:"exchanges"`
			Markets   []struct {
				RawSymbol string `json:"rawSymbol"`
			} `json:"markets"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Query != "DRAM" {
		t.Fatalf("unexpected query: %s", payload.Query)
	}
	if payload.Summary.GroupCount != 1 {
		t.Fatalf("expected 1 group, got %d", payload.Summary.GroupCount)
	}
	if payload.Summary.MarketCount != 2 {
		t.Fatalf("expected 2 markets, got %d", payload.Summary.MarketCount)
	}
	if len(payload.Groups) != 1 || payload.Groups[0].GroupKey != "DRAM/USDT" {
		t.Fatalf("unexpected groups: %+v", payload.Groups)
	}
	if len(payload.Groups[0].Exchanges) != 2 {
		t.Fatalf("expected 2 exchanges, got %+v", payload.Groups[0].Exchanges)
	}
}

func TestDefaultDiscoverySourceIDPrefersBuiltIn(t *testing.T) {
	app := &App{
		sources: []SyncSource{
			{ID: "slipstream-test", Kind: "discovery"},
			{ID: bootstrap.BuiltInSourceID, Kind: "discovery"},
			{ID: "veridex-test", Kind: "sample"},
		},
	}

	if got := app.defaultDiscoverySourceID(); got != bootstrap.BuiltInSourceID {
		t.Fatalf("expected built-in discovery source, got %s", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
