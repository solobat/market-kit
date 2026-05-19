package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/solobat/market-kit/bootstrap"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/identity"
)

type AutoSyncStatus struct {
	Enabled                bool      `json:"enabled"`
	Interval               string    `json:"interval,omitempty"`
	RuntimePath            string    `json:"runtimePath,omitempty"`
	ConfiguredSourceID     string    `json:"configuredSourceId,omitempty"`
	SourceID               string    `json:"sourceId,omitempty"`
	LastStartedAt          time.Time `json:"lastStartedAt,omitempty"`
	LastFinishedAt         time.Time `json:"lastFinishedAt,omitempty"`
	LastError              string    `json:"lastError,omitempty"`
	LastDiscoveryItems     int       `json:"lastDiscoveryItems,omitempty"`
	LastGeneratedAssets    int       `json:"lastGeneratedAssets,omitempty"`
	LastGeneratedOverrides int       `json:"lastGeneratedOverrides,omitempty"`
	RuntimeAssets          int       `json:"runtimeAssets,omitempty"`
	RuntimeOverrides       int       `json:"runtimeOverrides,omitempty"`
}

func loadRuntimeRegistry(path string) (identity.Registry, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return identity.Registry{}, nil
	}
	registry, err := identity.LoadRegistryFile(path)
	if err == nil {
		return registry, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return identity.Registry{}, nil
	}
	return identity.Registry{}, err
}

func (a *App) handleAutoSync(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.currentAutoSyncStatus())
	case http.MethodPost:
		if err := a.runAutoSyncOnce(r.Context()); err != nil {
			status := http.StatusBadGateway
			if strings.Contains(err.Error(), "no discovery source") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{
				"error":    err.Error(),
				"autoSync": a.currentAutoSyncStatus(),
			})
			return
		}
		writeJSON(w, http.StatusOK, a.currentAutoSyncStatus())
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (a *App) currentAutoSyncStatus() AutoSyncStatus {
	a.mu.RLock()
	status := a.autoSyncStatus
	a.mu.RUnlock()
	return status
}

func (a *App) updateAutoSyncStatus(fn func(*AutoSyncStatus)) {
	a.mu.Lock()
	fn(&a.autoSyncStatus)
	a.mu.Unlock()
}

func (a *App) generatedRegistrySnapshot() identity.Registry {
	a.mu.RLock()
	registry := a.generatedRegistry
	a.mu.RUnlock()
	return registry
}

func (a *App) applyGeneratedRegistry(generated identity.Registry) {
	generated.Normalize()
	runtime := a.baseRegistry.Merge(generated)

	a.mu.Lock()
	a.generatedRegistry = generated
	a.registry = runtime
	a.resolver = identity.NewResolver(runtime)
	a.autoSyncStatus.RuntimeAssets = len(runtime.AssetAliases)
	a.autoSyncStatus.RuntimeOverrides = len(runtime.MarketOverrides)
	a.mu.Unlock()
}

func (a *App) runAutoSyncLoop(ctx context.Context) {
	if !a.config.AutoSyncEnabled {
		return
	}
	if strings.TrimSpace(a.config.AutoSyncSourceID) == "" && a.defaultAutoSyncSourceID() == "" {
		err := "no production discovery source configured for auto-sync"
		a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
			status.Enabled = a.config.AutoSyncEnabled
			status.LastError = err
		})
		log.Printf("market-kit auto-sync disabled: %s", err)
		return
	}

	if err := a.runAutoSyncOnce(ctx); err != nil {
		log.Printf("market-kit auto-sync initial refresh failed: %v", err)
	}

	interval := a.config.AutoSyncInterval
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.runAutoSyncOnce(ctx); err != nil {
				log.Printf("market-kit auto-sync refresh failed: %v", err)
			}
		}
	}
}

func (a *App) runAutoSyncOnce(ctx context.Context) error {
	sourceID := strings.TrimSpace(a.config.AutoSyncSourceID)
	if sourceID == "" {
		sourceID = a.defaultAutoSyncSourceID()
	}
	if sourceID == "" {
		err := errors.New("no discovery source configured for auto-sync")
		a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
			status.Enabled = a.config.AutoSyncEnabled
			status.LastError = err.Error()
		})
		return err
	}

	startedAt := time.Now().UTC()
	a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
		status.Enabled = a.config.AutoSyncEnabled
		status.Interval = a.config.AutoSyncInterval.String()
		status.RuntimePath = a.config.RuntimeRegistryPath
		status.ConfiguredSourceID = a.config.AutoSyncSourceID
		status.SourceID = sourceID
		status.LastStartedAt = startedAt
		status.LastError = ""
	})

	_, envelope, err := a.fetchDiscoveryEnvelope(ctx, sourceID)
	if err != nil {
		a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
			status.LastError = err.Error()
			status.LastFinishedAt = time.Now().UTC()
		})
		return err
	}

	generated := curation.BuildGeneratedRegistry(envelope.Items)
	next := curation.MergeGeneratedRegistry(a.generatedRegistrySnapshot(), generated, false)
	if err := writeRuntimeRegistry(a.config.RuntimeRegistryPath, next); err != nil {
		a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
			status.LastError = err.Error()
			status.LastFinishedAt = time.Now().UTC()
		})
		return err
	}
	a.applyGeneratedRegistry(next)

	finishedAt := time.Now().UTC()
	a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
		status.SourceID = sourceID
		status.LastFinishedAt = finishedAt
		status.LastError = ""
		status.LastDiscoveryItems = len(envelope.Items)
		status.LastGeneratedAssets = len(next.AssetAliases)
		status.LastGeneratedOverrides = len(next.MarketOverrides)
	})
	log.Printf("market-kit auto-sync refreshed source=%s discovery_items=%d generated_assets=%d generated_overrides=%d", sourceID, len(envelope.Items), len(next.AssetAliases), len(next.MarketOverrides))
	return nil
}

func writeRuntimeRegistry(path string, registry identity.Registry) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func (a *App) defaultAutoSyncSourceID() string {
	if strings.TrimSpace(a.config.SlipstreamDiscoveryURL) == "" && strings.TrimSpace(a.config.SyncSourcesPath) == "" {
		return ""
	}
	for _, source := range a.sources {
		if source.Kind == "discovery" && source.ID != bootstrap.BuiltInSourceID {
			return source.ID
		}
	}
	return ""
}
