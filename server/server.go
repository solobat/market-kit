package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/solobat/market-kit/bootstrap"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

type App struct {
	config  Config
	client  *http.Client
	sources []SyncSource
}

type discoveryScoredGroup struct {
	group discovery.AssetCandidateGroup
	score int
}

func New(config Config) (*App, error) {
	sources, err := loadSyncSources(config)
	if err != nil {
		return nil, err
	}
	return &App{
		config: config,
		client: &http.Client{
			Timeout: config.RequestTimeout,
		},
		sources: sources,
	}, nil
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/healthz", a.handleHealthz)
	mux.HandleFunc("/api/discovery/sources", a.handleDiscoverySources)
	mux.HandleFunc("/api/discovery/sync", a.handleDiscoverySync)
	mux.HandleFunc("/api/discovery/lookup", a.handleDiscoveryLookup)
	mux.HandleFunc("/api/registry", a.handleRegistry)
	mux.Handle("/", a.frontendHandler())
	return mux
}

func (a *App) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"app":     "market-kit",
		"sources": len(a.sources),
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

func (a *App) handleDiscoverySync(w http.ResponseWriter, r *http.Request) {
	sourceID := strings.TrimSpace(r.URL.Query().Get("source"))
	if sourceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "source is required"})
		return
	}

	source, payload, err := a.fetchDiscoveryEnvelope(r.Context(), sourceID)
	if err != nil {
		a.writeDiscoveryError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source":  discoverySourcePayload(source),
		"payload": payload,
	})
}

func (a *App) handleDiscoveryLookup(w http.ResponseWriter, r *http.Request) {
	query := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "symbol is required"})
		return
	}

	sourceID := strings.TrimSpace(r.URL.Query().Get("source"))
	if sourceID == "" {
		sourceID = a.defaultDiscoverySourceID()
	}
	if sourceID == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no discovery source is configured"})
		return
	}

	source, envelope, err := a.fetchDiscoveryEnvelope(r.Context(), sourceID)
	if err != nil {
		a.writeDiscoveryError(w, err)
		return
	}

	registry, err := identity.LoadDefaultRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	aggregator := discovery.NewAggregator(registry)
	groups := aggregator.BuildAssetGroups(envelope.Items)
	matches := filterDiscoveryGroups(groups, query, registry)

	writeJSON(w, http.StatusOK, map[string]any{
		"query":  query,
		"source": discoverySourcePayload(source),
		"summary": map[string]any{
			"groupCount":  len(matches),
			"marketCount": totalDiscoveryMarkets(matches),
		},
		"groups": matches,
	})
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
		switch {
		case candidate == query, venueSymbol == query:
			best = max(best, 100)
		case baseAsset == query:
			best = max(best, 95)
		case strings.Contains(candidate, query), strings.Contains(venueSymbol, query):
			best = max(best, 60)
		case strings.Contains(strings.ToUpper(strings.TrimSpace(market.CanonicalSymbol)), query):
			best = max(best, 55)
		}
	}
	return best
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
		Addr:    a.config.HTTPAddr,
		Handler: a.Handler(),
	}

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
