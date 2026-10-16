package server

// The console reads immutable snapshots. Search text, facets and ordering are
// built once per published revision, outside ingest and resolver reader locks.
import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

type consoleSymbol struct {
	ID              string  `json:"id"`
	Layer           string  `json:"layer"`
	Platform        string  `json:"platform"`
	PlatformLabel   string  `json:"platformLabel"`
	VenueType       string  `json:"venueType"`
	MarketType      string  `json:"marketType"`
	RawSymbol       string  `json:"rawSymbol"`
	CanonicalSymbol string  `json:"canonicalSymbol"`
	BaseAsset       string  `json:"baseAsset"`
	QuoteAsset      string  `json:"quoteAsset"`
	AssetClass      string  `json:"assetClass"`
	UnitAlias       string  `json:"unitAlias"`
	UnitMultiplier  float64 `json:"unitMultiplier"`
	Chain           string  `json:"chain"`
	Status          string  `json:"status"`
	SourceID        string  `json:"sourceId"`
	search          string
}
type consoleIndex struct {
	resolver     *identity.Resolver
	cachedAt     time.Time
	registry     identity.Registry
	groups       []discovery.AssetCandidateGroup
	assets       []identity.AssetAliasRule
	assetText    []string
	overrideText []string
	groupText    []string
	symbols      []consoleSymbol
	facets       map[string][]string
	marketCount  int
	reviewCount  int
}

