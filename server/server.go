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
	"strings"
	"time"
)

type App struct {
	config  Config
	client  *http.Client
	sources []SyncSource
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

	var source *SyncSource
	for i := range a.sources {
		if a.sources[i].ID == sourceID {
			source = &a.sources[i]
			break
		}
	}
	if source == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "sync source not found"})
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, source.URL, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	for key, value := range source.Headers {
		req.Header.Set(key, value)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "source": source.ID})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "source": source.ID})
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, resp.StatusCode, map[string]any{
			"error":  "remote responded with non-2xx status",
			"source": source.ID,
			"body":   string(body),
		})
		return
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  "remote payload is not valid json",
			"source": source.ID,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source": map[string]any{
			"id":      source.ID,
			"label":   source.Label,
			"project": source.Project,
		},
		"payload": payload,
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
