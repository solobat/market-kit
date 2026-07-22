package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/solobat/market-kit/bootstrap"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

type App struct {
	config                Config
	client                *http.Client
	sources               []SyncSource
	mu                    sync.RWMutex
	baseRegistry          identity.Registry
	generatedRegistry     identity.Registry
	registry              identity.Registry
	resolver              *identity.Resolver
	autoSyncStatus        AutoSyncStatus
	discoveryCache        discovery.ImportEnvelope
	discoveryCacheSources []SyncSource
	discoveryCacheGroups  []discovery.AssetCandidateGroup
	discoveryCachedAt     time.Time
	discoveryCacheErr     string
	discoveryRefreshMu    sync.Mutex
	startedAt             time.Time
}

type discoveryScoredGroup struct {
	group discovery.AssetCandidateGroup
	score int
}

var (
	BuildVersion = "dev"
	BuildCommit  = ""
	BuildTime    = ""
)

func New(config Config) (*App, error) {
	sources, err := loadSyncSources(config)
	if err != nil {
		return nil, err
	}
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		return nil, err
	}
	generated, err := loadRuntimeRegistry(config.RuntimeRegistryPath)
	if err != nil {
		return nil, err
	}
	runtimeRegistry := registry.Merge(generated)
	runtimeRegistry = applyRuntimeAssetClassOverrides(runtimeRegistry, generated)
	runtimeRegistry = applyRuntimeMarketOverrides(runtimeRegistry, generated)
	return &App{
		config: config,
		client: &http.Client{
			Timeout: config.RequestTimeout,
		},
		sources:           sources,
		baseRegistry:      registry,
		generatedRegistry: generated,
		registry:          runtimeRegistry,
		resolver:          identity.NewResolver(runtimeRegistry),
		autoSyncStatus: AutoSyncStatus{
			Enabled:            config.AutoSyncEnabled,
			Interval:           config.AutoSyncInterval.String(),
			RuntimePath:        config.RuntimeRegistryPath,
			ConfiguredSourceID: config.AutoSyncSourceID,
		},
		startedAt: time.Now().UTC(),
	}, nil
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/healthz", a.handleHealthz)
	mux.HandleFunc("/api/v1/resolve/batch", a.handleResolveBatch)
	mux.HandleFunc("/api/v1/resolve", a.handleResolve)
	mux.HandleFunc("/api/v1/registry", a.handleRuntimeRegistry)
	mux.HandleFunc("/api/v1/registry/overrides", a.handleRuntimeRegistryOverride)
	mux.HandleFunc("/api/v1/assets/", a.handleAsset)
	mux.HandleFunc("/api/v1/version", a.handleVersion)
	mux.HandleFunc("/api/v1/auto-sync", a.handleAutoSync)
	mux.HandleFunc("/api/discovery/sources", a.handleDiscoverySources)
	mux.HandleFunc("/api/discovery/current", a.handleDiscoveryCurrent)
	mux.HandleFunc("/api/discovery/sync", a.handleDiscoverySync)
	mux.HandleFunc("/api/discovery/lookup", a.handleDiscoveryLookup)
	mux.HandleFunc("/api/registry", a.handleRegistry)
	mux.Handle("/", a.frontendHandler())
	return a.withCORS(mux)
}

func (a *App) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" && a.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	expected := strings.TrimSpace(a.config.AdminCode)
	if expected == "" {
		return true
	}

	provided := strings.TrimSpace(r.Header.Get("X-Market-Kit-Admin-Code"))
	if provided == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			provided = strings.TrimSpace(auth[len("bearer "):])
		}
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1 {
		return true
	}

	writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "admin authorization required"})
	return false
}

func (a *App) originAllowed(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	for _, allowed := range a.config.AllowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" || strings.EqualFold(origin, allowed) {
			return true
		}
		if strings.HasPrefix(allowed, "https://*.") && parsed.Scheme == "https" {
			suffix := strings.TrimPrefix(allowed, "https://*")
			if strings.HasSuffix(strings.ToLower(parsed.Hostname()), strings.ToLower(suffix)) {
				return true
			}
		}
	}
	return false
}

func (a *App) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	registry := a.runtimeRegistry()
	autoSync := a.currentAutoSyncStatus()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"app":       "market-kit",
		"sources":   len(a.sources),
		"assets":    len(registry.AssetAliases),
		"overrides": len(registry.MarketOverrides),
		"autoSync":  autoSync,
	})
}

type resolveAPIRequest struct {
	Exchange            string `json:"exchange"`
	Symbol              string `json:"symbol"`
	CanonicalSymbolHint string `json:"canonicalSymbolHint,omitempty"`
	MarketType          string `json:"marketType,omitempty"`
	MarketTypeHint      string `json:"marketTypeHint,omitempty"`
	InstType            string `json:"instType,omitempty"`
	ProductType         string `json:"productType,omitempty"`
}

type resolveBatchRequest struct {
	Items []resolveAPIRequest `json:"items"`
}

