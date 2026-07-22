package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/solobat/market-kit/bootstrap"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

func TestHandleResolveReturnsRuntimeIdentity(t *testing.T) {
	app := testIdentityApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resolve?exchange=gate&symbol=SPCX_USDT&marketType=spot", nil)
	rec := httptest.NewRecorder()
	app.handleResolve(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload identity.ResolveResult
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != identity.ResolveResolved {
		t.Fatalf("expected resolved status, got %+v", payload)
	}
	if payload.Market == nil || payload.Market.CanonicalSymbol != "SPCX/USDT" || payload.Market.AssetClass != "rwa_stock" {
		t.Fatalf("unexpected resolved market: %+v", payload.Market)
	}
}

func TestHandleResolveReturnsHyperliquidHIP3RawSymbol(t *testing.T) {
	app := testIdentityApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resolve?exchange=hyperliquid&symbol=SPCX&marketType=perpetual", nil)
	rec := httptest.NewRecorder()
	app.handleResolve(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload identity.ResolveResult
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != identity.ResolveResolved || payload.Market == nil {
		t.Fatalf("expected resolved HIP-3 market, got %+v", payload)
	}
	if payload.Market.RawSymbol != "xyz:SPCX" || payload.Market.CanonicalSymbol != "SPCX/USDT" {
		t.Fatalf("unexpected HIP-3 market: %+v", payload.Market)
	}
}

func TestHandleResolveAcceptsPostJSON(t *testing.T) {
	app := testIdentityApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/resolve", strings.NewReader(`{
		"exchange":"gate",
		"symbol":"SPCX_USDT",
		"marketType":"spot"
	}`))
	rec := httptest.NewRecorder()
	app.handleResolve(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload identity.ResolveResult
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != identity.ResolveResolved || payload.Market == nil || payload.Market.RawSymbol != "SPCX_USDT" {
		t.Fatalf("unexpected resolve response: %+v", payload)
	}
}

func TestHandleResolveBatchReturnsPerItemResults(t *testing.T) {
	app := testIdentityApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/resolve/batch", strings.NewReader(`{
		"items":[
			{"exchange":"gate","symbol":"SPCX_USDT","marketType":"spot"},
			{"exchange":"okx","symbol":"UNKNOWN"}
		]
	}`))
	rec := httptest.NewRecorder()
	app.handleResolveBatch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Summary struct {
			Count int `json:"count"`
		} `json:"summary"`
		Results []identity.ResolveResult `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Summary.Count != 2 || len(payload.Results) != 2 {
		t.Fatalf("unexpected batch response: %+v", payload)
	}
	if payload.Results[0].Status != identity.ResolveResolved {
		t.Fatalf("expected first item to resolve, got %+v", payload.Results[0])
	}
	if payload.Results[1].Status == identity.ResolveResolved {
		t.Fatalf("expected second item not to confidently resolve, got %+v", payload.Results[1])
	}
}

func TestHandleRuntimeRegistryReturnsMergedRegistry(t *testing.T) {
	app := testIdentityApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/registry", nil)
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload identity.Registry
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.AssetAliases) != 2 || len(payload.MarketOverrides) != 2 {
		t.Fatalf("unexpected registry payload: %+v", payload)
	}
}

