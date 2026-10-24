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

func TestBybitDatedFutureSurvivesImport(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"list":[{"symbol":"BTCUSDT-13NOV26","baseCoin":"BTC","quoteCoin":"USDT","contractType":"LinearFutures","deliveryTime":"1794556800000","settleCoin":"USDT"},{"symbol":"BTCUSDT","baseCoin":"BTC","quoteCoin":"USDT","contractType":"LinearPerpetual","deliveryTime":"0","settleCoin":"USDT"}]}}`))}, nil
	})}
	rows, err := fetchBybitCategory(context.Background(), client, "linear", "perp")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(rows)
	var restored []discovery.ImportedMarket
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored) != 2 || restored[0].MarketType != "future" || restored[1].MarketType != "perp" {
		t.Fatal(restored)
	}
	m := restored[0].InstrumentMetadata
	if m == nil || m.ExpiryAtMs != 1794556800000 || m.SettlementAsset != "USDT" || m.ContractType != "LinearFutures" {
		t.Fatal(m)
	}
}