type registryOverrideRequest struct {
	Exchange        string  `json:"exchange"`
	RawSymbol       string  `json:"rawSymbol"`
	MarketType      string  `json:"marketType"`
	CanonicalSymbol string  `json:"canonicalSymbol"`
	AssetClass      string  `json:"assetClass,omitempty"`
	UnitAlias       string  `json:"unitAlias,omitempty"`
	UnitMultiplier  float64 `json:"unitMultiplier,omitempty"`
}

type assetClassUpdateRequest struct {
	AssetClass string `json:"assetClass"`
}

func (a *App) handleResolve(w http.ResponseWriter, r *http.Request) {
	var req resolveAPIRequest
	switch r.Method {
	case http.MethodGet:
		req = resolveAPIRequest{
			Exchange:            r.URL.Query().Get("exchange"),
			Symbol:              r.URL.Query().Get("symbol"),
			CanonicalSymbolHint: firstNonEmpty(r.URL.Query().Get("canonicalSymbolHint"), r.URL.Query().Get("canonicalHint")),
			MarketType:          r.URL.Query().Get("marketType"),
			MarketTypeHint:      r.URL.Query().Get("marketTypeHint"),
			InstType:            r.URL.Query().Get("instType"),
			ProductType:         r.URL.Query().Get("productType"),
		}
	case http.MethodPost:
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid resolve request json"})
			return
		}
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	if strings.TrimSpace(req.Exchange) == "" || strings.TrimSpace(req.Symbol) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "exchange and symbol are required"})
		return
	}

	writeJSON(w, http.StatusOK, a.runtimeResolver().Resolve(req.toIdentityRequest()))
}

func (a *App) handleResolveBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var batch resolveBatchRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
	if err := decoder.Decode(&batch); err != nil || len(batch.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request body must contain a non-empty items array"})
		return
	}
	if len(batch.Items) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "batch limit is 500 items"})
		return
	}

	resolver := a.runtimeResolver()
	results := make([]identity.ResolveResult, 0, len(batch.Items))
	for _, item := range batch.Items {
		results = append(results, resolver.Resolve(item.toIdentityRequest()))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": map[string]any{
			"count": len(results),
		},
		"results": results,
	})
}

func (req resolveAPIRequest) toIdentityRequest() identity.ResolveRequest {
	return identity.ResolveRequest{
		Exchange:            strings.TrimSpace(req.Exchange),
		Symbol:              strings.TrimSpace(req.Symbol),
		CanonicalSymbolHint: strings.TrimSpace(req.CanonicalSymbolHint),
		MarketTypeHint:      firstNonEmpty(req.MarketTypeHint, req.MarketType),
		InstType:            strings.TrimSpace(req.InstType),
		ProductType:         strings.TrimSpace(req.ProductType),
	}
}

func (a *App) handleRuntimeRegistry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, a.runtimeRegistry())
}

func (a *App) handleRuntimeRegistryOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !a.requireAdmin(w, r) {
		return
	}

	var req registryOverrideRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid registry override json"})
		return
	}

	override, assetBase, err := normalizeRegistryOverrideRequest(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	assetClass := normalizeRegistryAssetClass(req.AssetClass)
	if strings.TrimSpace(req.AssetClass) != "" && assetClass == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "assetClass must be crypto, rwa_stock, rwa_commodity, fiat_stable, or unknown"})
		return
	}
	if registryOverrideCreatesNewAsset(a.generatedRegistrySnapshot(), a.runtimeRegistry(), assetBase) && assetClass == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "assetClass is required when assigning a market to a new base asset"})
		return
	}

	nextGenerated := upsertRuntimeRegistryOverride(a.generatedRegistrySnapshot(), a.runtimeRegistry(), override, assetBase, assetClass)
	if err := writeRuntimeRegistry(a.config.RuntimeRegistryPath, nextGenerated); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "write runtime registry failed"})
		return
	}
	a.applyGeneratedRegistry(nextGenerated)

	persistedOverride, ok := findMarketOverride(a.runtimeRegistry(), override)
	if !ok {
		persistedOverride = override
	}
	payload := map[string]any{
		"override": persistedOverride,
	}
	if !compactRegistryResponse(r) {
		payload["registry"] = a.runtimeRegistry()
	}
	writeJSON(w, http.StatusOK, payload)
}

func normalizeRegistryOverrideRequest(req registryOverrideRequest) (identity.MarketOverride, string, error) {
	exchange := strings.ToLower(strings.TrimSpace(req.Exchange))
	rawSymbol := strings.TrimSpace(req.RawSymbol)
	marketType := normalizeRegistryOverrideMarketType(req.MarketType)
	canonicalSymbol := strings.ToUpper(strings.TrimSpace(req.CanonicalSymbol))

	if exchange == "" {
		return identity.MarketOverride{}, "", errors.New("exchange is required")
	}
	if rawSymbol == "" {
		return identity.MarketOverride{}, "", errors.New("rawSymbol is required")
	}
	if marketType == "" {
		return identity.MarketOverride{}, "", errors.New("marketType must be spot, perpetual, or future")
	}
	base, quote, ok := splitCanonicalSymbol(canonicalSymbol)
	if !ok {
		return identity.MarketOverride{}, "", errors.New("canonicalSymbol must be BASE/QUOTE")
	}
	if req.UnitMultiplier < 0 {
		return identity.MarketOverride{}, "", errors.New("unitMultiplier must be greater than zero when provided")
	}

	return identity.MarketOverride{
		Exchange:        exchange,
		RawSymbol:       rawSymbol,
		MarketType:      marketType,
		CanonicalSymbol: base + "/" + quote,
		UnitAlias:       strings.ToUpper(strings.TrimSpace(req.UnitAlias)),
		UnitMultiplier:  req.UnitMultiplier,
	}, base, nil
}