func TestHandleRuntimeRegistryOverrideReassignsMarket(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "QNT", AssetClass: "crypto"},
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
	}
	base.Normalize()
	generated := identity.Registry{
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "hyperliquid", RawSymbol: "xyz:QNT", MarketType: "perpetual", CanonicalSymbol: "QNT/USDC"},
		},
	}
	generated.Normalize()
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	app := &App{
		config: Config{
			RuntimeRegistryPath: runtimePath,
		},
		baseRegistry:      base,
		generatedRegistry: generated,
		registry:          base.Merge(generated),
		resolver:          identity.NewResolver(base.Merge(generated)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides", strings.NewReader(`{
		"exchange":"hyperliquid",
		"rawSymbol":"xyz:QNT",
		"marketType":"perp",
		"canonicalSymbol":"QNTX/USDT",
		"assetClass":"crypto"
	}`))
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistryOverride(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	resolved := app.runtimeResolver().Resolve(identity.ResolveRequest{
		Exchange:       "hyperliquid",
		Symbol:         "xyz:QNT",
		MarketTypeHint: "perpetual",
	})
	if resolved.Status != identity.ResolveResolved || resolved.Market == nil {
		t.Fatalf("expected reassigned market to resolve, got %+v", resolved)
	}
	if resolved.Market.CanonicalSymbol != "QNTX/USDT" || resolved.Market.BaseAsset != "QNTX" {
		t.Fatalf("unexpected reassigned market: %+v", resolved.Market)
	}

	persisted, err := identity.LoadRegistryFile(runtimePath)
	if err != nil {
		t.Fatalf("load persisted registry: %v", err)
	}
	if !registryHasAsset(persisted, "QNTX") {
		t.Fatalf("expected QNTX asset in persisted runtime registry: %+v", persisted.AssetAliases)
	}
	matches := 0
	for _, item := range persisted.MarketOverrides {
		if item.Exchange == "hyperliquid" && item.RawSymbol == "xyz:QNT" && item.MarketType == "perpetual" {
			matches++
			if item.CanonicalSymbol != "QNTX/USDT" {
				t.Fatalf("expected persisted override to target QNTX/USDT, got %+v", item)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("expected exactly one persisted xyz:QNT override, got %d in %+v", matches, persisted.MarketOverrides)
	}
}

func TestHandleRuntimeRegistryOverrideCanSetRebaseMultiplier(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "OPENAI", AssetClass: "rwa_stock"},
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
	}
	base.Normalize()
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	app := &App{
		config: Config{
			RuntimeRegistryPath: runtimePath,
		},
		baseRegistry:      base,
		generatedRegistry: identity.Registry{},
		registry:          base,
		resolver:          identity.NewResolver(base),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides", strings.NewReader(`{
		"exchange":"okx",
		"rawSymbol":"OPENAI-USDT-SWAP",
		"marketType":"perp",
		"canonicalSymbol":"OPENAI/USDT",
		"unitMultiplier":0.1
	}`))
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistryOverride(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	resolved := app.runtimeResolver().Resolve(identity.ResolveRequest{
		Exchange: "okx",
		Symbol:   "OPENAI-USDT-SWAP",
	})
	if resolved.Status != identity.ResolveResolved || resolved.Market == nil {
		t.Fatalf("expected rebased market to resolve, got %+v", resolved)
	}
	if resolved.Market.CanonicalSymbol != "OPENAI/USDT" ||
		resolved.Market.UnitMultiplier != 0.1 ||
		resolved.Market.CanonicalPriceMultiplier != 10 ||
		resolved.Market.CanonicalQuantityMultiplier != 0.1 {
		t.Fatalf("expected rebase conversion to be preserved, got %+v", resolved.Market)
	}

	persisted, err := identity.LoadRegistryFile(runtimePath)
	if err != nil {
		t.Fatalf("load persisted registry: %v", err)
	}
	if len(persisted.MarketOverrides) != 1 || persisted.MarketOverrides[0].UnitMultiplier != 0.1 {
		t.Fatalf("expected persisted rebase multiplier, got %+v", persisted.MarketOverrides)
	}
}

func TestHandleRuntimeRegistryOverrideRebaseOverridesStaticMarket(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "OPENAI", AssetClass: "rwa_stock"},
		},
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "okx", RawSymbol: "OPENAI-USDT-SWAP", MarketType: "perpetual", CanonicalSymbol: "OPENAI/USDT"},
		},
	}
	base.Normalize()
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	app := &App{
		config: Config{
			RuntimeRegistryPath: runtimePath,
		},
		baseRegistry:      base,
		generatedRegistry: identity.Registry{},
		registry:          base,
		resolver:          identity.NewResolver(base),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides?compact=1", strings.NewReader(`{
		"exchange":"okx",
		"rawSymbol":"OPENAI-USDT-SWAP",
		"marketType":"perp",
		"canonicalSymbol":"OPENAI/USDT",
		"unitMultiplier":0.1
	}`))
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistryOverride(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	// Simulate the next process refresh by applying the persisted generated layer
	// over the static base registry.
	persisted, err := identity.LoadRegistryFile(runtimePath)
	if err != nil {
		t.Fatalf("load persisted registry: %v", err)
	}
	reloaded := &App{
		baseRegistry: base,
		registry:     base,
		resolver:     identity.NewResolver(base),
	}
	reloaded.applyGeneratedRegistry(persisted)

	resolved := reloaded.runtimeResolver().Resolve(identity.ResolveRequest{
		Exchange: "okx",
		Symbol:   "OPENAI-USDT-SWAP",
	})
	if resolved.Status != identity.ResolveResolved || resolved.Market == nil {
		t.Fatalf("expected rebased static market to resolve after reload, got %+v", resolved)
	}
	if resolved.Market.UnitMultiplier != 0.1 ||
		resolved.Market.CanonicalPriceMultiplier != 10 ||
		resolved.Market.CanonicalQuantityMultiplier != 0.1 {
		t.Fatalf("expected runtime rebase to override static market after reload, got %+v", resolved.Market)
	}
}

func TestHandleRuntimeRegistryOverrideCompactReturnsNormalizedOverride(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "OPENAI", AssetClass: "rwa_stock"},
		},
	}
	base.Normalize()
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	app := &App{
		config: Config{
			RuntimeRegistryPath: runtimePath,
		},
		baseRegistry:      base,
		generatedRegistry: identity.Registry{},
		registry:          base,
		resolver:          identity.NewResolver(base),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides?compact=1", strings.NewReader(`{
		"exchange":"okx",
		"rawSymbol":"OPENAI-USDT-SWAP",
		"marketType":"perp",
		"canonicalSymbol":"OPENAI/USDT",
		"unitAlias":"openai",
		"unitMultiplier":0.1
	}`))
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistryOverride(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Override identity.MarketOverride `json:"override"`
		Registry *identity.Registry      `json:"registry,omitempty"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Registry != nil {
		t.Fatalf("compact response should not include registry")
	}
	if payload.Override.MarketType != "perpetual" ||
		payload.Override.CanonicalSymbol != "OPENAI/USDT" ||
		payload.Override.UnitAlias != "OPENAI" ||
		payload.Override.UnitMultiplier != 0.1 {
		t.Fatalf("expected compact response to return normalized override, got %+v", payload.Override)
	}
}