func consoleText(parts ...string) string { return strings.ToUpper(strings.Join(parts, " ")) }
func consoleHas(values []string, value string) bool {
	if value == "" || value == "all" {
		return true
	}
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func (a *App) consoleSnapshot() *consoleIndex {
	a.consoleMu.Lock()
	defer a.consoleMu.Unlock()
	a.mu.RLock()
	reg, resolver, at, groups, count := a.registry, a.resolver, a.discoveryCachedAt, a.discoveryCacheGroups, len(a.discoveryCache.Items)
	items := a.discoveryCache.Items
	groupResolver := a.discoveryCacheResolver
	a.mu.RUnlock()
	if idx := a.consoleIndex; idx != nil && idx.resolver == resolver && idx.cachedAt.Equal(at) {
		return idx
	}
	if len(items) > 0 && groupResolver != resolver {
		groups = buildDiscoveryGroups(items, reg)
	}
	idx := &consoleIndex{resolver: resolver, cachedAt: at, registry: reg, groups: groups, marketCount: count, facets: map[string][]string{}}
	facetSets := map[string]map[string]bool{}
	addFacet := func(k, v string) {
		if v != "" {
			if facetSets[k] == nil {
				facetSets[k] = map[string]bool{}
			}
			facetSets[k][v] = true
		}
	}
	classes := map[string]string{}
	idx.assets = append([]identity.AssetAliasRule{}, reg.AssetAliases...)
	sort.Slice(idx.assets, func(i, j int) bool { return idx.assets[i].Canonical < idx.assets[j].Canonical })
	for _, v := range idx.assets {
		parts := append([]string{v.Canonical, v.AssetClass}, v.Aliases...)
		for _, u := range v.UnitAliases {
			parts = append(parts, u.Alias)
		}
		idx.assetText = append(idx.assetText, consoleText(parts...))
		classes[v.Canonical] = v.AssetClass
		addFacet("assetClasses", v.AssetClass)
	}
	seen := map[string]bool{}
	addSymbol := func(v consoleSymbol) {
		if v.RawSymbol == "" || v.BaseAsset == "" || seen[v.ID] {
			return
		}
		seen[v.ID] = true
		v.PlatformLabel = v.Platform
		if v.AssetClass == "" {
			v.AssetClass = "unknown"
		}
		v.search = consoleText(v.RawSymbol, v.CanonicalSymbol, v.BaseAsset, v.QuoteAsset, v.Platform, v.MarketType, v.AssetClass, v.Layer)
		idx.symbols = append(idx.symbols, v)
		addFacet("platforms", v.Platform)
		addFacet("marketTypes", v.MarketType)
		addFacet("symbolAssetClasses", v.AssetClass)
		addFacet("layers", v.Layer)
	}
	for _, v := range reg.MarketOverrides {
		idx.overrideText = append(idx.overrideText, consoleText(v.Exchange, v.RawSymbol, v.CanonicalSymbol, v.MarketType))
		base, quote, _ := strings.Cut(v.CanonicalSymbol, "/")
		addSymbol(consoleSymbol{ID: "registry:" + v.Exchange + ":" + v.MarketType + ":" + v.RawSymbol, Layer: "registry", Platform: v.Exchange, VenueType: "registry", MarketType: v.MarketType, RawSymbol: v.RawSymbol, CanonicalSymbol: v.CanonicalSymbol, BaseAsset: base, QuoteAsset: quote, AssetClass: classes[base], UnitAlias: v.UnitAlias, UnitMultiplier: v.UnitMultiplier, SourceID: "registry"})
	}
	for _, g := range groups {
		parts := []string{g.GroupKey, g.CanonicalAsset, g.AssetClass}
		parts = append(parts, g.Exchanges...)
		addFacet("groupAssetClasses", g.AssetClass)
		for _, v := range g.Exchanges {
			addFacet("groupPlatforms", v)
		}
		for _, v := range g.MarketTypes {
			addFacet("groupMarketTypes", string(v))
		}
		if g.NeedsReview {
			idx.reviewCount++
		}
		for _, m := range g.Markets {
			parts = append(parts, m.RawSymbol, m.Platform, string(m.MarketType), m.VenueType)
			platform := m.Exchange
			if platform == "" {
				platform = m.Platform
			}
			if alias := reg.ExchangeAliases[platform]; alias != "" {
				platform = alias
			}
			addSymbol(consoleSymbol{ID: "discovery:" + platform + ":" + string(m.MarketType) + ":" + m.RawSymbol + ":" + m.BaseAsset + ":" + m.QuoteAsset + ":" + m.Chain, Layer: "discovery", Platform: platform, VenueType: m.VenueType, MarketType: string(m.MarketType), RawSymbol: m.RawSymbol, CanonicalSymbol: m.CanonicalSymbol, BaseAsset: m.BaseAsset, QuoteAsset: m.QuoteAsset, AssetClass: m.AssetClass, Chain: m.Chain, Status: m.Status, SourceID: m.SourceID})
		}
		idx.groupText = append(idx.groupText, consoleText(parts...))
	}
	sort.Slice(idx.symbols, func(i, j int) bool {
		l, r := idx.symbols[i], idx.symbols[j]
		if l.BaseAsset != r.BaseAsset {
			return l.BaseAsset < r.BaseAsset
		}
		if l.Platform != r.Platform {
			return l.Platform < r.Platform
		}
		if l.RawSymbol != r.RawSymbol {
			return l.RawSymbol < r.RawSymbol
		}
		return l.ID < r.ID
	})
	for k, set := range facetSets {
		for v := range set {
			idx.facets[k] = append(idx.facets[k], v)
		}
		sort.Strings(idx.facets[k])
	}
	a.consoleIndex = idx
	return idx
}

type consoleSymbolFilter struct{ platforms, types, classes, layers []string }

func newConsoleSymbolFilter(q url.Values, prefix string) consoleSymbolFilter {
	split := func(key string) []string {
		if q.Get(prefix+key) == "" {
			return nil
		}
		return strings.Split(q.Get(prefix+key), ",")
	}
	return consoleSymbolFilter{split("platform"), split("type"), split("class"), split("layer")}
}
func (f consoleSymbolFilter) matches(v consoleSymbol) bool {
	has := func(values []string, value string) bool {
		if len(values) == 0 {
			return true
		}
		for _, v := range values {
			if v == value {
				return true
			}
		}
		return false
	}
	return has(f.platforms, v.Platform) && has(f.types, v.MarketType) && has(f.classes, v.AssetClass) && has(f.layers, v.Layer)
}
func (a *App) handleConsole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	q := r.URL.Query()
	view := q.Get("view")
	if view == "" {
		view = "registry"
	}
	switch view {
	case "registry", "rules", "groups", "asset-detail", "symbols", "samples", "playground":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown console view"})
		return
	}
	page, err := strconv.Atoi(q.Get("page"))
	if q.Get("page") == "" {
		page = 1
		err = nil
	}
	if err != nil || page < 1 || page > 1000000 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid page"})
		return
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if q.Get("limit") == "" {
		limit = 50
		err = nil
	}
	if err != nil || limit < 1 || limit > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be 1..100"})
		return
	}
	format, field := q.Get("export"), q.Get("field")
	if format != "" {
		validFormat := format == "quotedCsv" || format == "csv" || format == "lines" || format == "json"
		validField := field == "rawSymbol" || field == "canonicalSymbol" || field == "baseAsset"
		if view != "symbols" || !validFormat || !validField {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid export format, field or view"})
			return
		}
	}
	idx := a.consoleSnapshot()
	search := consoleText(strings.TrimSpace(q.Get("q")))
	reg := identity.Registry{ExchangeAliases: idx.registry.ExchangeAliases, AssetAliases: []identity.AssetAliasRule{}, MarketOverrides: []identity.MarketOverride{}}
	groups := []discovery.AssetCandidateGroup{}
	symbols := []consoleSymbol{}
	groupCounts := map[string]int{}
	total := 0
	presenceCount := 0
	start := (page - 1) * limit
	take := func() bool { n := total; total++; return n >= start && n < start+limit }
	switch view {
	case "registry":
		// Exact canonical / alias matches precede substring matches without sorting on every query.
		for pass := 0; pass < 3; pass++ {
			for i, v := range idx.assets {
				if !consoleHas([]string{v.AssetClass}, q.Get("class")) || !strings.Contains(idx.assetText[i], search) {
					continue
				}
				rank := 2
				if search != "" {
					for _, alias := range v.Aliases {
						if alias == search {
							rank = 1
						}
					}
					for _, alias := range v.UnitAliases {
						if alias.Alias == search {
							rank = 1
						}
					}
					if v.Canonical == search {
						rank = 0
					}
				}
				if pass != rank {
					continue
				}
				if take() {
					reg.AssetAliases = append(reg.AssetAliases, v)
				}
			}
		}
	case "rules":
		for i, v := range idx.registry.MarketOverrides {
			if strings.Contains(idx.overrideText[i], search) && take() {
				reg.MarketOverrides = append(reg.MarketOverrides, v)
			}
		}
	case "groups":
		for i, g := range idx.groups {
			if !strings.Contains(idx.groupText[i], search) || !consoleHas([]string{g.AssetClass}, q.Get("class")) || !consoleHas(g.Exchanges, q.Get("platform")) {
				continue
			}
			types := []string{}
			for _, v := range g.MarketTypes {
				types = append(types, string(v))
			}
			if !consoleHas(types, q.Get("type")) {
				continue
			}
			if q.Get("status") == "review" && !g.NeedsReview || q.Get("status") == "ready" && g.NeedsReview {
				continue
			}
			if take() {
				if len(g.Markets) > 3 {
					groupCounts[g.GroupKey] = len(g.Markets)
					g.Markets = g.Markets[:3]
				}
				groups = append(groups, g)
			}
		}
	case "asset-detail":
		asset := strings.ToUpper(strings.TrimSpace(q.Get("asset")))
		for _, v := range idx.assets {
			if v.Canonical == asset {
				reg.AssetAliases = append(reg.AssetAliases, v)
				break
			}
		}
		// Page unique venue contracts, keeping registry and discovery rows for a
		// contract together so the detail UI can retain provenance and overrides.
		keys := map[string]bool{}
		selected := map[string]bool{}
		key := func(p, t, s string) string { return p + ":" + t + ":" + s }
		for _, v := range idx.symbols {
			if v.BaseAsset != asset {
				continue
			}
			k := key(v.Platform, v.MarketType, v.RawSymbol)
			if keys[k] {
				continue
			}
			keys[k] = true
			if take() {
				selected[k] = true
			}
		}
		for _, v := range idx.registry.MarketOverrides {
			if selected[key(v.Exchange, v.MarketType, v.RawSymbol)] {
				reg.MarketOverrides = append(reg.MarketOverrides, v)
			}
		}
		for _, g := range idx.groups {
			if g.CanonicalAsset != asset {
				continue
			}
			markets := []discovery.CandidateMarket{}
			for _, m := range g.Markets {
				platform := m.Exchange
				if platform == "" {
					platform = m.Platform
				}
				if alias := idx.registry.ExchangeAliases[platform]; alias != "" {
					platform = alias
				}
				if selected[key(platform, string(m.MarketType), m.RawSymbol)] {
					markets = append(markets, m)
				}
			}
			if len(markets) > 0 {
				g.Markets = markets
				groups = append(groups, g)
			}
		}
	case "symbols":
		mainFilter, presenceFilter := newConsoleSymbolFilter(q, ""), newConsoleSymbolFilter(q, "presence_")
		presence := map[string]bool{}
		if q.Get("presence") == "true" {
			for _, v := range idx.symbols {
				if presenceFilter.matches(v) {
					presence[v.BaseAsset] = true
				}
			}
		}
		presenceCount = len(presence)
		matches := func(v consoleSymbol) bool {
			return mainFilter.matches(v) && strings.Contains(v.search, search) && (q.Get("presence") != "true" || presence[v.BaseAsset])
		}
		if format != "" {
			writeConsoleSymbolExport(w, r, idx.symbols, matches, field, format)
			return
		}
		for _, v := range idx.symbols {
			if !mainFilter.matches(v) || !strings.Contains(v.search, search) {
				continue
			}
			if q.Get("presence") == "true" && !presence[v.BaseAsset] {
				continue
			}
			if take() {
				symbols = append(symbols, v)
			}
		}
	}
	a.mu.RLock()
	cacheError := a.discoveryCacheErr
	a.mu.RUnlock()
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"registry": reg, "groups": groups, "groupCounts": groupCounts, "symbols": symbols, "total": total, "page": page, "limit": limit, "facets": idx.facets, "stats": map[string]int{"assets": len(idx.assets), "overrides": len(idx.registry.MarketOverrides), "markets": idx.marketCount, "review": idx.reviewCount}, "presenceCount": presenceCount, "cachedAt": idx.cachedAt, "cacheError": cacheError})
}