func normalizeRegistryOverrideMarketType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "spot":
		return string(identity.MarketTypeSpot)
	case "perp", "perps", "perpetual", "swap", "linear":
		return string(identity.MarketTypePerpetual)
	case "future", "futures", "delivery":
		return string(identity.MarketTypeFuture)
	default:
		return ""
	}
}

func normalizeRegistryAssetClass(value string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	raw = strings.ReplaceAll(raw, "-", "_")
	raw = strings.Join(strings.Fields(raw), "_")
	switch raw {
	case "":
		return ""
	case "crypto":
		return "crypto"
	case "rwa_stock", "stock", "equity", "share", "security", "etf", "index", "tradfi":
		return "rwa_stock"
	case "rwa_commodity", "commodity":
		return "rwa_commodity"
	case "fiat_stable", "stable", "stablecoin":
		return "fiat_stable"
	case "unknown":
		return "unknown"
	default:
		return ""
	}
}

func splitCanonicalSymbol(value string) (string, string, bool) {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(value)), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	base := strings.TrimSpace(parts[0])
	quote := strings.TrimSpace(parts[1])
	return base, quote, base != "" && quote != ""
}

func upsertRuntimeRegistryOverride(generated identity.Registry, runtime identity.Registry, override identity.MarketOverride, assetBase string, assetClass string) identity.Registry {
	generated.Normalize()
	runtime.Normalize()
	override.Exchange = strings.ToLower(strings.TrimSpace(override.Exchange))
	override.RawSymbol = strings.TrimSpace(override.RawSymbol)
	override.MarketType = normalizeRegistryOverrideMarketType(override.MarketType)
	override.CanonicalSymbol = strings.ToUpper(strings.TrimSpace(override.CanonicalSymbol))
	override.UnitAlias = strings.ToUpper(strings.TrimSpace(override.UnitAlias))
	if override.UnitMultiplier <= 0 {
		override.UnitMultiplier = 0
	}
	targetKey := registryOverrideKey(override)

	nextOverrides := make([]identity.MarketOverride, 0, len(generated.MarketOverrides)+1)
	for _, item := range generated.MarketOverrides {
		if registryOverrideKey(item) == targetKey {
			continue
		}
		nextOverrides = append(nextOverrides, item)
	}
	nextOverrides = append(nextOverrides, override)
	generated.MarketOverrides = nextOverrides

	assetBase = strings.ToUpper(strings.TrimSpace(assetBase))
	if assetBase != "" && !registryHasAsset(runtime, assetBase) && !registryHasAsset(generated, assetBase) {
		class := strings.TrimSpace(assetClass)
		if class != "" {
			generated.AssetAliases = append(generated.AssetAliases, identity.AssetAliasRule{
				Canonical:  assetBase,
				AssetClass: class,
			})
		}
	}

	generated.Normalize()
	return generated
}

func registryOverrideCreatesNewAsset(generated identity.Registry, runtime identity.Registry, assetBase string) bool {
	assetBase = strings.ToUpper(strings.TrimSpace(assetBase))
	if assetBase == "" {
		return false
	}
	return !registryHasAsset(runtime, assetBase) && !registryHasAsset(generated, assetBase)
}

func registryOverrideKey(item identity.MarketOverride) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(item.Exchange)),
		strings.TrimSpace(item.RawSymbol),
		normalizeRegistryOverrideMarketType(item.MarketType),
	}, "|")
}

func findMarketOverride(registry identity.Registry, target identity.MarketOverride) (identity.MarketOverride, bool) {
	targetKey := registryOverrideKey(target)
	for _, item := range registry.MarketOverrides {
		if registryOverrideKey(item) == targetKey {
			return item, true
		}
	}
	return identity.MarketOverride{}, false
}

func registryHasAsset(registry identity.Registry, canonical string) bool {
	canonical = strings.ToUpper(strings.TrimSpace(canonical))
	for _, item := range registry.AssetAliases {
		if strings.EqualFold(item.Canonical, canonical) {
			return true
		}
	}
	return false
}

func (a *App) handleAsset(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/assets/"), "/ "))
	if symbol == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "asset symbol is required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		a.handleAssetGet(w, symbol)
	case http.MethodPost, http.MethodPatch:
		a.handleAssetClassUpdate(w, r, symbol)
	default:
		w.Header().Set("Allow", "GET, POST, PATCH")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (a *App) handleAssetGet(w http.ResponseWriter, symbol string) {
	if asset, ok := findAssetAlias(a.runtimeRegistry(), symbol); ok {
		writeJSON(w, http.StatusOK, map[string]any{"asset": asset})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"error": "asset not found", "asset": symbol})
}

