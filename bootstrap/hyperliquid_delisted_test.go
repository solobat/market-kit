package bootstrap

import (
	"context"
	"encoding/json"
	"github.com/solobat/market-kit/discovery"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHyperliquidCoreAndHIP3PreserveDelistedStatus(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var q map[string]any
		if err := json.NewDecoder(req.Body).Decode(&q); err != nil {
			t.Fatal(err)
		}
		body := `{"universe":[{"name":"OLD","isDelisted":true},{"name":"BTC","isDelisted":false}]}`
		if q["type"] == "perpDexs" {
			body = `[{"name":"io"}]`
		}
		if q["type"] == "metaAndAssetCtxs" {
			body = `[{"universe":[{"name":"io:SKHX","isDelisted":true},{"name":"io:ANTH"}]},[]]`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	rows, err := fetchHyperliquid(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(rows)
	var restored []discovery.ImportedMarket
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored) != 4 {
		t.Fatal(restored)
	}
	for _, row := range restored {
		inactive := row.Symbol == "OLD" || row.Symbol == "io:SKHX"
		if row.InstrumentMetadata == nil || row.InstrumentMetadata.IsDelisted != inactive || row.Status != hyperliquidMarketStatus(inactive) {
			t.Fatal(row)
		}
	}
}
