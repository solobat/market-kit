package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

func manualTestApp(t *testing.T) *App {
	t.Helper()
	base := identity.Registry{}
	return &App{config: Config{RuntimeRegistryPath: filepath.Join(t.TempDir(), "runtime.json"), AutoSyncSourceID: "test"}, baseRegistry: base, registry: base, resolver: identity.NewResolver(base), client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"source":"slipstream","items":[{"platformId":"test","venueType":"cex","marketType":"perp","symbol":"REVIEWUSDT","baseAsset":"REVIEW","quoteAsset":"USDT","assetClassHint":"crypto"}]}`), nil
	})}, sources: []SyncSource{{ID: "test", Kind: "discovery", URL: "https://example.invalid/inventory"}}}
}
func TestManualOverrideSurvivesSyncVersionChangeAndRestart(t *testing.T) {
	app := manualTestApp(t)
	_, err := app.commitManualOverrides([]registryOverrideRequest{{Exchange: "test", RawSymbol: "REVIEWUSDT", MarketType: "perpetual", CanonicalSymbol: "REVIEWED/USDT", AssetClass: "rwa_stock", UnitMultiplier: 0.1}})
	if err != nil {
		t.Fatal(err)
	}
	if err = app.runAutoSyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	check := func(a *App) {
		t.Helper()
		r := a.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: "test", Symbol: "REVIEWUSDT", MarketTypeHint: "perpetual"})
		if r.Market == nil || r.Market.CanonicalSymbol != "REVIEWED/USDT" || r.Market.UnitMultiplier != 0.1 || r.Market.AssetClass != "rwa_stock" {
			t.Fatalf("manual identity lost: %+v", r)
		}
	}
	check(app)
	// A generated version mismatch may discard generated state, never the sidecar.
	if err = os.WriteFile(app.config.RuntimeRegistryPath, []byte(`{"generated_version":1,"asset_aliases":[{"canonical":"OLD","asset_class":"crypto"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(app.config)
	if err != nil {
		t.Fatal(err)
	}
	check(restarted)
}
func TestBatchManualOverrideRejectsWholeInvalidBatchAndRequiresAdmin(t *testing.T) {
	app := manualTestApp(t)
	app.config.AdminCode = "test-only"
	body := `{"items":[{"exchange":"test","rawSymbol":"GOODUSDT","marketType":"spot","canonicalSymbol":"GOOD/USDT","assetClass":"crypto"},{"exchange":"test","rawSymbol":"BADUSDT","marketType":"spot","canonicalSymbol":"invalid"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides/batch", strings.NewReader(body))
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistryOverrideBatch(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatal(rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides/batch", strings.NewReader(body))
	req.Header.Set("X-Market-Kit-Admin-Code", "test-only")
	rec = httptest.NewRecorder()
	app.handleRuntimeRegistryOverrideBatch(rec, req)
	if rec.Code != http.StatusBadRequest || len(app.manualRegistry.MarketOverrides) != 0 {
		t.Fatalf("partial batch persisted: %d %+v", rec.Code, app.manualRegistry)
	}
	if _, err := os.Stat(manualRegistryPath(app.config)); !os.IsNotExist(err) {
		t.Fatalf("invalid batch wrote sidecar: %v", err)
	}
}
func TestConcurrentManualUpdatesKeepBothEntriesAndReaderSnapshot(t *testing.T) {
	app := manualTestApp(t)
	var wg sync.WaitGroup
	for _, symbol := range []string{"FIRST", "SECOND"} {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			for i := 0; i < 4; i++ {
				_, err := app.commitManualOverrides([]registryOverrideRequest{{Exchange: "test", RawSymbol: symbol + "USDT", MarketType: "spot", CanonicalSymbol: symbol + "/USDT", AssetClass: "crypto"}})
				if err != nil {
					t.Error(err)
				}
			}
		}(symbol)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			app.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: "test", Symbol: "FIRSTUSDT", MarketTypeHint: "spot"})
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			discovery.NewAggregator(app.runtimeRegistry()).BuildAssetGroups([]discovery.ImportedMarket{{PlatformID: "test", MarketType: "spot", Symbol: "FIRSTUSDT", BaseAsset: "FIRST", QuoteAsset: "USDT"}})
		}
	}()
	wg.Wait()
	if len(app.manualRegistry.MarketOverrides) != 2 {
		t.Fatalf("lost update: %+v", app.manualRegistry)
	}
	persisted, err := loadManualRegistry(app.config)
	if err != nil || len(persisted.MarketOverrides) != 2 {
		t.Fatalf("bad persistence: %+v %v", persisted, err)
	}
}
func TestBatchManualOverridesOneCommit(t *testing.T) {
	app := manualTestApp(t)
	items := []registryOverrideRequest{{Exchange: "test", RawSymbol: "AUSDT", MarketType: "spot", CanonicalSymbol: "A/USDT", AssetClass: "crypto"}, {Exchange: "test", RawSymbol: "BUSDT", MarketType: "spot", CanonicalSymbol: "B/USDT", AssetClass: "rwa_stock"}}
	payload, _ := json.Marshal(map[string]any{"items": items})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides/batch", strings.NewReader(string(payload)))
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistryOverrideBatch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}
	app.config.RequestTimeout = time.Second
	reloaded, err := New(app.config)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.manualRegistry.MarketOverrides) != 2 {
		t.Fatal("batch did not survive restart")
	}
}

func TestUpgradePreservesExplicitLegacyRebaseWithoutFreezingGeneratedFeed(t *testing.T) {
	app := manualTestApp(t)
	legacy := identity.Registry{GeneratedVersion: 1, AssetAliases: []identity.AssetAliasRule{{Canonical: "STALE", AssetClass: "crypto"}}, MarketOverrides: []identity.MarketOverride{{Exchange: "legacy", RawSymbol: "REBASEUSDT", MarketType: "perpetual", CanonicalSymbol: "OPENAI/USDT", AssetClass: "rwa_stock", UnitMultiplier: 0.1}}}
	payload, _ := json.Marshal(legacy)
	if err := os.WriteFile(app.config.RuntimeRegistryPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(app.config)
	if err != nil {
		t.Fatal(err)
	}
	v := reloaded.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: "legacy", Symbol: "REBASEUSDT", MarketTypeHint: "perpetual"})
	if v.Market == nil || v.Market.UnitMultiplier != 0.1 {
		t.Fatalf("lost legacy split/rebase: %+v", v)
	}
	if registryHasAsset(reloaded.runtimeRegistry(), "STALE") {
		t.Fatal("old generated aliases were frozen into manual state")
	}
	if _, err := os.Stat(manualRegistryPath(app.config)); err != nil {
		t.Fatal("recovered settings not durably migrated", err)
	}
}
func TestIdenticalManualBatchDoesNotRewriteSidecar(t *testing.T) {
	app := manualTestApp(t)
	q := []registryOverrideRequest{{Exchange: "test", RawSymbol: "NOOPUSDT", MarketType: "spot", CanonicalSymbol: "NOOP/USDT", AssetClass: "crypto"}}
	if _, err := app.commitManualOverrides(q); err != nil {
		t.Fatal(err)
	}
	timestamp := time.Unix(1000000000, 0)
	path := manualRegistryPath(app.config)
	if err := os.Chtimes(path, timestamp, timestamp); err != nil {
		t.Fatal(err)
	}
	if _, err := app.commitManualOverrides(q); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(path)
	if err != nil || !stat.ModTime().Equal(timestamp) {
		t.Fatalf("no-op batch rewrote registry: %v %v", stat, err)
	}
}

func TestLargeDiscoveryPayloadIsNotSilentlyTruncated(t *testing.T) {
	app := manualTestApp(t)
	payload := `{"source":"slipstream","items":[{"platformId":"test","symbol":"BIGUSDT","baseAsset":"BIG","quoteAsset":"USDT","marketType":"perp","venueType":"cex"}],"padding":"` + strings.Repeat("x", (4<<20)+1) + `"}`
	app.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(payload), nil })
	_, got, err := app.fetchDiscoveryEnvelope(context.Background(), "test")
	if err != nil || len(got.Items) != 1 {
		t.Fatalf("valid >4MiB discovery source lost: %v", err)
	}
	payload = `{"padding":"` + strings.Repeat("x", 32<<20) + `"}`
	_, _, err = app.fetchDiscoveryEnvelope(context.Background(), "test")
	if err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Fatalf("oversize must report explicit boundary: %v", err)
	}
}

func TestExplicitMarketClassDoesNotInheritOtherClassAssetID(t *testing.T) {
	app := manualTestApp(t)
	app.baseRegistry = identity.Registry{AssetAliases: []identity.AssetAliasRule{{Canonical: "SAME", AssetClass: "crypto"}}}
	app.applyGeneratedRegistry(identity.Registry{})
	_, err := app.commitManualOverrides([]registryOverrideRequest{{Exchange: "stocks", RawSymbol: "SAMEUSDT", MarketType: "spot", CanonicalSymbol: "SAME/USDT", AssetClass: "rwa_stock"}})
	if err != nil {
		t.Fatal(err)
	}
	r := app.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: "stocks", Symbol: "SAMEUSDT", MarketTypeHint: "spot"})
	if r.Market == nil || r.Market.AssetID != "equity:ticker:same" || r.Market.UnderlyingID != "equity:ticker:same" {
		t.Fatalf("stock inherited crypto identity: %+v", r)
	}
}

func TestManualClassUpdatePreservesExistingUnitAndSymbolAliases(t *testing.T) {
	app := manualTestApp(t)
	app.baseRegistry = identity.Registry{AssetAliases: []identity.AssetAliasRule{{Canonical: "PEPE", AssetClass: "crypto", Aliases: []string{"PEPEALT"}, UnitAliases: []identity.AssetUnitAlias{{Alias: "1000PEPE", Multiplier: 1000}}}}}
	app.applyGeneratedRegistry(identity.Registry{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/assets/PEPE", strings.NewReader(`{"assetClass":"crypto"}`))
	rec := httptest.NewRecorder()
	app.handleAsset(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}
	for _, symbol := range []string{"1000PEPEUSDT", "PEPEALTUSDT"} {
		r := app.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: "test", Symbol: symbol, MarketTypeHint: "spot"})
		if r.Market == nil || r.Market.BaseAsset != "PEPE" {
			t.Fatalf("alias lost after class edit: %+v", r)
		}
		if symbol == "1000PEPEUSDT" && r.Market.UnitMultiplier != 1000 {
			t.Fatal(r)
		}
	}
}