func (a *App) handleAssetClassUpdate(w http.ResponseWriter, r *http.Request, symbol string) {
	if !a.requireAdmin(w, r) {
		return
	}

	var req assetClassUpdateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid asset class update json"})
		return
	}
	assetClass := normalizeRegistryAssetClass(req.AssetClass)
	if assetClass == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "assetClass must be crypto, rwa_stock, rwa_commodity, fiat_stable, or unknown"})
		return
	}
	if !registryHasAsset(a.runtimeRegistry(), symbol) && !registryHasAsset(a.generatedRegistrySnapshot(), symbol) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "asset not found", "asset": symbol})
		return
	}

	nextGenerated := updateRuntimeAssetClass(a.generatedRegistrySnapshot(), symbol, assetClass)
	if err := writeRuntimeRegistry(a.config.RuntimeRegistryPath, nextGenerated); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "write runtime registry failed"})
		return
	}
	a.applyGeneratedRegistry(nextGenerated)

	asset, _ := findAssetAlias(a.runtimeRegistry(), symbol)
	payload := map[string]any{"asset": asset}
	if !compactRegistryResponse(r) {
		payload["registry"] = a.runtimeRegistry()
	}
	writeJSON(w, http.StatusOK, payload)
}

func compactRegistryResponse(r *http.Request) bool {
	value := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("compact")))
	return value == "1" || value == "true" || value == "yes"
}

func updateRuntimeAssetClass(generated identity.Registry, symbol string, assetClass string) identity.Registry {
	generated.Normalize()
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	assetClass = strings.TrimSpace(assetClass)
	if symbol == "" || assetClass == "" {
		return generated
	}
	for idx := range generated.AssetAliases {
		if strings.EqualFold(generated.AssetAliases[idx].Canonical, symbol) {
			generated.AssetAliases[idx].AssetClass = assetClass
			generated.Normalize()
			return generated
		}
	}
	generated.AssetAliases = append(generated.AssetAliases, identity.AssetAliasRule{
		Canonical:  symbol,
		AssetClass: assetClass,
	})
	generated.Normalize()
	return generated
}

func applyRuntimeAssetClassOverrides(runtime identity.Registry, generated identity.Registry) identity.Registry {
	overrides := map[string]string{}
	for _, item := range generated.AssetAliases {
		canonical := strings.ToUpper(strings.TrimSpace(item.Canonical))
		assetClass := strings.TrimSpace(item.AssetClass)
		if canonical == "" || assetClass == "" {
			continue
		}
		overrides[canonical] = assetClass
	}
	if len(overrides) == 0 {
		return runtime
	}
	for idx := range runtime.AssetAliases {
		canonical := strings.ToUpper(strings.TrimSpace(runtime.AssetAliases[idx].Canonical))
		if assetClass, ok := overrides[canonical]; ok {
			runtime.AssetAliases[idx].AssetClass = assetClass
		}
	}
	runtime.Normalize()
	return runtime
}

func applyRuntimeMarketOverrides(runtime identity.Registry, generated identity.Registry) identity.Registry {
	replacements := map[string]identity.MarketOverride{}
	for _, item := range generated.MarketOverrides {
		key := registryOverrideKey(item)
		if key == "||" {
			continue
		}
		replacements[key] = item
	}
	if len(replacements) == 0 {
		return runtime
	}

	next := make([]identity.MarketOverride, 0, len(runtime.MarketOverrides))
	replaced := map[string]bool{}
	for _, item := range runtime.MarketOverrides {
		key := registryOverrideKey(item)
		if replacement, ok := replacements[key]; ok {
			next = append(next, replacement)
			replaced[key] = true
			continue
		}
		next = append(next, item)
	}
	for key, item := range replacements {
		if replaced[key] {
			continue
		}
		next = append(next, item)
	}

	runtime.MarketOverrides = next
	runtime.Normalize()
	return runtime
}

func (a *App) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	registry := a.runtimeRegistry()
	writeJSON(w, http.StatusOK, map[string]any{
		"app":           "market-kit",
		"version":       BuildVersion,
		"commit":        BuildCommit,
		"buildTime":     BuildTime,
		"startedAt":     a.startedAt,
		"assetCount":    len(registry.AssetAliases),
		"overrideCount": len(registry.MarketOverrides),
	})
}

