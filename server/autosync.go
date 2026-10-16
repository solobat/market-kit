package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/solobat/market-kit/bootstrap"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

type AutoSyncStatus struct {
	ManualOverrideCount      int                                  `json:"manualOverrideCount"`
	ReviewedOverrideCount    int                                  `json:"reviewedOverrideCount"`
	ManualPath               string                               `json:"manualPath,omitempty"`
	Enabled                  bool                                 `json:"enabled"`
	Interval                 string                               `json:"interval,omitempty"`
	RuntimePath              string                               `json:"runtimePath,omitempty"`
	RuntimeGeneratedVersion  int                                  `json:"runtimeGeneratedVersion,omitempty"`
	RequiredGeneratedVersion int                                  `json:"requiredGeneratedVersion,omitempty"`
	ConfiguredSourceID       string                               `json:"configuredSourceId,omitempty"`
	SourceID                 string                               `json:"sourceId,omitempty"`
	LastStartedAt            time.Time                            `json:"lastStartedAt,omitempty"`
	LastFinishedAt           time.Time                            `json:"lastFinishedAt,omitempty"`
	LastError                string                               `json:"lastError,omitempty"`
	LastDiscoveryItems       int                                  `json:"lastDiscoveryItems,omitempty"`
	LastGeneratedAssets      int                                  `json:"lastGeneratedAssets,omitempty"`
	LastGeneratedOverrides   int                                  `json:"lastGeneratedOverrides,omitempty"`
	RuntimeAssets            int                                  `json:"runtimeAssets,omitempty"`
	RuntimeOverrides         int                                  `json:"runtimeOverrides,omitempty"`
	SuspiciousCryptoCount    int                                  `json:"suspiciousCryptoCount,omitempty"`
	SuspiciousCryptoAssets   []curation.SuspiciousCryptoCandidate `json:"suspiciousCryptoAssets,omitempty"`
}

type runtimeRegistryLoadResult struct {
	Registry     identity.Registry
	LegacyManual identity.Registry
	Warning      string
}

func loadRuntimeRegistry(path string) (runtimeRegistryLoadResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return runtimeRegistryLoadResult{}, nil
	}
	registry, err := identity.LoadRegistryFile(path)
	if err == nil {
		if isStaleRuntimeRegistry(registry) {
			return runtimeRegistryLoadResult{
				Warning:      staleRuntimeRegistryWarning(path, registry.GeneratedVersion),
				LegacyManual: recoverLegacyProductSettings(registry),
			}, nil
		}
		return runtimeRegistryLoadResult{Registry: registry}, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return runtimeRegistryLoadResult{}, nil
	}
	return runtimeRegistryLoadResult{}, err
}

func isStaleRuntimeRegistry(registry identity.Registry) bool {
	if !hasGeneratedRegistryContent(registry) {
		return false
	}
	return registry.GeneratedVersion < curation.GeneratedRegistryVersion
}

func hasGeneratedRegistryContent(registry identity.Registry) bool {
	return len(registry.ExchangeAliases) > 0 ||
		len(registry.AssetAliases) > 0 ||
		len(registry.MarketOverrides) > 0
}

func staleRuntimeRegistryWarning(path string, version int) string {
	return fmt.Sprintf("runtime registry %s generated_version=%d is older than required generated_version=%d; ignored until regenerated", path, version, curation.GeneratedRegistryVersion)
}

func curationGeneratedRegistryVersion() int {
	return curation.GeneratedRegistryVersion
}

func stampCurrentGeneratedRegistryVersion(registry identity.Registry) identity.Registry {
	if hasGeneratedRegistryContent(registry) && registry.GeneratedVersion < curation.GeneratedRegistryVersion {
		registry.GeneratedVersion = curation.GeneratedRegistryVersion
	}
	return registry
}

