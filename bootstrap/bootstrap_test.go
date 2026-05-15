package bootstrap

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFetchDefaultBuildsImportEnvelope(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			key := req.Method + " " + req.URL.String()
			payloads := map[string]string{
				"GET https://api.binance.com/api/v3/exchangeInfo":                                 `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT"}]}`,
				"GET https://fapi.binance.com/fapi/v1/exchangeInfo":                               `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT"}]}`,
				"GET https://api.bybit.com/v5/market/instruments-info?category=spot&limit=1000":   `{"result":{"list":[{"symbol":"ETHUSDT","status":"Trading","baseCoin":"ETH","quoteCoin":"USDT"}]}}`,
				"GET https://api.bybit.com/v5/market/instruments-info?category=linear&limit=1000": `{"result":{"list":[{"symbol":"AAPLUSDT","status":"Trading","baseCoin":"AAPL","quoteCoin":"USDT"}]}}`,
				"GET https://www.okx.com/api/v5/public/instruments?instType=SPOT":                 `{"data":[{"instId":"SOL-USDT","baseCcy":"SOL","quoteCcy":"USDT","state":"live"}]}`,
				"GET https://www.okx.com/api/v5/public/instruments?instType=SWAP":                 `{"data":[{"instId":"CL-USDT-SWAP","baseCcy":"CL","quoteCcy":"USDT","state":"live"}]}`,
				"GET https://api.bitget.com/api/v2/spot/public/symbols":                           `{"data":[{"symbol":"DOGEUSDT","baseCoin":"DOGE","quoteCoin":"USDT","symbolStatus":"online"}]}`,
				"GET https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES": `{"data":[{"symbol":"NGUSDT","baseCoin":"NG","quoteCoin":"USDT","symbolStatus":"normal"}]}`,
				"GET https://api.gateio.ws/api/v4/spot/currency_pairs":                            `[{"id":"XRP_USDT","base":"XRP","quote":"USDT","trade_status":"tradable"},{"id":"BTC3L_USDT","base":"BTC3L","quote":"USDT","trade_status":"tradable"}]`,
				"GET https://api.gateio.ws/api/v4/futures/usdt/contracts":                         `[{"name":"NATGAS_USDT","quanto_base":"NATGAS","settle":"USDT","in_delisting":false}]`,
				"POST https://api.hyperliquid.xyz/info":                                           `{"universe":[{"name":"HYPE"}]}`,
			}
			body, ok := payloads[key]
			if !ok {
				t.Fatalf("unexpected request: %s", key)
			}
			return jsonResponse(body), nil
		}),
	}

	envelope, err := FetchDefault(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchDefault returned error: %v", err)
	}

	if envelope.Source != "market-kit-bootstrap" {
		t.Fatalf("unexpected source: %s", envelope.Source)
	}
	if len(envelope.Items) != 11 {
		t.Fatalf("unexpected item count: %d", len(envelope.Items))
	}

	found := map[string]bool{}
	for _, item := range envelope.Items {
		found[item.PlatformID+":"+item.Symbol+":"+item.MarketType] = true
		if item.SourceID != BuiltInSourceID {
			t.Fatalf("unexpected source id: %s", item.SourceID)
		}
	}

	for _, key := range []string{
		"binance:BTCUSDT:spot",
		"binance:BTCUSDT:perp",
		"bybit:AAPLUSDT:perp",
		"okx:CL-USDT-SWAP:perp",
		"bitget:NGUSDT:perp",
		"gate:NATGAS_USDT:perp",
		"hyperliquid:HYPE:perp",
	} {
		if !found[key] {
			t.Fatalf("missing expected market %s", key)
		}
	}
	if found["gate:BTC3L_USDT:spot"] {
		t.Fatalf("expected leveraged gate token to be filtered")
	}
}

func TestFetchSubset(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host != "api.hyperliquid.xyz" {
				t.Fatalf("unexpected host: %s", req.URL.Host)
			}
			return jsonResponse(`{"universe":[{"name":"PURR"}]}`), nil
		}),
	}

	envelope, err := Fetch(context.Background(), client, []string{"hyperliquid"})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(envelope.Items) != 1 {
		t.Fatalf("unexpected item count: %d", len(envelope.Items))
	}
	if envelope.Items[0].PlatformID != "hyperliquid" {
		t.Fatalf("unexpected platform: %s", envelope.Items[0].PlatformID)
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
