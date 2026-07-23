package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/solobat/market-kit/discovery"
)

func TestFetchDefaultBuildsImportEnvelope(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			key := req.Method + " " + req.URL.String()
			payloads := map[string]string{
				"GET https://api.binance.com/api/v3/exchangeInfo?permissions=SPOT&symbolStatus=TRADING":                          `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT","permissionSets":[["SPOT","MARGIN"]]},{"symbol":"HALTUSDT","status":"HALT","baseAsset":"HALT","quoteAsset":"USDT","permissionSets":[["SPOT"]]},{"symbol":"MARGINUSDT","status":"TRADING","baseAsset":"MARGIN","quoteAsset":"USDT","permissionSets":[["MARGIN"]]}]}`,
				"GET https://fapi.binance.com/fapi/v1/exchangeInfo":                                                              `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT","underlyingType":"COIN","underlyingSubType":["Layer-1"],"contractType":"PERPETUAL"},{"symbol":"KORUUSDT","status":"TRADING","baseAsset":"KORU","quoteAsset":"USDT","underlyingType":"EQUITY","underlyingSubType":["TradFi"],"contractType":"TRADIFI_PERPETUAL"}]}`,
				"GET https://www.binance.com/bapi/defi/v1/public/wallet-direct/buw/wallet/market/token/rwa/stock/detail/list/ai": `{"data":{"list":[{"ticker":"AAPL","symbol":"AAPLONDO","quoteAsset":"USD","chainId":"1","contractAddress":"0xabc","type":1,"status":"TRADING","externalUrl":"https://www.binance.com/en/web3"}]}}`,
				"GET https://api.bybit.com/v5/market/instruments-info?category=spot&limit=1000":                                  `{"result":{"list":[{"symbol":"ETHUSDT","status":"Trading","baseCoin":"ETH","quoteCoin":"USDT"}]}}`,
				"GET https://api.bybit.com/v5/market/instruments-info?category=linear&limit=1000":                                `{"result":{"list":[{"symbol":"AAPLUSDT","status":"Trading","baseCoin":"AAPL","quoteCoin":"USDT"},{"symbol":"AEHRUSDT","status":"Trading","baseCoin":"AEHR","quoteCoin":"USDT","symbolType":"stock"}]}}`,
				"GET https://www.okx.com/api/v5/public/instruments?instType=SPOT":                                                `{"data":[{"instId":"SOL-USDT","baseCcy":"SOL","quoteCcy":"USDT","state":"live","preDelisting":"true"}]}`,
				"GET https://www.okx.com/api/v5/public/instruments?instType=SWAP":                                                `{"data":[{"instId":"CL-USDT-SWAP","baseCcy":"CL","quoteCcy":"USDT","state":"live"}]}`,
				"GET https://api.bitget.com/api/v3/market/instruments?category=SPOT":                                             `{"data":[{"symbol":"DOGEUSDT","category":"SPOT","baseCoin":"DOGE","quoteCoin":"USDT","symbolType":"crypto","status":"online","isReality":"no","st":true},{"symbol":"RAAPLUSDT","category":"SPOT","baseCoin":"rAAPL","quoteCoin":"USDT","symbolType":"stock","status":"online","isReality":"yes"}]}`,
				"GET https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES":                                `{"data":[{"symbol":"NGUSDT","baseCoin":"NG","quoteCoin":"USDT","symbolStatus":"normal"}]}`,
				"GET https://api.gateio.ws/api/v4/spot/currency_pairs":                                                           `[{"id":"XRP_USDT","base":"XRP","quote":"USDT","trade_status":"tradable"},{"id":"BTC3L_USDT","base":"BTC3L","quote":"USDT","trade_status":"tradable"}]`,
				"GET https://api.gateio.ws/api/v4/futures/usdt/contracts":                                                        `[{"name":"NATGAS_USDT","quanto_base":"NATGAS","settle":"USDT","in_delisting":true}]`,
			}
			if key == "POST https://api.hyperliquid.xyz/info" {
				body, _ := io.ReadAll(req.Body)
				var payload map[string]any
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("decode hyperliquid payload: %v", err)
				}
				switch payload["type"] {
				case "meta":
					return jsonResponse(`{"universe":[{"name":"HYPE"}]}`), nil
				case "perpDexs":
					return jsonResponse(`[{"name":"km"}]`), nil
				case "metaAndAssetCtxs":
					if payload["dex"] != "km" {
						t.Fatalf("unexpected hyperliquid dex payload: %s", string(body))
					}
					return jsonResponse(`[{"universe":[{"name":"USOIL"}]},[]]`), nil
				default:
					t.Fatalf("unexpected hyperliquid payload: %s", string(body))
				}
			}
			body, ok := payloads[key]
			if !ok {
				t.Fatalf("unexpected request: %s", key)
			}
			if req.URL.String() == binanceWeb3OndoStockURL {
				if req.Header.Get("Referer") == "" || !strings.Contains(req.Header.Get("User-Agent"), "Mozilla") {
					t.Fatalf("expected browser-like headers for binance web3 request: %+v", req.Header)
				}
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
	if len(envelope.Items) != 16 {
		t.Fatalf("unexpected item count: %d", len(envelope.Items))
	}

	found := map[string]bool{}
	for _, item := range envelope.Items {
		found[item.PlatformID+":"+item.Symbol+":"+item.MarketType] = true
		if item.PlatformID == "binance-web3" && item.AssetClassHint != "stock" {
			t.Fatalf("expected binance-web3 item to carry stock hint: %+v", item)
		}
		if item.PlatformID == "binance" && item.Symbol == "KORUUSDT" {
			if item.AssetClassHint != "equity" || item.UnderlyingCategory != "equity" {
				t.Fatalf("expected Binance TradFi perpetual to carry equity evidence: %+v", item)
			}
		}
		if item.PlatformID == "bitget" && item.Symbol == "RAAPLUSDT" {
			if item.MarketType != "spot" || item.BaseAsset != "AAPL" || item.AssetClassHint != "stock" {
				t.Fatalf("expected Bitget Reality stock to import as stock spot: %+v", item)
			}
		}
		if item.PlatformID == "bitget" && item.Symbol == "DOGEUSDT" {
			if !item.ST || !containsString(item.Flags, discovery.MarketFlagST) {
				t.Fatalf("expected Bitget st flag to be preserved: %+v", item)
			}
		}
		if item.PlatformID == "bybit" && item.Symbol == "AEHRUSDT" {
			if item.MarketType != "perp" || item.BaseAsset != "AEHR" || item.AssetClassHint != "stock" {
				t.Fatalf("expected Bybit stock contract to import as stock perp: %+v", item)
			}
		}
		if item.PlatformID == "gate" && item.Symbol == "NATGAS_USDT" {
			if !item.PreDelisting || !containsString(item.Flags, discovery.MarketFlagPreDelisting) {
				t.Fatalf("expected Gate pre-delisting flag to be preserved: %+v", item)
			}
		}
		if item.PlatformID == "okx" && item.Symbol == "SOL-USDT" {
			if !item.PreDelisting || !containsString(item.Flags, discovery.MarketFlagPreDelisting) {
				t.Fatalf("expected OKX pre-delisting flag to be preserved: %+v", item)
			}
		}
		if item.SourceID != BuiltInSourceID {
			t.Fatalf("unexpected source id: %s", item.SourceID)
		}
	}

	for _, key := range []string{
		"binance:BTCUSDT:spot",
		"binance:BTCUSDT:perp",
		"binance:KORUUSDT:perp",
		"binance-web3:AAPLONDO:spot",
		"bybit:AAPLUSDT:perp",
		"bybit:AEHRUSDT:perp",
		"okx:CL-USDT-SWAP:perp",
		"bitget:RAAPLUSDT:spot",
		"bitget:NGUSDT:perp",
		"gate:NATGAS_USDT:perp",
		"hyperliquid:HYPE:perp",
		"hyperliquid:km:USOIL:perp",
	} {
		if !found[key] {
			t.Fatalf("missing expected market %s", key)
		}
	}
	if found["gate:BTC3L_USDT:spot"] {
		t.Fatalf("expected leveraged gate token to be filtered")
	}
	if found["binance:HALTUSDT:spot"] {
		t.Fatalf("expected halted Binance spot symbol to be filtered")
	}
	if found["binance:MARGINUSDT:spot"] {
		t.Fatalf("expected non-spot Binance symbol to be filtered")
	}
}