func (a *App) handleDiscoverySources(w http.ResponseWriter, _ *http.Request) {
	items := make([]map[string]any, 0, len(a.sources))
	for _, source := range a.sources {
		items = append(items, map[string]any{
			"id":         source.ID,
			"label":      source.Label,
			"project":    source.Project,
			"kind":       source.Kind,
			"url":        source.URL,
			"hasHeaders": len(source.Headers) > 0,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": items})
}

func (a *App) handleDiscoveryCurrent(w http.ResponseWriter, r *http.Request) {
	sourceID := strings.TrimSpace(r.URL.Query().Get("source"))
	if sourceID == "" {
		sourceID = "all"
	}

	payload, sources, cachedAt, errText, _ := a.discoveryCacheSnapshot()
	if !discoveryCacheMatchesSource(payload, sources, sourceID) {
		if err := a.refreshDiscoveryCache(r.Context(), sourceID); err != nil {
			a.writeDiscoveryError(w, err)
			return
		}
		payload, sources, cachedAt, errText, _ = a.discoveryCacheSnapshot()
	}
	if len(sources) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no discovery source is configured"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source":      discoveryLookupSourcePayload(sources),
		"sources":     discoverySourcePayloads(sources),
		"payload":     payload,
		"cachedAt":    cachedAt,
		"cacheError":  errText,
		"serverCache": true,
	})
}

func (a *App) handleDiscoverySync(w http.ResponseWriter, r *http.Request) {
	sourceID := strings.TrimSpace(r.URL.Query().Get("source"))
	if sourceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "source is required"})
		return
	}
	if !a.requireAdmin(w, r) {
		return
	}

	forceRefresh := parseBool(firstNonEmpty(r.URL.Query().Get("refresh"), r.URL.Query().Get("force")), false)
	if !forceRefresh {
		payload, sources, cachedAt, errText, _ := a.discoveryCacheSnapshot()
		if discoveryCacheMatchesSource(payload, sources, sourceID) && discoveryCacheFresh(cachedAt, a.config.DiscoveryCacheTTL) {
			writeJSON(w, http.StatusOK, map[string]any{
				"source":      discoveryLookupSourcePayload(sources),
				"sources":     discoverySourcePayloads(sources),
				"payload":     payload,
				"cachedAt":    cachedAt,
				"cacheError":  errText,
				"serverCache": true,
			})
			return
		}
	}

	if err := a.refreshDiscoveryCacheWithOptions(r.Context(), sourceID, forceRefresh); err != nil {
		a.writeDiscoveryError(w, err)
		return
	}
	payload, sources, cachedAt, errText, _ := a.discoveryCacheSnapshot()
	if len(sources) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no discovery source is configured"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source":      discoveryLookupSourcePayload(sources),
		"sources":     discoverySourcePayloads(sources),
		"payload":     payload,
		"cachedAt":    cachedAt,
		"cacheError":  errText,
		"serverCache": true,
	})
}

func (a *App) handleDiscoveryLookup(w http.ResponseWriter, r *http.Request) {
	query := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "symbol is required"})
		return
	}

	sourceID := strings.TrimSpace(r.URL.Query().Get("source"))
	sources, envelope, groups, cachedAt, cacheErr, err := a.discoveryLookupEnvelopeFromCacheOrRefresh(r.Context(), sourceID)
	if err != nil {
		a.writeDiscoveryError(w, err)
		return
	}
	if len(sources) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no discovery source is configured"})
		return
	}

	registry := a.runtimeRegistry()
	if len(groups) == 0 && len(envelope.Items) > 0 {
		groups = buildDiscoveryGroups(envelope.Items, registry)
	}
	matches := filterDiscoveryGroups(groups, query, registry)

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"source":  discoveryLookupSourcePayload(sources),
		"sources": discoverySourcePayloads(sources),
		"summary": map[string]any{
			"groupCount":  len(matches),
			"marketCount": totalDiscoveryMarkets(matches),
		},
		"groups":      matches,
		"cachedAt":    cachedAt,
		"cacheError":  cacheErr,
		"serverCache": !cachedAt.IsZero(),
	})
}

func (a *App) discoveryLookupEnvelopeFromCacheOrRefresh(ctx context.Context, sourceID string) ([]SyncSource, discovery.ImportEnvelope, []discovery.AssetCandidateGroup, time.Time, string, error) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		sourceID = "all"
	}

	payload, sources, cachedAt, errText, groups := a.discoveryCacheSnapshot()
	if discoveryCacheMatchesSource(payload, sources, sourceID) {
		return sources, payload, groups, cachedAt, errText, nil
	}

	if err := a.refreshDiscoveryCache(ctx, sourceID); err != nil {
		return nil, discovery.ImportEnvelope{}, nil, time.Time{}, "", err
	}
	payload, sources, cachedAt, errText, groups = a.discoveryCacheSnapshot()
	return sources, payload, groups, cachedAt, errText, nil
}

func discoveryCacheMatchesSource(payload discovery.ImportEnvelope, sources []SyncSource, sourceID string) bool {
	if len(payload.Items) == 0 || strings.TrimSpace(string(payload.Source)) == "" || len(sources) == 0 {
		return false
	}

	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" || strings.EqualFold(sourceID, "all") {
		return strings.EqualFold(string(payload.Source), "market-kit-all") || len(sources) > 1
	}

	if strings.EqualFold(sourceID, string(payload.Source)) {
		return true
	}
	if len(sources) == 1 && strings.EqualFold(sources[0].ID, sourceID) {
		return true
	}
	return false
}