func TestHandleRuntimeRegistryOverrideRequiresAssetClassForNewAsset(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
	}
	base.Normalize()
	app := &App{
		config: Config{
			RuntimeRegistryPath: filepath.Join(t.TempDir(), "runtime_generated_registry.json"),
		},
		baseRegistry:      base,
		generatedRegistry: identity.Registry{},
		registry:          base,
		resolver:          identity.NewResolver(base),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/registry/overrides", strings.NewReader(`{
		"exchange":"binance",
		"rawSymbol":"KORUUSDT",
		"marketType":"perp",
		"canonicalSymbol":"KORU/USDT"
	}`))
	rec := httptest.NewRecorder()
	app.handleRuntimeRegistryOverride(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing assetClass to be rejected, got status %d body=%s", rec.Code, rec.Body.String())
	}
	if registryHasAsset(app.runtimeRegistry(), "KORU") {
		t.Fatalf("expected KORU not to be inserted without explicit assetClass")
	}
}

func TestHandleAssetClassUpdateOverridesStaticAssetClass(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "OPENAI", AssetClass: "crypto"},
		},
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "okx", RawSymbol: "OPENAI-USDT-SWAP", MarketType: "perpetual", CanonicalSymbol: "OPENAI/USDT"},
		},
	}
	base.Normalize()
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	app := &App{
		config: Config{
			RuntimeRegistryPath: runtimePath,
		},
		baseRegistry:      base,
		generatedRegistry: identity.Registry{},
		registry:          base,
		resolver:          identity.NewResolver(base),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets/OPENAI?compact=1", strings.NewReader(`{"assetClass":"rwa_stock"}`))
	rec := httptest.NewRecorder()
	app.handleAssetClassUpdate(rec, req, "OPENAI")

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	resolved := app.runtimeResolver().Resolve(identity.ResolveRequest{
		Exchange: "okx",
		Symbol:   "OPENAI-USDT-SWAP",
	})
	if resolved.Status != identity.ResolveResolved || resolved.Market == nil {
		t.Fatalf("expected OPENAI market to resolve, got %+v", resolved)
	}
	if resolved.Market.AssetClass != "rwa_stock" {
		t.Fatalf("expected runtime asset class override, got %+v", resolved.Market)
	}

	persisted, err := identity.LoadRegistryFile(runtimePath)
	if err != nil {
		t.Fatalf("load persisted registry: %v", err)
	}
	if asset, ok := findAssetAlias(persisted, "OPENAI"); !ok || asset.AssetClass != "rwa_stock" {
		t.Fatalf("expected persisted runtime asset class override, got %+v", persisted.AssetAliases)
	}
}

func TestNewAppliesPersistedRuntimeAssetClassOverrides(t *testing.T) {
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	runtime := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "OPENAI", AssetClass: "rwa_stock"},
		},
	}
	runtime.Normalize()
	payload, err := json.MarshalIndent(runtime, "", "  ")
	if err != nil {
		t.Fatalf("encode runtime registry: %v", err)
	}
	if err := os.WriteFile(runtimePath, append(payload, '\n'), 0o644); err != nil {
		t.Fatalf("write runtime registry: %v", err)
	}

	app, err := New(Config{
		RuntimeRegistryPath: runtimePath,
		RequestTimeout:      time.Second,
		AutoSyncInterval:    time.Minute,
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	asset, ok := findAssetAlias(app.runtimeRegistry(), "OPENAI")
	if !ok {
		t.Fatalf("expected OPENAI asset alias")
	}
	if asset.AssetClass != "rwa_stock" {
		t.Fatalf("expected persisted runtime asset class to override static class, got %+v", asset)
	}
}

func TestHandlerAllowsConfiguredCORSPreflight(t *testing.T) {
	app := &App{
		config: Config{
			AllowedOrigins: []string{"https://console.example.com"},
		},
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/registry/overrides", nil)
	req.Header.Set("Origin", "https://console.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("expected POST in allow methods, got %q", got)
	}
}

func TestHandlerDoesNotAllowUnconfiguredCORSPreflight(t *testing.T) {
	app := &App{
		config: Config{
			AllowedOrigins: []string{"https://console.example.com"},
		},
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/registry/overrides", nil)
	req.Header.Set("Origin", "https://other.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
}

func TestAutoSyncUpdatesRuntimeRegistryFromSlipstream(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
	}
	base.Normalize()
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	app := &App{
		config: Config{
			AutoSyncEnabled:     true,
			AutoSyncInterval:    time.Minute,
			AutoSyncSourceID:    "slipstream-test",
			RuntimeRegistryPath: runtimePath,
		},
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://example.com/slipstream.json" {
					t.Fatalf("unexpected request: %s", req.URL.String())
				}
				return jsonResponse(`{
				  "source":"slipstream",
				  "generatedAt":"2026-05-19T00:00:00Z",
				  "items":[
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"NEW-USDT-SWAP","baseAsset":"NEW","quoteAsset":"USDT","assetClassHint":"stock","status":"live"}
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
				URL:     "https://example.com/slipstream.json",
			},
		},
		baseRegistry: base,
		registry:     base,
		resolver:     identity.NewResolver(base),
		autoSyncStatus: AutoSyncStatus{
			Enabled: true,
		},
	}

	if err := app.runAutoSyncOnce(context.Background()); err != nil {
		t.Fatalf("run auto-sync: %v", err)
	}

	result := app.runtimeResolver().Resolve(identity.ResolveRequest{
		Exchange:       "okx",
		Symbol:         "NEW-USDT-SWAP",
		MarketTypeHint: "perpetual",
	})
	if result.Status != identity.ResolveResolved || result.Market == nil {
		t.Fatalf("expected auto-synced market to resolve, got %+v", result)
	}
	if result.Market.AssetClass != "rwa_stock" || result.Market.CanonicalSymbol != "NEW/USDT" {
		t.Fatalf("unexpected auto-synced market: %+v", result.Market)
	}

	persisted, err := identity.LoadRegistryFile(runtimePath)
	if err != nil {
		t.Fatalf("load persisted runtime registry: %v", err)
	}
	if len(persisted.AssetAliases) != 2 || len(persisted.MarketOverrides) != 1 {
		t.Fatalf("unexpected persisted registry: %+v", persisted)
	}
}

