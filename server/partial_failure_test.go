package server

import (
	"context"
	"fmt"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMergedDiscoveryAndSyncExposePartialCoverage(t *testing.T) {
	app := &App{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "good.example" {
			return jsonResponse(`{"source":"slipstream","failedSources":["upstream-a"],"items":[{"platformId":"bybit","symbol":"BTCUSDT","venueType":"cex","marketType":"perpetual","baseAsset":"BTC","quoteAsset":"USDT","assetClassHint":"crypto"}]}`), nil
		}
		return nil, fmt.Errorf("upstream detail must not become public source ID")
	})}, sources: []SyncSource{{ID: "good", Kind: "discovery", URL: "https://good.example/data"}, {ID: "offline", Kind: "discovery", URL: "https://offline.example/data"}}}
	_, env, err := app.fetchDiscoveryLookupEnvelope(context.Background(), "all")
	if err != nil || len(env.Items) != 1 || !reflect.DeepEqual(env.FailedSources, []string{"good:upstream-a", "offline"}) {
		t.Fatalf("partial evidence lost: %+v %v", env, err)
	}
	base := identity.Registry{}
	app.baseRegistry = base
	app.registry = base
	app.resolver = identity.NewResolver(base)
	app.config = Config{AutoSyncSourceID: "all", RuntimeRegistryPath: filepath.Join(t.TempDir(), "runtime.json")}
	if err := app.runAutoSyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(app.currentAutoSyncStatus().LastDiscoveryFailedSources, env.FailedSources) {
		t.Fatal("sync status lost partial coverage")
	}
	app.autoSyncStatus.LastDiscoveryFailedSources = env.FailedSources
	status := app.currentAutoSyncStatus()
	status.LastDiscoveryFailedSources[0] = "changed"
	if app.currentAutoSyncStatus().LastDiscoveryFailedSources[0] != "good:upstream-a" {
		t.Fatal("status read mutated runtime state")
	}
}