// Explicit exports contain only the selected output field. They are sent as a
// download rather than loading every market object into browser state.
func writeConsoleSymbolExport(w http.ResponseWriter, r *http.Request, rows []consoleSymbol, matches func(consoleSymbol) bool, field, format string) {
	values := map[string]bool{}
	for _, v := range rows {
		if r.Context().Err() != nil {
			return
		}
		if !matches(v) {
			continue
		}
		value := v.RawSymbol
		switch field {
		case "canonicalSymbol":
			value = v.CanonicalSymbol
		case "baseAsset":
			value = v.BaseAsset
		}
		if value != "" {
			values[value] = true
		}
	}
	sorted := make([]string, 0, len(values))
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="market-kit-symbols.txt"`)
	w.Header().Set("Cache-Control", "no-store")
	out := bufio.NewWriter(w)
	defer out.Flush()
	if format == "json" {
		out.WriteString("[")
	}
	for i, value := range sorted {
		if r.Context().Err() != nil {
			return
		}
		if i > 0 {
			separator := ","
			if format == "lines" {
				separator = "\n"
			}
			if _, err := out.WriteString(separator); err != nil {
				return
			}
		}
		if format == "json" || format == "quotedCsv" {
			encoded, _ := json.Marshal(value)
			value = string(encoded)
		}
		if _, err := out.WriteString(value); err != nil {
			return
		}
	}
	if format == "json" {
		out.WriteString("]")
	}
}
