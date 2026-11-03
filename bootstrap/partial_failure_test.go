package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestPartialCollectorsRetainSuccessfulRowsAndIdentifyGaps(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://api.binance.com/api/v3/exchangeInfo?permissions=SPOT&symbolStatus=TRADING":
			return jsonResponse(`{"symbols":[]}`), nil
		case "https://fapi.binance.com/fapi/v1/exchangeInfo":
			return jsonResponse(`{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT","underlyingType":"COIN","contractType":"PERPETUAL"}]}`), nil
		default:
			return nil, fmt.Errorf("upstream diagnostic must not become public source ID")
		}
	})}
	env, err := Fetch(context.Background(), client, nil)
	if err != nil || len(env.Items) == 0 || len(env.FailedSources) == 0 {
		t.Fatalf("partial success lost: %+v %v", env, err)
	}
	found := false
	for _, id := range env.FailedSources {
		if id == "bybit" {
			found = true
		}
		if id == "binance" || strings.Contains(id, "diagnostic") {
			t.Fatal(id)
		}
	}
	if !found {
		t.Fatal("missing failed collector", env.FailedSources)
	}
}
