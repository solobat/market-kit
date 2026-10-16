package server

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

type consoleResponse struct {
	Registry identity.Registry               `json:"registry"`
	Groups   []discovery.AssetCandidateGroup `json:"groups"`
	Symbols  []consoleSymbol                 `json:"symbols"`
	Total    int                             `json:"total"`
	Facets   map[string][]string             `json:"facets"`
}

func consoleRead(t *testing.T, a *App, query string) consoleResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/console?"+query, nil))
	if rec.Code != 200 {
		t.Fatalf("%d: %s", rec.Code, rec.Body.String())
	}
	var v consoleResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestConsolePaginationSearchAndFacets(t *testing.T) {
	a := testIdentityApp()
	first := consoleRead(t, a, "view=registry&limit=1")
	next := consoleRead(t, a, "view=registry&limit=1&page=2")
	if first.Total != 2 || len(first.Registry.AssetAliases) != 1 || len(next.Registry.AssetAliases) != 1 || first.Registry.AssetAliases[0].Canonical == next.Registry.AssetAliases[0].Canonical {
		t.Fatalf("pagination: %+v %+v", first, next)
	}
	if len(first.Registry.MarketOverrides) != 0 || len(first.Groups) != 0 || len(first.Symbols) != 0 {
		t.Fatal("registry page leaked other inventories")
	}
	result := consoleRead(t, a, "view=registry&q=spcx&class=rwa_stock")
	if result.Total != 1 || result.Registry.AssetAliases[0].Canonical != "SPCX" || len(result.Facets["assetClasses"]) != 2 {
		t.Fatalf("search/facets: %+v", result)
	}
	rules := consoleRead(t, a, "view=rules&q=xyz&limit=1")
	if rules.Total != 1 || rules.Registry.MarketOverrides[0].Exchange != "hyperliquid" {
		t.Fatalf("rules: %+v", rules)
	}
	for _, q := range []string{"page=-1", "limit=101", "limit=0", "page=oops", "view=bad"} {
		rec := httptest.NewRecorder()
		a.handleConsole(rec, httptest.NewRequest("GET", "/?"+q, nil))
		if rec.Code != 400 {
			t.Fatalf("accepted %s", q)
		}
	}
}
func TestConsolePresenceUsesFullInventoryAndInvalidates(t *testing.T) {
	a := testIdentityApp()
	v := consoleRead(t, a, "view=symbols&limit=1&platform=hyperliquid&presence=true&presence_platform=gate&presence_type=spot")
	if v.Total != 1 || len(v.Symbols) != 1 || v.Symbols[0].Platform != "hyperliquid" {
		t.Fatalf("presence: %+v", v)
	}
	old := a.consoleSnapshot()
	if old != a.consoleSnapshot() {
		t.Fatal("snapshot not reused")
	}
	reg := a.registry.Clone()
	reg.AssetAliases = append(reg.AssetAliases, identity.AssetAliasRule{Canonical: "OPENAI", AssetClass: "rwa_stock", Aliases: []string{"OAI"}})
	reg.Normalize()
	a.mu.Lock()
	a.registry = reg
	a.resolver = identity.NewResolver(reg)
	a.mu.Unlock()
	v = consoleRead(t, a, "view=registry&q=oai")
	if v.Total != 1 || v.Registry.AssetAliases[0].Canonical != "OPENAI" || a.consoleSnapshot() == old {
		t.Fatalf("stale registry: %+v", v)
	}
}
func TestConsoleDetailKeepsContractSourcesTogether(t *testing.T) {
	a := testIdentityApp()
	a.discoveryCachedAt = time.Now()
	a.discoveryCacheGroups = []discovery.AssetCandidateGroup{{GroupKey: "SPCX/USDT", CanonicalAsset: "SPCX", AssetClass: "rwa_stock", Markets: []discovery.CandidateMarket{{Exchange: "gate", RawSymbol: "SPCX_USDT", MarketType: "spot", CanonicalSymbol: "SPCX/USDT", BaseAsset: "SPCX", QuoteAsset: "USDT", AssetClass: "rwa_stock"}}}}
	v := consoleRead(t, a, "view=asset-detail&asset=SPCX&limit=1")
	if v.Total != 2 || len(v.Registry.AssetAliases) != 1 || len(v.Registry.MarketOverrides) != 1 || len(v.Groups) != 1 {
		t.Fatalf("detail: %+v", v)
	}
	if other := consoleRead(t, a, "view=asset-detail&asset=MISSING"); other.Total != 0 || len(other.Registry.AssetAliases) != 0 {
		t.Fatalf("unknown asset: %+v", other)
	}
	// Advancing the discovery timestamp also invalidates the browse snapshot.
	a.discoveryCache = discovery.ImportEnvelope{Items: []discovery.ImportedMarket{{PlatformID: "gate", Platform: "Gate", Symbol: "SPCX_USDT", MarketType: "spot", BaseAsset: "SPCX", QuoteAsset: "USDT"}}}
	old := a.consoleSnapshot()
	a.discoveryCachedAt = a.discoveryCachedAt.Add(time.Second)
	if a.consoleSnapshot() == old {
		t.Fatal("discovery revision not invalidated")
	}
}
func TestConsoleGroupPayloadIsBounded(t *testing.T) {
	a := testIdentityApp()
	g := discovery.AssetCandidateGroup{GroupKey: "SPCX/USDT", CanonicalAsset: "SPCX", AssetClass: "rwa_stock"}
	for i := 0; i < 1000; i++ {
		g.Markets = append(g.Markets, discovery.CandidateMarket{Exchange: "gate", RawSymbol: fmt.Sprintf("SPCX%d", i), BaseAsset: "SPCX", QuoteAsset: "USDT"})
	}
	a.discoveryCacheGroups = []discovery.AssetCandidateGroup{g}
	v := consoleRead(t, a, "view=groups&limit=1")
	if v.Total != 1 || len(v.Groups[0].Markets) != 3 || len(a.discoveryCacheGroups[0].Markets) != 1000 {
		t.Fatalf("unbounded/mutated group")
	}
}
func TestConsoleFullSnapshotPayloadAndLatency(t *testing.T) {
	file := os.Getenv("MARKET_KIT_CONSOLE_INVENTORY")
	if file == "" {
		t.Skip("optional full inventory measurement")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		Payload discovery.ImportEnvelope `json:"payload"`
	}
	if err = json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	a := testIdentityApp()
	regFile := os.Getenv("MARKET_KIT_CONSOLE_REGISTRY")
	if regFile != "" {
		reg, err := identity.LoadRegistryFile(regFile)
		if err != nil {
			t.Fatal(err)
		}
		a.registry = reg
		a.resolver = identity.NewResolver(reg)
	}
	a.discoveryCache = env.Payload
	a.discoveryCachedAt = time.Now()
	a.discoveryCacheSources = []SyncSource{{ID: "fixture"}}
	for _, legacy := range []string{"registry", "discovery"} {
		start := time.Now()
		rec := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/?source=all", nil)
		if legacy == "registry" {
			a.handleRuntimeRegistry(rec, request)
		} else {
			a.handleDiscoveryCurrent(rec, request)
		}
		t.Logf("legacy %s bytes=%d time=%s", legacy, rec.Body.Len(), time.Since(start))
		if rec.Code != 200 {
			t.Fatal(rec.Body.String())
		}
	}
	start := time.Now()
	a.consoleSnapshot()
	t.Logf("index build: %s, markets=%d", time.Since(start), len(env.Payload.Items))
	for _, q := range []string{"view=registry", "view=rules", "view=groups", "view=asset-detail&asset=OPENAI", "view=symbols&platform=binance&class=rwa_stock&presence=true&presence_platform=binance-web3"} {
		start = time.Now()
		rec := httptest.NewRecorder()
		a.handleConsole(rec, httptest.NewRequest("GET", "/?"+q, nil))
		t.Logf("%s bytes=%d time=%s", q, rec.Body.Len(), time.Since(start))
		if rec.Code != 200 || rec.Body.Len() > 500000 {
			t.Fatalf("payload budget %d, %d", rec.Code, rec.Body.Len())
		}
	}
}