func (a *App) fetchDiscoveryLookupEnvelope(ctx context.Context, sourceID string) ([]SyncSource, discovery.ImportEnvelope, error) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID != "" && !strings.EqualFold(sourceID, "all") {
		source, envelope, err := a.fetchDiscoveryEnvelope(ctx, sourceID)
		if err != nil {
			return nil, discovery.ImportEnvelope{}, err
		}
		return []SyncSource{source}, envelope, nil
	}

	sources := a.discoverySources()
	if len(sources) == 0 {
		return nil, discovery.ImportEnvelope{}, nil
	}

	items := make([]discovery.ImportedMarket, 0)
	seen := make(map[string]struct{})
	var latestGeneratedAt time.Time

	type fetchResult struct {
		index   int
		source  SyncSource
		payload discovery.ImportEnvelope
		err     error
	}
	results := make(chan fetchResult, len(sources))
	for index, source := range sources {
		go func(index int, source SyncSource) {
			fetchedSource, envelope, err := a.fetchDiscoveryEnvelope(ctx, source.ID)
			if err != nil {
				results <- fetchResult{index: index, err: err}
				return
			}
			results <- fetchResult{index: index, source: fetchedSource, payload: envelope}
		}(index, source)
	}

	fetched := make([]fetchResult, len(sources))
	for range sources {
		result := <-results
		fetched[result.index] = result
	}

	successful := 0
	failed := make([]string, 0)
	for index, source := range sources {
		result := fetched[index]
		if result.err != nil {
			failed = append(failed, result.err.Error())
			continue
		}
		successful++
		if result.payload.GeneratedAt.After(latestGeneratedAt) {
			latestGeneratedAt = result.payload.GeneratedAt
		}
		for _, item := range result.payload.Items {
			key := discoveryImportedMarketKey(item)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			items = append(items, item)
		}
		if source.ID != result.source.ID {
			source = result.source
		}
		sources[index] = source
	}
	if successful == 0 && len(failed) > 0 {
		return nil, discovery.ImportEnvelope{}, fmt.Errorf("all discovery sources failed: %s", strings.Join(failed, "; "))
	}
	if latestGeneratedAt.IsZero() {
		latestGeneratedAt = time.Now().UTC()
	}
	return sources, discovery.ImportEnvelope{
		Source:      discovery.SourceKind("market-kit-all"),
		GeneratedAt: latestGeneratedAt,
		Items:       items,
	}, nil
}

func (a *App) refreshDiscoveryCache(ctx context.Context, sourceID string) error {
	return a.refreshDiscoveryCacheWithOptions(ctx, sourceID, false)
}

func (a *App) refreshDiscoveryCacheWithOptions(ctx context.Context, sourceID string, force bool) error {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		sourceID = "all"
	}
	a.discoveryRefreshMu.Lock()
	defer a.discoveryRefreshMu.Unlock()

	if !force {
		payload, sources, cachedAt, _, _ := a.discoveryCacheSnapshot()
		if discoveryCacheMatchesSource(payload, sources, sourceID) && discoveryCacheFresh(cachedAt, a.config.DiscoveryCacheTTL) {
			return nil
		}
	}

	sources, envelope, err := a.fetchDiscoveryLookupEnvelope(ctx, sourceID)
	now := time.Now().UTC()
	groups := []discovery.AssetCandidateGroup(nil)
	if err == nil {
		groups = buildDiscoveryGroups(envelope.Items, a.runtimeRegistry())
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.discoveryCacheErr = err.Error()
		return err
	}
	a.discoveryCacheSources = sources
	a.discoveryCache = envelope
	a.discoveryCacheGroups = groups
	a.discoveryCachedAt = now
	a.discoveryCacheErr = ""
	return nil
}

func (a *App) discoveryCacheSnapshot() (discovery.ImportEnvelope, []SyncSource, time.Time, string, []discovery.AssetCandidateGroup) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	sources := append([]SyncSource(nil), a.discoveryCacheSources...)
	groups := append([]discovery.AssetCandidateGroup(nil), a.discoveryCacheGroups...)
	return a.discoveryCache, sources, a.discoveryCachedAt, a.discoveryCacheErr, groups
}

func discoveryCacheFresh(cachedAt time.Time, ttl time.Duration) bool {
	if cachedAt.IsZero() {
		return false
	}
	if ttl <= 0 {
		return true
	}
	return time.Since(cachedAt) <= ttl
}

func buildDiscoveryGroups(items []discovery.ImportedMarket, registry identity.Registry) []discovery.AssetCandidateGroup {
	aggregator := discovery.NewAggregator(registry)
	return aggregator.BuildAssetGroups(items)
}

func (a *App) handleRegistry(w http.ResponseWriter, _ *http.Request) {
	registry, err := os.ReadFile(filepath.Join("identity", "default_registry.json"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(registry)
}

func (a *App) runtimeRegistry() identity.Registry {
	a.mu.RLock()
	if a.registry.AssetAliases != nil || a.registry.MarketOverrides != nil || a.registry.ExchangeAliases != nil {
		registry := a.registry
		a.mu.RUnlock()
		return registry
	}
	a.mu.RUnlock()
	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		return identity.Registry{}
	}
	return registry
}

func (a *App) runtimeResolver() *identity.Resolver {
	a.mu.RLock()
	if a.resolver != nil {
		resolver := a.resolver
		a.mu.RUnlock()
		return resolver
	}
	a.mu.RUnlock()
	return identity.NewResolver(a.runtimeRegistry())
}

func findAssetAlias(registry identity.Registry, symbol string) (identity.AssetAliasRule, bool) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	for _, item := range registry.AssetAliases {
		if item.Canonical == symbol {
			return item, true
		}
		for _, alias := range item.Aliases {
			if strings.EqualFold(alias, symbol) {
				return item, true
			}
		}
		for _, alias := range item.UnitAliases {
			if strings.EqualFold(alias.Alias, symbol) {
				return item, true
			}
		}
	}
	return identity.AssetAliasRule{}, false
}