func (a *App) handleAutoSync(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.currentAutoSyncStatus())
	case http.MethodPost:
		if !a.requireAdmin(w, r) {
			return
		}
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
	generated = stampCurrentGeneratedRegistryVersion(generated.Clone())
	generated.Normalize()
	runtime := a.baseRegistry.Clone().Merge(generated.Clone())
	runtime = applyRuntimeAssetClassOverrides(runtime, generated)
	runtime = applyRuntimeMarketOverrides(runtime, generated)
	runtime = runtime.Overlay(a.reviewedRegistry)
	runtime = overlayManualRegistry(runtime, a.manualRegistry)
	resolver := identity.NewResolver(runtime)

	a.mu.Lock()
	a.generatedRegistry = generated
	a.registry = runtime
	a.resolver = resolver
	a.autoSyncStatus.ManualOverrideCount = len(a.manualRegistry.MarketOverrides)
	a.autoSyncStatus.ReviewedOverrideCount = len(a.reviewedRegistry.MarketOverrides)
	a.autoSyncStatus.ManualPath = manualRegistryPath(a.config)
	a.autoSyncStatus.RuntimeAssets = len(runtime.AssetAliases)
	a.autoSyncStatus.RuntimeOverrides = len(runtime.MarketOverrides)
	a.autoSyncStatus.RuntimeGeneratedVersion = generated.GeneratedVersion
	a.autoSyncStatus.RequiredGeneratedVersion = curation.GeneratedRegistryVersion
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
	runtimeGeneratedVersion := a.generatedRegistrySnapshot().GeneratedVersion
	a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
		status.Enabled = a.config.AutoSyncEnabled
		status.Interval = a.config.AutoSyncInterval.String()
		status.RuntimePath = a.config.RuntimeRegistryPath
		status.RuntimeGeneratedVersion = runtimeGeneratedVersion
		status.RequiredGeneratedVersion = curation.GeneratedRegistryVersion
		status.ConfiguredSourceID = a.config.AutoSyncSourceID
		status.SourceID = sourceID
		status.LastStartedAt = startedAt
		status.LastError = ""
	})

	envelope, err := a.fetchAutoSyncDiscoveryEnvelope(ctx, sourceID)
	if err != nil {
		a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
			status.LastError = err.Error()
			status.LastFinishedAt = time.Now().UTC()
		})
		return err
	}

	generated := curation.BuildGeneratedRegistry(envelope.Items)
	a.registryUpdateMu.Lock()
	next := curation.MergeGeneratedRegistry(a.generatedRegistrySnapshot().Clone(), generated, false)
	suspiciousCrypto := curation.SuspiciousCryptoCandidates(envelope.Items, next, len(envelope.Items))
	if err := writeRuntimeRegistry(a.config.RuntimeRegistryPath, overlayManualRegistry(next, a.manualRegistry)); err != nil {
		a.registryUpdateMu.Unlock()
		a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
			status.LastError = err.Error()
			status.LastFinishedAt = time.Now().UTC()
		})
		return err
	}
	a.applyGeneratedRegistry(next)
	a.registryUpdateMu.Unlock()
	if err := a.refreshDiscoveryCache(ctx, "all"); err != nil {
		log.Printf("market-kit discovery cache refresh failed: %v", err)
	}

	finishedAt := time.Now().UTC()
	a.updateAutoSyncStatus(func(status *AutoSyncStatus) {
		status.SourceID = sourceID
		status.LastFinishedAt = finishedAt
		status.LastError = ""
		status.LastDiscoveryItems = len(envelope.Items)
		status.LastGeneratedAssets = len(next.AssetAliases)
		status.LastGeneratedOverrides = len(next.MarketOverrides)
		status.RuntimeGeneratedVersion = next.GeneratedVersion
		status.RequiredGeneratedVersion = curation.GeneratedRegistryVersion
		status.SuspiciousCryptoCount = len(suspiciousCrypto)
		status.SuspiciousCryptoAssets = suspiciousCrypto
		if len(status.SuspiciousCryptoAssets) > 25 {
			status.SuspiciousCryptoAssets = status.SuspiciousCryptoAssets[:25]
		}
	})
	log.Printf("market-kit auto-sync refreshed source=%s discovery_items=%d generated_assets=%d generated_overrides=%d", sourceID, len(envelope.Items), len(next.AssetAliases), len(next.MarketOverrides))
	return nil
}

func (a *App) fetchAutoSyncDiscoveryEnvelope(ctx context.Context, sourceID string) (discovery.ImportEnvelope, error) {
	if strings.EqualFold(strings.TrimSpace(sourceID), "all") {
		sources, envelope, _, _, _, err := a.discoveryLookupEnvelopeFromCacheOrRefresh(ctx, "all")
		if err != nil {
			return discovery.ImportEnvelope{}, err
		}
		if len(sources) == 0 {
			return discovery.ImportEnvelope{}, errors.New("no discovery source configured for auto-sync")
		}
		return envelope, nil
	}

	_, envelope, err := a.fetchDiscoveryEnvelope(ctx, sourceID)
	return envelope, err
}

func writeRuntimeRegistry(path string, registry identity.Registry) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	registry = stampCurrentGeneratedRegistryVersion(registry)
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".market-kit-registry-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0o644); err == nil {
		_, err = tmp.Write(payload)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
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