func TestConsoleExportCoversAllFilteredPages(t *testing.T) {
	a := testIdentityApp()
	rec := httptest.NewRecorder()
	a.handleConsole(rec, httptest.NewRequest("GET", "/?view=symbols&limit=1&export=json&field=rawSymbol&platform=gate,hyperliquid", nil))
	var values []string
	if err := json.Unmarshal(rec.Body.Bytes(), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || rec.Header().Get("Content-Disposition") == "" {
		t.Fatalf("export lost pages: %s", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	a.handleConsole(rec, httptest.NewRequest("GET", "/?view=symbols&export=oops&field=rawSymbol", nil))
	if rec.Code != 400 {
		t.Fatal("invalid export accepted")
	}
}
func TestConsoleCurationRefreshesDiscoveryClassification(t *testing.T) {
	a := testIdentityApp()
	a.discoveryCachedAt = time.Now()
	a.discoveryCache = discovery.ImportEnvelope{Items: []discovery.ImportedMarket{{PlatformID: "gate", Symbol: "SPCX_USDT", MarketType: "spot", BaseAsset: "SPCX", QuoteAsset: "USDT"}}}
	before := consoleRead(t, a, "view=groups&q=SPCX")
	if before.Groups[0].AssetClass != "rwa_stock" {
		t.Fatal("fixture classification")
	}
	reg := a.registry.Clone()
	reg.AssetAliases[0].AssetClass = "crypto"
	reg.Normalize()
	a.mu.Lock()
	a.registry = reg
	a.resolver = identity.NewResolver(reg)
	a.mu.Unlock()
	after := consoleRead(t, a, "view=groups&q=SPCX")
	if after.Groups[0].AssetClass != "crypto" {
		t.Fatalf("stale discovery classification: %+v", after.Groups)
	}
}

func TestConsoleSyncCompactDoesNotReturnInventory(t *testing.T) {
	a := testIdentityApp()
	a.discoveryCache = discovery.ImportEnvelope{Source: "market-kit-all", Items: []discovery.ImportedMarket{{PlatformID: "gate", Symbol: "SPCX_USDT"}}}
	a.discoveryCachedAt = time.Now()
	a.discoveryCacheSources = []SyncSource{{ID: "fixture"}}
	for _, compact := range []bool{false, true} {
		query := "/?source=all"
		if compact {
			query += "&compact=1"
		}
		rec := httptest.NewRecorder()
		a.handleDiscoverySync(rec, httptest.NewRequest("GET", query, nil))
		if rec.Code != 200 {
			t.Fatal(rec.Body.String())
		}
		var payload struct {
			Payload map[string]json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		_, hasItems := payload.Payload["items"]
		if hasItems == compact {
			t.Fatalf("compact=%v inventory=%v", compact, hasItems)
		}
		if compact && string(payload.Payload["marketCount"]) != "1" {
			t.Fatal("missing compact count")
		}
	}
}