func TestFetchSubset(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host != "api.hyperliquid.xyz" {
				t.Fatalf("unexpected host: %s", req.URL.Host)
			}
			body, _ := io.ReadAll(req.Body)
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode hyperliquid payload: %v", err)
			}
			switch payload["type"] {
			case "meta":
				return jsonResponse(`{"universe":[{"name":"PURR"}]}`), nil
			case "perpDexs":
				return jsonResponse(`[]`), nil
			default:
				t.Fatalf("unexpected hyperliquid payload: %s", string(body))
				return nil, nil
			}
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

func TestPermissionListUnmarshalAcceptsBinanceShapes(t *testing.T) {
	cases := map[string][]string{
		`[["SPOT","MARGIN"]]`: {"SPOT", "MARGIN"},
		`["SPOT","MARGIN"]`:   {"SPOT", "MARGIN"},
		`"SPOT"`:              {"SPOT"},
		`null`:                nil,
	}

	for payload, expected := range cases {
		var permissions permissionList
		if err := json.Unmarshal([]byte(payload), &permissions); err != nil {
			t.Fatalf("unmarshal %s: %v", payload, err)
		}
		if len(permissions) != len(expected) {
			t.Fatalf("unmarshal %s: expected %v, got %v", payload, expected, permissions)
		}
		for idx := range expected {
			if permissions[idx] != expected[idx] {
				t.Fatalf("unmarshal %s: expected %v, got %v", payload, expected, permissions)
			}
		}
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

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