func (a *App) frontendHandler() http.Handler {
	dist := http.Dir(a.config.FrontendDistDir)
	fileServer := http.FileServer(dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean("/" + strings.TrimSpace(r.URL.Path))
		target := filepath.Join(a.config.FrontendDistDir, strings.TrimPrefix(cleanPath, "/"))
		if cleanPath == "/" {
			http.ServeFile(w, r, filepath.Join(a.config.FrontendDistDir, "index.html"))
			return
		}
		if stat, err := os.Stat(target); err == nil && !stat.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(a.config.FrontendDistDir, "index.html"))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type discoveryFetchError struct {
	Status  int
	Source  string
	Message string
	Body    string
}

func (e *discoveryFetchError) Error() string {
	if e == nil {
		return ""
	}
	if e.Source == "" {
		return e.Message
	}
	return e.Source + ": " + e.Message
}

func (a *App) fetchDiscoveryEnvelope(ctx context.Context, sourceID string) (SyncSource, discovery.ImportEnvelope, error) {
	source, ok := a.lookupSource(sourceID)
	if !ok {
		return SyncSource{}, discovery.ImportEnvelope{}, &discoveryFetchError{
			Status:  http.StatusNotFound,
			Source:  sourceID,
			Message: "sync source not found",
		}
	}

	if source.URL == bootstrap.BuiltInSourceURL {
		payload, err := bootstrap.FetchDefault(ctx, a.client)
		if err != nil {
			return SyncSource{}, discovery.ImportEnvelope{}, &discoveryFetchError{
				Status:  http.StatusBadGateway,
				Source:  source.ID,
				Message: err.Error(),
			}
		}
		return source, payload, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return SyncSource{}, discovery.ImportEnvelope{}, &discoveryFetchError{
			Status:  http.StatusInternalServerError,
			Source:  source.ID,
			Message: err.Error(),
		}
	}
	for key, value := range source.Headers {
		req.Header.Set(key, value)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return SyncSource{}, discovery.ImportEnvelope{}, &discoveryFetchError{
			Status:  http.StatusBadGateway,
			Source:  source.ID,
			Message: err.Error(),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return SyncSource{}, discovery.ImportEnvelope{}, &discoveryFetchError{
			Status:  http.StatusBadGateway,
			Source:  source.ID,
			Message: err.Error(),
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SyncSource{}, discovery.ImportEnvelope{}, &discoveryFetchError{
			Status:  resp.StatusCode,
			Source:  source.ID,
			Message: "remote responded with non-2xx status",
			Body:    string(body),
		}
	}

	var payload discovery.ImportEnvelope
	if err := json.Unmarshal(body, &payload); err != nil {
		return SyncSource{}, discovery.ImportEnvelope{}, &discoveryFetchError{
			Status:  http.StatusBadGateway,
			Source:  source.ID,
			Message: "remote payload is not valid discovery json",
		}
	}
	if payload.Source == "" {
		payload.Source = discovery.SourceKind(source.Project)
	}
	return source, payload, nil
}

func (a *App) lookupSource(sourceID string) (SyncSource, bool) {
	for i := range a.sources {
		if a.sources[i].ID == sourceID {
			return a.sources[i], true
		}
	}
	return SyncSource{}, false
}

func (a *App) defaultDiscoverySourceID() string {
	for _, source := range a.sources {
		if source.Kind == "discovery" && source.ID == bootstrap.BuiltInSourceID {
			return source.ID
		}
	}
	for _, source := range a.sources {
		if source.Kind == "discovery" {
			return source.ID
		}
	}
	return ""
}

func (a *App) discoverySources() []SyncSource {
	items := make([]SyncSource, 0)
	for _, source := range a.sources {
		if source.Kind != "discovery" {
			continue
		}
		items = append(items, source)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ID == bootstrap.BuiltInSourceID {
			return true
		}
		if items[j].ID == bootstrap.BuiltInSourceID {
			return false
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func (a *App) writeDiscoveryError(w http.ResponseWriter, err error) {
	var fetchErr *discoveryFetchError
	if errors.As(err, &fetchErr) {
		payload := map[string]any{
			"error":  fetchErr.Message,
			"source": fetchErr.Source,
		}
		if fetchErr.Body != "" {
			payload["body"] = fetchErr.Body
		}
		writeJSON(w, fetchErr.Status, payload)
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
}

func discoverySourcePayload(source SyncSource) map[string]any {
	return map[string]any{
		"id":      source.ID,
		"label":   source.Label,
		"project": source.Project,
		"kind":    source.Kind,
	}
}

func discoverySourcePayloads(sources []SyncSource) []map[string]any {
	items := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		items = append(items, discoverySourcePayload(source))
	}
	return items
}

func discoveryLookupSourcePayload(sources []SyncSource) map[string]any {
	if len(sources) == 1 {
		return discoverySourcePayload(sources[0])
	}
	return map[string]any{
		"id":      "all",
		"label":   "全部市场发现源",
		"project": "market-kit",
		"kind":    "discovery",
	}
}

func discoveryImportedMarketKey(item discovery.ImportedMarket) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(item.PlatformID)),
		strings.ToLower(strings.TrimSpace(item.MarketType)),
		strings.ToUpper(strings.TrimSpace(item.Symbol)),
		strings.ToUpper(strings.TrimSpace(item.BaseAsset)),
		strings.ToUpper(strings.TrimSpace(item.QuoteAsset)),
		strings.ToLower(strings.TrimSpace(item.Chain)),
	}, "\x00")
}

func filterDiscoveryGroups(groups []discovery.AssetCandidateGroup, query string, registry identity.Registry) []discovery.AssetCandidateGroup {
	aliasIndex := registryAssetAliasIndex(registry)
	out := make([]discoveryScoredGroup, 0)
	for _, group := range groups {
		score := discoveryGroupScore(group, query, aliasIndex)
		if score <= 0 {
			continue
		}
		out = append(out, discoveryScoredGroup{group: group, score: score})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score == out[j].score {
			return out[i].group.GroupKey < out[j].group.GroupKey
		}
		return out[i].score > out[j].score
	})

	if hasHighConfidenceDiscoveryScore(out) {
		filtered := out[:0]
		for _, item := range out {
			if item.score >= 95 {
				filtered = append(filtered, item)
			}
		}
		out = filtered
	}

	matches := make([]discovery.AssetCandidateGroup, 0, len(out))
	for _, item := range out {
		matches = append(matches, item.group)
	}
	return matches
}

func discoveryGroupScore(group discovery.AssetCandidateGroup, query string, aliasIndex map[string][]string) int {
	query = strings.ToUpper(strings.TrimSpace(query))
	if query == "" {
		return 0
	}

	best := 0
	switch {
	case strings.EqualFold(group.CanonicalAsset, query):
		best = 120
	case strings.EqualFold(group.CanonicalSymbol, query), strings.EqualFold(group.GroupKey, query):
		best = 110
	case strings.Contains(strings.ToUpper(group.CanonicalSymbol), query):
		best = 70
	case strings.Contains(strings.ToUpper(group.GroupKey), query):
		best = 65
	}

	for _, alias := range aliasIndex[strings.ToUpper(strings.TrimSpace(group.CanonicalAsset))] {
		switch {
		case alias == query:
			best = max(best, 105)
		case strings.Contains(alias, query):
			best = max(best, 62)
		}
	}

	for _, market := range group.Markets {
		candidate := strings.ToUpper(strings.TrimSpace(market.RawSymbol))
		venueSymbol := strings.ToUpper(strings.TrimSpace(market.VenueSymbol))
		baseAsset := strings.ToUpper(strings.TrimSpace(market.BaseAsset))
		canonicalSymbol := strings.ToUpper(strings.TrimSpace(market.CanonicalSymbol))
		switch {
		case candidate == query, venueSymbol == query:
			best = max(best, 100)
		case namespacedSuffix(candidate) == query, namespacedSuffix(venueSymbol) == query, namespacedSuffix(baseAsset) == query, namespacedSuffix(canonicalSymbol) == query:
			best = max(best, 98)
		case baseAsset == query:
			best = max(best, 95)
		case strings.Contains(candidate, query), strings.Contains(venueSymbol, query):
			best = max(best, 60)
		case strings.Contains(canonicalSymbol, query):
			best = max(best, 55)
		}
	}
	return best
}

func namespacedSuffix(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if slash := strings.Index(value, "/"); slash >= 0 {
		value = value[:slash]
	}
	if colon := strings.LastIndex(value, ":"); colon >= 0 && colon+1 < len(value) {
		return strings.TrimSpace(value[colon+1:])
	}
	return ""
}

func registryAssetAliasIndex(registry identity.Registry) map[string][]string {
	index := make(map[string][]string, len(registry.AssetAliases))
	for _, item := range registry.AssetAliases {
		canonical := strings.ToUpper(strings.TrimSpace(item.Canonical))
		if canonical == "" {
			continue
		}
		seen := map[string]bool{}
		for _, alias := range item.Aliases {
			alias = strings.ToUpper(strings.TrimSpace(alias))
			if alias == "" || alias == canonical || seen[alias] {
				continue
			}
			index[canonical] = append(index[canonical], alias)
			seen[alias] = true
		}
		for _, alias := range item.UnitAliases {
			value := strings.ToUpper(strings.TrimSpace(alias.Alias))
			if value == "" || value == canonical || seen[value] {
				continue
			}
			index[canonical] = append(index[canonical], value)
			seen[value] = true
		}
	}
	return index
}

func totalDiscoveryMarkets(groups []discovery.AssetCandidateGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Markets)
	}
	return total
}

func hasHighConfidenceDiscoveryScore(items []discoveryScoredGroup) bool {
	for _, item := range items {
		if item.score >= 95 {
			return true
		}
	}
	return false
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func (a *App) ListenAndServe(ctx context.Context) error {
	srv := &http.Server{
		Addr:              a.config.HTTPAddr,
		Handler:           a.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go a.runAutoSyncLoop(ctx)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