func TestHandleAssetReturnsAliasRule(t *testing.T) {
	app := testIdentityApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/SPCX", nil)
	rec := httptest.NewRecorder()
	app.handleAsset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Asset identity.AssetAliasRule `json:"asset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Asset.Canonical != "SPCX" || payload.Asset.AssetClass != "rwa_stock" {
		t.Fatalf("unexpected asset payload: %+v", payload.Asset)
	}
}

func TestHandleAssetClassUpdateCorrectsRuntimeAsset(t *testing.T) {
	base := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
	}
	base.Normalize()
	generated := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "KORU", AssetClass: "crypto"},
		},
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "binance", RawSymbol: "KORUUSDT", MarketType: "perpetual", CanonicalSymbol: "KORU/USDT"},
		},
	}
	generated.Normalize()
	runtimePath := filepath.Join(t.TempDir(), "runtime_generated_registry.json")
	app := &App{
		config: Config{
			RuntimeRegistryPath: runtimePath,
		},
		baseRegistry:      base,
		generatedRegistry: generated,
		registry:          base.Merge(generated),
		resolver:          identity.NewResolver(base.Merge(generated)),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets/KORU", strings.NewReader(`{"assetClass":"rwa_stock"}`))
	rec := httptest.NewRecorder()
	app.handleAsset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	resolved := app.runtimeResolver().Resolve(identity.ResolveRequest{
		Exchange:       "binance",
		Symbol:         "KORUUSDT",
		MarketTypeHint: "perpetual",
	})
	if resolved.Market == nil || resolved.Market.AssetClass != "rwa_stock" {
		t.Fatalf("expected runtime resolver to use corrected asset class, got %+v", resolved)
	}
	persisted, err := identity.LoadRegistryFile(runtimePath)
	if err != nil {
		t.Fatalf("load persisted runtime registry: %v", err)
	}
	for _, asset := range persisted.AssetAliases {
		if asset.Canonical == "KORU" {
			if asset.AssetClass != "rwa_stock" {
				t.Fatalf("expected persisted KORU class to be rwa_stock, got %+v", asset)
			}
			return
		}
	}
	t.Fatalf("expected persisted KORU asset, got %+v", persisted.AssetAliases)
}

func TestHandleDiscoverySyncBuiltInBootstrap(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				payloads := map[string]string{
					"GET https://api.binance.com/api/v3/exchangeInfo":                                                                `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT"}]}`,
					"GET https://fapi.binance.com/fapi/v1/exchangeInfo":                                                              `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT"}]}`,
					"GET https://www.binance.com/bapi/defi/v1/public/wallet-direct/buw/wallet/market/token/rwa/stock/detail/list/ai": `{"data":{"list":[{"ticker":"AAPL","symbol":"AAPLONDO","quoteAsset":"USD","chainId":"1","contractAddress":"0xabc","type":1,"status":"TRADING"}]}}`,
					"GET https://api.bybit.com/v5/market/instruments-info?category=spot&limit=1000":                                  `{"result":{"list":[]}}`,
					"GET https://api.bybit.com/v5/market/instruments-info?category=linear&limit=1000":                                `{"result":{"list":[]}}`,
					"GET https://www.okx.com/api/v5/public/instruments?instType=SPOT":                                                `{"data":[]}`,
					"GET https://www.okx.com/api/v5/public/instruments?instType=SWAP":                                                `{"data":[]}`,
					"GET https://api.bitget.com/api/v2/spot/public/symbols":                                                          `{"data":[]}`,
					"GET https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES":                                `{"data":[]}`,
					"GET https://api.gateio.ws/api/v4/spot/currency_pairs":                                                           `[]`,
					"GET https://api.gateio.ws/api/v4/futures/usdt/contracts":                                                        `[]`,
				}
				key := req.Method + " " + req.URL.String()
				body, ok := payloads[key]
				if !ok {
					if key != "POST https://api.hyperliquid.xyz/info" {
						t.Fatalf("unexpected request: %s", key)
					}
					requestBody, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("read request body: %v", err)
					}
					bodyText := string(requestBody)
					switch {
					case strings.Contains(bodyText, `"type":"meta"`):
						body = `{"universe":[]}`
					case strings.Contains(bodyText, `"type":"perpDexs"`):
						body = `[]`
					default:
						t.Fatalf("unexpected hyperliquid request body: %s", bodyText)
					}
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
	if len(payload.Payload.Items) != 3 {
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

func TestHandleDiscoveryLookupUsesServerCache(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("lookup should use discovery cache, got upstream request: %s", req.URL.String())
				return nil, nil
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
		discoveryCacheSources: []SyncSource{
			{
				ID:      "slipstream-test",
				Label:   "Slipstream Test",
				Project: "slipstream",
				Kind:    "discovery",
				URL:     "https://example.com/discovery.json",
			},
		},
		discoveryCache: discovery.ImportEnvelope{
			Source:      discovery.SourceKind("slipstream"),
			GeneratedAt: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			Items: []discovery.ImportedMarket{
				{
					SourceID:   "slipstream",
					PlatformID: "bitget",
					Platform:   "Bitget",
					VenueType:  "cex",
					MarketType: string(identity.MarketTypeSpot),
					Symbol:     "PRESPCXUSDT",
					BaseAsset:  "SPCX",
					QuoteAsset: "USDT",
					Status:     "live",
				},
			},
		},
		discoveryCachedAt: time.Date(2026, 5, 15, 0, 1, 0, 0, time.UTC),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/lookup?source=slipstream-test&symbol=SPCX", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoveryLookup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		ServerCache bool `json:"serverCache"`
		Summary     struct {
			MarketCount int `json:"marketCount"`
		} `json:"summary"`
		Groups []struct {
			Markets []struct {
				RawSymbol string `json:"rawSymbol"`
			} `json:"markets"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.ServerCache {
		t.Fatalf("expected server cache marker")
	}
	if payload.Summary.MarketCount != 1 || len(payload.Groups) != 1 || len(payload.Groups[0].Markets) != 1 {
		t.Fatalf("expected cached market result, got %+v", payload)
	}
	if payload.Groups[0].Markets[0].RawSymbol != "PRESPCXUSDT" {
		t.Fatalf("unexpected market from cache: %+v", payload.Groups[0].Markets)
	}
}

func TestHandleDiscoveryLookupAggregatesAllDiscoverySources(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.String() {
				case "https://example.com/bootstrap.json":
					return jsonResponse(`{
					  "source":"market-kit-bootstrap",
					  "generatedAt":"2026-05-15T00:00:00Z",
					  "items":[
					    {"sourceId":"market-kit-bootstrap","platformId":"bitget","platform":"Bitget","venueType":"cex","marketType":"perp","symbol":"CBRSUSDT","baseAsset":"CBRS","quoteAsset":"USDT","status":"live"},
					    {"sourceId":"market-kit-bootstrap","platformId":"gate","platform":"Gate","venueType":"cex","marketType":"perp","symbol":"CBRS_USDT","baseAsset":"CBRS","quoteAsset":"USDT","status":"live"}
					  ]
					}`), nil
				case "https://example.com/slipstream.json":
					return jsonResponse(`{
					  "source":"slipstream",
					  "generatedAt":"2026-05-16T00:00:00Z",
					  "items":[
					    {"sourceId":"slipstream","platformId":"aster","platform":"Aster","venueType":"dex","marketType":"perp","symbol":"CBRSUSDT","baseAsset":"CBRS","quoteAsset":"USDT","status":"live"},
					    {"sourceId":"slipstream","platformId":"lighter","platform":"Lighter","venueType":"dex","marketType":"perp","symbol":"CBRS","baseAsset":"CBRS","quoteAsset":"USDC","status":"live"}
					  ]
					}`), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				}
				return nil, nil
			}),
		},
		sources: []SyncSource{
			{
				ID:      "bootstrap-test",
				Label:   "Bootstrap Test",
				Project: "market-kit",
				Kind:    "discovery",
				URL:     "https://example.com/bootstrap.json",
			},
			{
				ID:      "slipstream-test",
				Label:   "Slipstream Test",
				Project: "slipstream",
				Kind:    "discovery",
				URL:     "https://example.com/slipstream.json",
			},
			{
				ID:      "sample-test",
				Label:   "Sample Test",
				Project: "veridex",
				Kind:    "sample",
				URL:     "https://example.com/sample.json",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/lookup?source=all&symbol=CBRS", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoveryLookup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Source struct {
			ID string `json:"id"`
		} `json:"source"`
		Sources []struct {
			ID string `json:"id"`
		} `json:"sources"`
		Summary struct {
			GroupCount  int `json:"groupCount"`
			MarketCount int `json:"marketCount"`
		} `json:"summary"`
		Groups []struct {
			GroupKey string `json:"groupKey"`
			Markets  []struct {
				Exchange  string `json:"exchange"`
				RawSymbol string `json:"rawSymbol"`
			} `json:"markets"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Source.ID != "all" {
		t.Fatalf("expected aggregate source, got %s", payload.Source.ID)
	}
	if len(payload.Sources) != 2 {
		t.Fatalf("expected 2 discovery sources, got %+v", payload.Sources)
	}
	if payload.Summary.GroupCount != 2 || payload.Summary.MarketCount != 4 {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}

	foundAster := false
	foundLighter := false
	for _, group := range payload.Groups {
		for _, market := range group.Markets {
			if group.GroupKey == "CBRS/USDT" && market.Exchange == "aster" && market.RawSymbol == "CBRSUSDT" {
				foundAster = true
			}
			if group.GroupKey == "CBRS/USDC" && market.Exchange == "lighter" && market.RawSymbol == "CBRS" {
				foundLighter = true
			}
		}
	}
	if !foundAster || !foundLighter {
		t.Fatalf("expected aggregate lookup to include aster and lighter markets, got %+v", payload.Groups)
	}
}

func TestHandleDiscoverySyncAllSourcesReturnsMergedPayload(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.String() {
				case "https://example.com/bootstrap.json":
					return jsonResponse(`{
					  "source":"market-kit-bootstrap",
					  "generatedAt":"2026-05-16T00:00:00Z",
					  "items":[
					    {"sourceId":"market-kit-bootstrap","platformId":"binance-web3","platform":"Binance Web3","venueType":"web3","marketType":"spot","symbol":"MRVLon","baseAsset":"MRVL","quoteAsset":"USD","assetClassHint":"stock","status":"live"}
					  ]
					}`), nil
				case "https://example.com/slipstream.json":
					return jsonResponse(`{
					  "source":"slipstream",
					  "generatedAt":"2026-05-17T00:00:00Z",
					  "items":[
					    {"sourceId":"slipstream","platformId":"binance","platform":"Binance","venueType":"cex","marketType":"perp","symbol":"MRVLUSDT","baseAsset":"MRVL","quoteAsset":"USDT","assetClassHint":"stock","status":"live"}
					  ]
					}`), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				}
				return nil, nil
			}),
		},
		sources: []SyncSource{
			{
				ID:      "bootstrap-test",
				Label:   "Bootstrap Test",
				Project: "market-kit",
				Kind:    "discovery",
				URL:     "https://example.com/bootstrap.json",
			},
			{
				ID:      "slipstream-test",
				Label:   "Slipstream Test",
				Project: "slipstream",
				Kind:    "discovery",
				URL:     "https://example.com/slipstream.json",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/sync?source=all", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoverySync(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Source struct {
			ID string `json:"id"`
		} `json:"source"`
		Sources []struct {
			ID string `json:"id"`
		} `json:"sources"`
		Payload struct {
			Source string `json:"source"`
			Items  []struct {
				PlatformID string `json:"platformId"`
				Symbol     string `json:"symbol"`
			} `json:"items"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Source.ID != "all" {
		t.Fatalf("expected aggregate source, got %s", payload.Source.ID)
	}
	if len(payload.Sources) != 2 {
		t.Fatalf("expected 2 discovery sources, got %+v", payload.Sources)
	}
	if payload.Payload.Source != "market-kit-all" {
		t.Fatalf("expected merged payload source, got %s", payload.Payload.Source)
	}
	if len(payload.Payload.Items) != 2 {
		t.Fatalf("expected merged payload items, got %+v", payload.Payload.Items)
	}
}

func TestHandleDiscoverySyncUsesFreshServerCache(t *testing.T) {
	upstreamCalls := 0
	app := &App{
		config: Config{
			DiscoveryCacheTTL: time.Hour,
		},
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				upstreamCalls++
				t.Fatalf("sync should use fresh server cache, got upstream request: %s", req.URL.String())
				return nil, nil
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
		discoveryCacheSources: []SyncSource{
			{
				ID:      "slipstream-test",
				Label:   "Slipstream Test",
				Project: "slipstream",
				Kind:    "discovery",
				URL:     "https://example.com/discovery.json",
			},
		},
		discoveryCache: discovery.ImportEnvelope{
			Source:      discovery.SourceKind("slipstream"),
			GeneratedAt: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			Items: []discovery.ImportedMarket{
				{
					SourceID:   "slipstream",
					PlatformID: "bitget",
					Platform:   "Bitget",
					VenueType:  "cex",
					MarketType: string(identity.MarketTypeSpot),
					Symbol:     "PRESPCXUSDT",
					BaseAsset:  "SPCX",
					QuoteAsset: "USDT",
					Status:     "live",
				},
			},
		},
		discoveryCachedAt: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/sync?source=slipstream-test", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoverySync(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if upstreamCalls != 0 {
		t.Fatalf("expected no upstream calls, got %d", upstreamCalls)
	}

	var payload struct {
		ServerCache bool `json:"serverCache"`
		Payload     struct {
			Items []struct {
				Symbol string `json:"symbol"`
			} `json:"items"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.ServerCache {
		t.Fatalf("expected server cache marker")
	}
	if len(payload.Payload.Items) != 1 || payload.Payload.Items[0].Symbol != "PRESPCXUSDT" {
		t.Fatalf("unexpected cached payload: %+v", payload.Payload.Items)
	}
}

func TestHandleDiscoverySyncRequiresAdminCodeWhenConfigured(t *testing.T) {
	app := &App{
		config: Config{
			AdminCode: "secret",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/sync?source=all", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoverySync(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleDiscoveryCurrentBuildsServerCachedSnapshot(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.String() {
				case "https://example.com/bootstrap.json":
					return jsonResponse(`{
					  "source":"market-kit-bootstrap",
					  "generatedAt":"2026-05-16T00:00:00Z",
					  "items":[
					    {"sourceId":"market-kit-bootstrap","platformId":"binance-web3","platform":"Binance Web3","venueType":"web3","marketType":"spot","symbol":"MRVLon","baseAsset":"MRVL","quoteAsset":"USD","assetClassHint":"stock","status":"live"}
					  ]
					}`), nil
				case "https://example.com/slipstream.json":
					return jsonResponse(`{
					  "source":"slipstream",
					  "generatedAt":"2026-05-17T00:00:00Z",
					  "items":[
					    {"sourceId":"slipstream","platformId":"binance","platform":"Binance","venueType":"cex","marketType":"perp","symbol":"MRVLUSDT","baseAsset":"MRVL","quoteAsset":"USDT","assetClassHint":"stock","status":"live"}
					  ]
					}`), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				}
				return nil, nil
			}),
		},
		sources: []SyncSource{
			{ID: "bootstrap-test", Label: "Bootstrap Test", Project: "market-kit", Kind: "discovery", URL: "https://example.com/bootstrap.json"},
			{ID: "slipstream-test", Label: "Slipstream Test", Project: "slipstream", Kind: "discovery", URL: "https://example.com/slipstream.json"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/current", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoveryCurrent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		ServerCache bool `json:"serverCache"`
		Payload     struct {
			Source string `json:"source"`
			Items  []struct {
				PlatformID string `json:"platformId"`
				Symbol     string `json:"symbol"`
			} `json:"items"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.ServerCache {
		t.Fatalf("expected server cache marker")
	}
	if payload.Payload.Source != "market-kit-all" || len(payload.Payload.Items) != 2 {
		t.Fatalf("unexpected current payload: %+v", payload.Payload)
	}
}

func TestHandleDiscoveryLookupRecallsNamespacedHIP3MarketBySuffix(t *testing.T) {
	registry := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "SPCX", AssetClass: "rwa_stock"},
		},
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "gate", RawSymbol: "SPCX_USDT", MarketType: "spot", CanonicalSymbol: "SPCX/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "xyz:SPCX", MarketType: "perpetual", CanonicalSymbol: "SPCX/USDT"},
		},
	}
	registry.Normalize()
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://example.com/discovery.json" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				}
				return jsonResponse(`{
				  "source":"market-kit-bootstrap",
				  "generatedAt":"2026-05-18T00:00:00Z",
				  "items":[
				    {"sourceId":"market-kit-bootstrap","platformId":"gate","platform":"Gate","venueType":"cex","marketType":"spot","symbol":"SPCX_USDT","baseAsset":"SPCX","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"market-kit-bootstrap","platformId":"hyperliquid","platform":"Hyperliquid","venueType":"dex","marketType":"perp","symbol":"xyz:SPCX","baseAsset":"xyz:SPCX","quoteAsset":"USDC","status":"live"}
				  ]
				}`), nil
			}),
		},
		sources: []SyncSource{
			{
				ID:      "bootstrap-test",
				Label:   "Bootstrap Test",
				Project: "market-kit",
				Kind:    "discovery",
				URL:     "https://example.com/discovery.json",
			},
		},
		registry: registry,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/lookup?source=bootstrap-test&symbol=SPCX", nil)
	rec := httptest.NewRecorder()
	app.handleDiscoveryLookup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Summary struct {
			GroupCount  int `json:"groupCount"`
			MarketCount int `json:"marketCount"`
		} `json:"summary"`
		Groups []struct {
			GroupKey string `json:"groupKey"`
			Markets  []struct {
				Exchange        string `json:"exchange"`
				RawSymbol       string `json:"rawSymbol"`
				CanonicalSymbol string `json:"canonicalSymbol"`
			} `json:"markets"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Summary.GroupCount != 1 || payload.Summary.MarketCount != 2 {
		t.Fatalf("expected gate and HL SPCX markets in one published group, got summary %+v groups %+v", payload.Summary, payload.Groups)
	}

	foundGate := false
	foundHL := false
	for _, group := range payload.Groups {
		for _, market := range group.Markets {
			if group.GroupKey == "SPCX/USDT" && market.Exchange == "gate" && market.RawSymbol == "SPCX_USDT" && market.CanonicalSymbol == "SPCX/USDT" {
				foundGate = true
			}
			if group.GroupKey == "SPCX/USDT" && market.Exchange == "hyperliquid" && market.RawSymbol == "xyz:SPCX" && market.CanonicalSymbol == "SPCX/USDT" {
				foundHL = true
			}
		}
	}
	if !foundGate || !foundHL {
		t.Fatalf("expected lookup to recall gate SPCX and published HL xyz:SPCX together, got %+v", payload.Groups)
	}
}

func TestHandleDiscoveryLookupMatchesCanonicalAliasForHyperliquidTicker(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://example.com/discovery.json" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				}
				return jsonResponse(`{
				  "source":"slipstream",
				  "generatedAt":"2026-05-16T00:00:00Z",
				  "items":[
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"CL-USDT-SWAP","baseAsset":"CL","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"hyperliquid","platform":"Hyperliquid","venueType":"dex","marketType":"perp","symbol":"xyz:CL","baseAsset":"xyz:CL","quoteAsset":"USDC","status":"live"}
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

	req := httptest.NewRequest(http.MethodGet, "/api/discovery/lookup?source=slipstream-test&symbol=WTI", nil)
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
			GroupKey string `json:"groupKey"`
			Markets  []struct {
				Exchange  string `json:"exchange"`
				RawSymbol string `json:"rawSymbol"`
			} `json:"markets"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Query != "WTI" {
		t.Fatalf("unexpected query: %s", payload.Query)
	}
	if payload.Summary.GroupCount != 1 || payload.Summary.MarketCount != 2 {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}
	if len(payload.Groups) != 1 || payload.Groups[0].GroupKey != "CL/USDT" {
		t.Fatalf("unexpected groups: %+v", payload.Groups)
	}
	foundHL := false
	for _, market := range payload.Groups[0].Markets {
		if market.Exchange == "hyperliquid" && market.RawSymbol == "xyz:CL" {
			foundHL = true
			break
		}
	}
	if !foundHL {
		t.Fatalf("expected WTI lookup to surface hyperliquid ticker, got %+v", payload.Groups[0].Markets)
	}
}

func TestHandleDiscoveryLookupMatchesCommodityAliasesForHyperliquidTickers(t *testing.T) {
	app := &App{
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://example.com/discovery.json" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				}
				return jsonResponse(`{
				  "source":"slipstream",
				  "generatedAt":"2026-05-16T00:00:00Z",
				  "items":[
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"BZ-USDT-SWAP","baseAsset":"BZ","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"hyperliquid","platform":"Hyperliquid","venueType":"dex","marketType":"perp","symbol":"xyz:BRENTOIL","baseAsset":"xyz:BRENTOIL","quoteAsset":"USDC","status":"live"},
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"XAU-USDT-SWAP","baseAsset":"XAU","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"hyperliquid","platform":"Hyperliquid","venueType":"dex","marketType":"perp","symbol":"xyz:GOLD","baseAsset":"xyz:GOLD","quoteAsset":"USDC","status":"live"},
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"XAG-USDT-SWAP","baseAsset":"XAG","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"hyperliquid","platform":"Hyperliquid","venueType":"dex","marketType":"perp","symbol":"xyz:SILVER","baseAsset":"xyz:SILVER","quoteAsset":"USDC","status":"live"},
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"XPD-USDT-SWAP","baseAsset":"XPD","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"hyperliquid","platform":"Hyperliquid","venueType":"dex","marketType":"perp","symbol":"xyz:PALLADIUM","baseAsset":"xyz:PALLADIUM","quoteAsset":"USDC","status":"live"},
				    {"sourceId":"slipstream","platformId":"okx","platform":"OKX","venueType":"cex","marketType":"perp","symbol":"XPT-USDT-SWAP","baseAsset":"XPT","quoteAsset":"USDT","status":"live"},
				    {"sourceId":"slipstream","platformId":"hyperliquid","platform":"Hyperliquid","venueType":"dex","marketType":"perp","symbol":"xyz:PLATINUM","baseAsset":"xyz:PLATINUM","quoteAsset":"USDC","status":"live"}
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

	for _, tc := range []struct {
		query       string
		groupKey    string
		hlRawSymbol string
	}{
		{query: "BRENT", groupKey: "BZ/USDT", hlRawSymbol: "xyz:BRENTOIL"},
		{query: "GOLD", groupKey: "XAU/USDT", hlRawSymbol: "xyz:GOLD"},
		{query: "SILVER", groupKey: "XAG/USDT", hlRawSymbol: "xyz:SILVER"},
		{query: "PALLADIUM", groupKey: "XPD/USDT", hlRawSymbol: "xyz:PALLADIUM"},
		{query: "PLATINUM", groupKey: "XPT/USDT", hlRawSymbol: "xyz:PLATINUM"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/discovery/lookup?source=slipstream-test&symbol="+tc.query, nil)
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
					GroupKey string `json:"groupKey"`
					Markets  []struct {
						Exchange  string `json:"exchange"`
						RawSymbol string `json:"rawSymbol"`
					} `json:"markets"`
				} `json:"groups"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Query != tc.query {
				t.Fatalf("unexpected query: %s", payload.Query)
			}
			if payload.Summary.GroupCount != 1 || payload.Summary.MarketCount != 2 {
				t.Fatalf("unexpected summary for %s: %+v", tc.query, payload.Summary)
			}
			if len(payload.Groups) != 1 || payload.Groups[0].GroupKey != tc.groupKey {
				t.Fatalf("unexpected groups for %s: %+v", tc.query, payload.Groups)
			}
			foundHL := false
			for _, market := range payload.Groups[0].Markets {
				if market.Exchange == "hyperliquid" && market.RawSymbol == tc.hlRawSymbol {
					foundHL = true
					break
				}
			}
			if !foundHL {
				t.Fatalf("expected %s lookup to surface %s, got %+v", tc.query, tc.hlRawSymbol, payload.Groups[0].Markets)
			}
		})
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

func testIdentityApp() *App {
	registry := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "SPCX", AssetClass: "rwa_stock"},
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "gate", RawSymbol: "SPCX_USDT", MarketType: "spot", CanonicalSymbol: "SPCX/USDT"},
			{Exchange: "hyperliquid", RawSymbol: "xyz:SPCX", MarketType: "perpetual", CanonicalSymbol: "SPCX/USDT"},
		},
	}
	registry.Normalize()
	return &App{
		registry: registry,
		resolver: identity.NewResolver(registry),
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
