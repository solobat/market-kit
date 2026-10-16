package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/solobat/market-kit/identity"
)

// The sidecar contains only explicit human edits. Generated version changes and
// discovery refreshes must never promote the full generated registry to manual.
func manualRegistryPath(config Config) string {
	if strings.TrimSpace(config.RuntimeRegistryPath) == "" {
		return ""
	}
	return config.RuntimeRegistryPath + ".manual.json"
}
func loadManualRegistry(config Config) (identity.Registry, error) {
	path := manualRegistryPath(config)
	if path == "" {
		return identity.Registry{}, nil
	}
	r, err := identity.LoadRegistryFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return identity.Registry{}, nil
	}
	return r, err
}

type invalidRegistryMutation struct{ err error }

func (e invalidRegistryMutation) Error() string { return e.err.Error() }
func writeRegistryMutationError(w http.ResponseWriter, err error) {
	var invalid invalidRegistryMutation
	if errors.As(err, &invalid) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "persist manual registry failed"})
}

// overlayManualRegistry applies explicit asset corrections and then per-market
// corrections. Global class edits never override a discovered ticker collision.
func overlayManualRegistry(runtime, manual identity.Registry) identity.Registry {
	if !hasGeneratedRegistryContent(manual) {
		return runtime
	}
	manual = manual.Clone()
	collisions := map[string]bool{}
	for _, v := range runtime.AssetCollisions {
		collisions[v.Symbol] = true
	}
	classes := map[string]string{}
	aliases := manual.AssetAliases[:0]
	for _, v := range manual.AssetAliases {
		if !collisions[v.Canonical] {
			aliases = append(aliases, v)
			classes[v.Canonical] = v.AssetClass
		}
	}
	manual.AssetAliases = aliases
	runtime = runtime.Clone()
	for i := range runtime.MarketOverrides {
		v := &runtime.MarketOverrides[i]
		base, quote, ok := splitCanonicalSymbol(v.CanonicalSymbol)
		if !ok {
			continue
		}
		class, ok := classes[base]
		if !ok || class == "" {
			continue
		}
		oldID := v.AssetID
		oldUnderlying := v.UnderlyingID
		oldClass := v.AssetClass
		v.AssetClass = class
		if oldID == "" || oldID == identity.CanonicalAssetID(oldClass, base) || strings.HasPrefix(oldID, "unknown:") {
			v.AssetID = identity.CanonicalAssetID(class, base)
		}
		if oldUnderlying == "" || oldUnderlying == oldID || oldUnderlying == identity.CanonicalAssetID(oldClass, base) || strings.HasPrefix(oldUnderlying, "unknown:") {
			v.UnderlyingID = v.AssetID
		}
		if v.ComparisonKey == "" || v.ComparisonKey == identity.DefaultComparisonKey(oldID, quote) || v.ComparisonKey == identity.DefaultComparisonKey(oldUnderlying, quote) {
			v.ComparisonKey = identity.DefaultComparisonKey(v.UnderlyingID, quote)
		}
		v.InstrumentKind = identity.DefaultInstrumentKind(identity.MarketType(v.MarketType), class)
	}
	return runtime.Overlay(manual)
}

// Caller holds registryUpdateMu. Publish only complete immutable snapshots.
func (a *App) persistManualRegistry(manual identity.Registry) error {
	manual = stampCurrentGeneratedRegistryVersion(manual.Clone())
	manual.Normalize()
	if err := writeRuntimeRegistry(manualRegistryPath(a.config), manual); err != nil {
		return err
	}
	// This compatibility snapshot can be regenerated; the sidecar is authoritative.
	// If it fails, the acknowledged edit is still durable in the sidecar.
	generated := a.generatedRegistrySnapshot().Clone()
	if err := writeRuntimeRegistry(a.config.RuntimeRegistryPath, overlayManualRegistry(generated, manual)); err != nil {
		log.Printf("market-kit compatibility snapshot write failed; manual sidecar is durable: %v", err)
	}
	a.manualRegistry = manual
	a.applyGeneratedRegistry(generated)
	return nil
}

func (a *App) commitManualOverrides(requests []registryOverrideRequest) ([]identity.MarketOverride, error) {
	a.registryUpdateMu.Lock()
	defer a.registryUpdateMu.Unlock()
	manual := a.manualRegistry.Clone()
	runtime := a.runtimeRegistry()
	known := map[string]bool{}
	for _, v := range runtime.AssetAliases {
		known[v.Canonical] = true
	}
	for _, v := range manual.AssetAliases {
		known[v.Canonical] = true
	}
	index := map[string]int{}
	for i, v := range manual.MarketOverrides {
		index[registryOverrideKey(v)] = i
	}
	targets := make([]identity.MarketOverride, 0, len(requests))
	changed := false
	for i, req := range requests {
		v, base, err := normalizeRegistryOverrideRequest(req)
		if err != nil {
			return nil, invalidRegistryMutation{fmt.Errorf("item %d: %w", i, err)}
		}
		class := normalizeRegistryAssetClass(req.AssetClass)
		if strings.TrimSpace(req.AssetClass) != "" && class == "" {
			return nil, invalidRegistryMutation{errors.New("assetClass must be crypto, rwa_stock, rwa_commodity, fiat_stable, or unknown")}
		}
		if !known[base] {
			if class == "" {
				return nil, invalidRegistryMutation{errors.New("assetClass is required when assigning a market to a new base asset")}
			}
			manual.AssetAliases = append(manual.AssetAliases, identity.AssetAliasRule{Canonical: base, AssetClass: class, AssetID: identity.CanonicalAssetID(class, base), UnderlyingID: identity.CanonicalAssetID(class, base)})
			known[base] = true
			changed = true
		}
		k := registryOverrideKey(v)
		if pos, ok := index[k]; ok {
			if manual.MarketOverrides[pos] != v {
				manual.MarketOverrides[pos] = v
				changed = true
			}
		} else {
			index[k] = len(manual.MarketOverrides)
			manual.MarketOverrides = append(manual.MarketOverrides, v)
			changed = true
		}
		targets = append(targets, v)
	}
	if changed {
		if err := a.persistManualRegistry(manual); err != nil {
			return nil, err
		}
	}
	published := a.runtimeRegistry()
	lookup := map[string]identity.MarketOverride{}
	for _, v := range published.MarketOverrides {
		lookup[registryOverrideKey(v)] = v
	}
	for i, v := range targets {
		if actual, ok := lookup[registryOverrideKey(v)]; ok {
			targets[i] = actual
		}
	}
	return targets, nil
}

func (a *App) handleRuntimeRegistryOverrideBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if !a.requireAdmin(w, r) {
		return
	}
	var req struct {
		Items []registryOverrideRequest `json:"items"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil || len(req.Items) == 0 || len(req.Items) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "items must contain 1 to 500 registry overrides"})
		return
	}
	overrides, err := a.commitManualOverrides(req.Items)
	if err != nil {
		writeRegistryMutationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(overrides), "overrides": overrides})
}

// Automatic discovery in older versions did not emit fractional split/rebase,
// dated-contract or prohibited-comparison settings. Preserve these on upgrade.
func recoverLegacyProductSettings(legacy identity.Registry) identity.Registry {
	out := identity.Registry{}
	for _, v := range legacy.MarketOverrides {
		if (v.UnitMultiplier > 0 && v.UnitMultiplier < 1) || v.ExpiryAtMs > 0 || v.ContractType != "" || v.ComparisonStatus == identity.ComparisonProhibited {
			out.MarketOverrides = append(out.MarketOverrides, v)
		}
	}
	return out
}
