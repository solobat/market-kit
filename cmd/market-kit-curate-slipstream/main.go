package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/solobat/market-kit/bootstrap"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

type discoveryEnvelope struct {
	Source string          `json:"source"`
	Items  []discoveryItem `json:"items"`
}

type discoveryItem struct {
	PlatformID         string   `json:"platformId"`
	VenueType          string   `json:"venueType"`
	MarketType         string   `json:"marketType"`
	Symbol             string   `json:"symbol"`
	BaseAsset          string   `json:"baseAsset"`
	QuoteAsset         string   `json:"quoteAsset"`
	Status             string   `json:"status"`
	AssetClass         string   `json:"assetClass"`
	AssetClassHint     string   `json:"assetClassHint"`
	Category           string   `json:"category"`
	UnderlyingCategory string   `json:"underlyingCategory"`
	Sector             string   `json:"sector"`
	Tags               []string `json:"tags"`
}

type repeatedHeaders []string

var (
	stableAssets = map[string]bool{
		"USDT": true, "USDC": true, "USD": true, "FDUSD": true, "USDE": true, "USDS": true,
		"DAI": true, "BUSD": true, "USDB": true, "USD1": true, "USDY": true, "RLUSD": true,
		"EURC": true, "EURT": true, "AEUR": true, "EURI": true, "PYUSD": true, "TUSD": true,
		"USDG": true, "USDGO": true, "USDTB": true, "WUSD": true, "XUSD": true,
	}
	rwaStockAssets = map[string]bool{
		"AAPL": true, "AAPLX": true, "AAOI": true, "AMAT": true, "AMD": true, "AMZN": true,
		"APLD": true, "APP": true, "ARM": true, "ASML": true, "AVGO": true, "BA": true,
		"BABA": true, "BILL": true, "COHR": true, "COIN": true, "COST": true, "CRCL": true,
		"CRWV": true, "CSCO": true, "D": true, "DAR": true, "DIA": true,
		"DIS": true, "DRAM": true, "EAT": true, "EWH": true, "EWJ": true, "EWT": true,
		"EWY": true, "F": true, "FIS": true, "FUTU": true, "GE": true, "GOOGL": true,
		"HD": true, "HEI": true, "HIMS": true, "HOOD": true, "IAG": true, "IAU": true,
		"INDA": true, "INTC": true, "IONQ": true, "ITOT": true, "IVV": true, "IWM": true,
		"JD": true, "KLAC": true, "KOPN": true, "KWEB": true, "LLY": true, "LWLG": true,
		"MAS": true, "MCD": true, "META": true, "MP": true, "MPLX": true, "MRVL": true,
		"MSFT": true, "MSTR": true, "MU": true, "NBIS": true, "NFLX": true, "NIO": true,
		"NVDA": true, "OKLO": true, "ORCL": true, "OXY": true, "PAYP": true, "PLTR": true,
		"QCOM": true, "QQQ": true, "RKLB": true, "RTX": true, "SLV": true, "SNDK": true,
		"SOXL": true, "SOXS": true, "SPACEX": true, "SPCX": true, "SPY": true, "SQQQ": true,
		"STXSTOCK": true, "TCOM": true, "TSLA": true, "TSM": true, "UBER": true,
		"UNH": true, "USO": true, "WDC": true, "XLE": true, "XOM": true,
	}
	rwaCommodityAssets = map[string]bool{
		"BRENT": true, "BRENTOIL": true, "CL": true, "GOLD": true, "NATGAS": true, "NG": true,
		"PAXG": true, "SILVER": true, "WTI": true, "WTIOIL": true, "XAG": true, "XAGT": true,
		"XAU": true, "XAUT": true,
	}
)

func main() {
	inputPath := flag.String("input", "", "Path to slipstream discovery export JSON")
	sourceURL := flag.String("url", "", "Remote discovery export URL. Used when --input is empty.")
	bootstrapFlag := flag.Bool("bootstrap", false, "Fetch discovery directly from built-in exchange collectors.")
	bootstrapSources := flag.String("sources", "", "Comma-separated built-in exchange collector ids for --bootstrap, e.g. binance,bybit,okx,bitget,gate,hyperliquid")
	sourceName := flag.String("source-name", "slipstream", "Human-readable source name for review output")
	outputPath := flag.String("output", filepath.Join("identity", "generated_registry.json"), "Path to write generated registry JSON")
	prune := flag.Bool("prune", false, "Replace output exactly with the current source. By default, existing generated rules are preserved and new rules are merged in.")
	reviewOutputPath := flag.String("review-output", filepath.Join("identity", "generated_registry.review.md"), "Path to write a compact human review report. Set empty to disable.")
	reviewLimit := flag.Int("review-limit", 80, "Maximum changed rows to include per review section")
	timeoutFlag := flag.Duration("timeout", 30*time.Second, "HTTP timeout for --url")
	var headers repeatedHeaders
	flag.Var(&headers, "header", "HTTP header for --url in 'Name: Value' form. May be repeated.")
	flag.Parse()

	payload, err := loadDiscoveryPayload(discoveryLoadOptions{
		InputPath:        *inputPath,
		SourceURL:        *sourceURL,
		UseBootstrap:     *bootstrapFlag,
		BootstrapSources: splitCSV(*bootstrapSources),
		Headers:          headers,
		Timeout:          *timeoutFlag,
	})
	if err != nil {
		fatalf("%v", err)
	}
	if *bootstrapFlag && strings.TrimSpace(*sourceName) == "slipstream" {
		*sourceName = bootstrap.BuiltInSourceID
	}

	var existing identity.Registry
	if strings.TrimSpace(*outputPath) != "" {
		if loaded, err := identity.LoadRegistryFile(*outputPath); err == nil {
			existing = loaded
		}
	}

	envelope, err := decodeDiscoveryEnvelope(payload)
	if err != nil {
		fatalf("decode input: %v", err)
	}

	generated := buildGeneratedRegistry(envelope.Items)
	existing = sanitizeExistingGeneratedRegistry(existing, generated)
	registry := generated
	if !*prune {
		registry = existing.Merge(generated)
	}
	encoded, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		fatalf("encode registry: %v", err)
	}
	encoded = append(encoded, '\n')

	if err := os.WriteFile(*outputPath, encoded, 0o644); err != nil {
		fatalf("write output: %v", err)
	}

	review := buildReviewReport(existing, registry, envelope.Items, strings.TrimSpace(*sourceName), *reviewLimit)
	if strings.TrimSpace(*reviewOutputPath) != "" {
		if err := os.WriteFile(*reviewOutputPath, []byte(review), 0o644); err != nil {
			fatalf("write review output: %v", err)
		}
	}

	fmt.Printf("wrote %s with %d asset aliases and %d market overrides\n", *outputPath, len(registry.AssetAliases), len(registry.MarketOverrides))
	if strings.TrimSpace(*reviewOutputPath) != "" {
		fmt.Printf("wrote review report %s\n", *reviewOutputPath)
	}
}

func decodeDiscoveryEnvelope(payload []byte) (discoveryEnvelope, error) {
	var wrapped struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(payload, &wrapped); err == nil && len(wrapped.Payload) > 0 {
		var envelope discoveryEnvelope
		if err := json.Unmarshal(wrapped.Payload, &envelope); err != nil {
			return discoveryEnvelope{}, err
		}
		return envelope, nil
	}

	var envelope discoveryEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return discoveryEnvelope{}, err
	}
	return envelope, nil
}

func (h *repeatedHeaders) String() string {
	if h == nil {
		return ""
	}
	return strings.Join(*h, ", ")
}

func (h *repeatedHeaders) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if !strings.Contains(value, ":") {
		return fmt.Errorf("header must be in 'Name: Value' form")
	}
	*h = append(*h, value)
	return nil
}

type discoveryLoadOptions struct {
	InputPath        string
	SourceURL        string
	UseBootstrap     bool
	BootstrapSources []string
	Headers          []string
	Timeout          time.Duration
}

func loadDiscoveryPayload(options discoveryLoadOptions) ([]byte, error) {
	inputPath := strings.TrimSpace(options.InputPath)
	sourceURL := strings.TrimSpace(options.SourceURL)
	sourceCount := 0
	if inputPath != "" {
		sourceCount++
	}
	if sourceURL != "" {
		sourceCount++
	}
	if options.UseBootstrap {
		sourceCount++
	}
	if sourceCount == 0 {
		return nil, fmt.Errorf("missing discovery source: pass --bootstrap, --input, or --url")
	}
	if sourceCount > 1 {
		return nil, fmt.Errorf("pass only one discovery source: --bootstrap, --input, or --url")
	}
	if options.UseBootstrap {
		client := &http.Client{Timeout: options.Timeout}
		envelope, err := bootstrap.Fetch(context.Background(), client, options.BootstrapSources)
		if err != nil {
			return nil, fmt.Errorf("fetch bootstrap discovery: %w", err)
		}
		payload, err := json.Marshal(envelope)
		if err != nil {
			return nil, fmt.Errorf("encode bootstrap discovery: %w", err)
		}
		return payload, nil
	}
	if inputPath != "" {
		payload, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("read input: %w", err)
		}
		return payload, nil
	}

	client := &http.Client{Timeout: options.Timeout}
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build source request: %w", err)
	}
	for _, header := range options.Headers {
		key, value, ok := strings.Cut(header, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header %q", header)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid empty header name")
		}
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch source: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("read source response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch source failed: %s - %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func buildGeneratedRegistry(items []discoveryItem) identity.Registry {
	assets := map[string]identity.AssetAliasRule{}
	overrides := map[string]identity.MarketOverride{}

	for _, item := range items {
		if !shouldInclude(item) {
			continue
		}

		exchange := strings.ToLower(strings.TrimSpace(item.PlatformID))
		base := strings.ToUpper(strings.TrimSpace(item.BaseAsset))
		quote := strings.ToUpper(strings.TrimSpace(item.QuoteAsset))
		symbol := strings.TrimSpace(item.Symbol)
		marketType := normalizeGeneratedMarketType(item.MarketType)

		if exchange == "" || base == "" || quote == "" || symbol == "" || marketType == "" {
			continue
		}

		ensureAsset(assets, base, classifyGeneratedAsset(item, base))
		ensureAsset(assets, quote, classifyGeneratedAsset(item, quote))

		override := identity.MarketOverride{
			Exchange:        exchange,
			RawSymbol:       symbol,
			MarketType:      marketType,
			CanonicalSymbol: base + "/" + quote,
		}
		overrides[overrideKey(override)] = override
	}

	assetList := make([]identity.AssetAliasRule, 0, len(assets))
	for _, item := range assets {
		assetList = append(assetList, item)
	}
	sort.Slice(assetList, func(i, j int) bool {
		return assetList[i].Canonical < assetList[j].Canonical
	})

	overrideList := make([]identity.MarketOverride, 0, len(overrides))
	for _, item := range overrides {
		overrideList = append(overrideList, item)
	}
	sort.Slice(overrideList, func(i, j int) bool {
		if overrideList[i].Exchange == overrideList[j].Exchange {
			if overrideList[i].RawSymbol == overrideList[j].RawSymbol {
				return overrideList[i].MarketType < overrideList[j].MarketType
			}
			return overrideList[i].RawSymbol < overrideList[j].RawSymbol
		}
		return overrideList[i].Exchange < overrideList[j].Exchange
	})

	registry := identity.Registry{
		ExchangeAliases: map[string]string{},
		AssetAliases:    assetList,
		MarketOverrides: overrideList,
	}
	registry.Normalize()
	return registry
}

func sanitizeExistingGeneratedRegistry(existing identity.Registry, current identity.Registry) identity.Registry {
	current.Normalize()
	supportedRWA := map[string]string{}
	for _, item := range current.AssetAliases {
		if item.AssetClass == "rwa_stock" || item.AssetClass == "rwa_commodity" {
			supportedRWA[item.Canonical] = item.AssetClass
		}
	}

	out := identity.Registry{
		ExchangeAliases: existing.ExchangeAliases,
		MarketOverrides: existing.MarketOverrides,
		AssetAliases:    make([]identity.AssetAliasRule, 0, len(existing.AssetAliases)),
	}
	for _, item := range existing.AssetAliases {
		switch item.AssetClass {
		case "rwa_stock", "rwa_commodity":
			if preciseGeneratedAssetClass(item.Canonical) == item.AssetClass || supportedRWA[item.Canonical] == item.AssetClass {
				out.AssetAliases = append(out.AssetAliases, item)
			}
		default:
			out.AssetAliases = append(out.AssetAliases, item)
		}
	}
	out.Normalize()
	return out
}

func preciseGeneratedAssetClass(asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	switch {
	case stableAssets[asset]:
		return "fiat_stable"
	case rwaStockAssets[asset]:
		return "rwa_stock"
	case rwaCommodityAssets[asset]:
		return "rwa_commodity"
	default:
		return ""
	}
}

func shouldInclude(item discoveryItem) bool {
	venueType := strings.ToLower(strings.TrimSpace(item.VenueType))
	if venueType != "cex" && venueType != "dex" {
		return false
	}
	if discovery.IsExcludedLeveragedToken(item.PlatformID, item.BaseAsset, item.Symbol) {
		return false
	}
	quote := strings.ToUpper(strings.TrimSpace(item.QuoteAsset))
	if !stableAssets[quote] {
		return false
	}
	marketType := strings.ToLower(strings.TrimSpace(item.MarketType))
	return marketType == "spot" || marketType == "perp" || marketType == "perpetual"
}

func buildReviewReport(existing identity.Registry, next identity.Registry, items []discoveryItem, sourceName string, limit int) string {
	if sourceName == "" {
		sourceName = "discovery"
	}
	if limit <= 0 {
		limit = 80
	}
	existing.Normalize()
	next.Normalize()

	addedOverrides := diffMarketOverrides(next.MarketOverrides, existing.MarketOverrides)
	removedOverrides := diffMarketOverrides(existing.MarketOverrides, next.MarketOverrides)
	addedAssets := diffAssetAliases(next.AssetAliases, existing.AssetAliases)
	removedAssets := diffAssetAliases(existing.AssetAliases, next.AssetAliases)
	reviewItems := reviewCandidates(addedOverrides, items, limit)
	stats := discoveryStats(items)

	var b bytes.Buffer
	fmt.Fprintf(&b, "# Generated Registry Review\n\n")
	fmt.Fprintf(&b, "- Source: `%s`\n", sourceName)
	fmt.Fprintf(&b, "- Generated at: `%s`\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- Discovery rows: `%d`\n", len(items))
	fmt.Fprintf(&b, "- Included stable-quoted rows: `%d`\n", stats.Included)
	fmt.Fprintf(&b, "- Skipped non-CEX rows: `%d`\n", stats.SkippedNonCEX)
	fmt.Fprintf(&b, "- Skipped non-stable quote rows: `%d`\n", stats.SkippedNonStableQuote)
	fmt.Fprintf(&b, "- Skipped leveraged rows: `%d`\n", stats.SkippedLeveraged)
	fmt.Fprintf(&b, "- Generated assets: `%d` (`+%d`, `-%d`)\n", len(next.AssetAliases), len(addedAssets), len(removedAssets))
	fmt.Fprintf(&b, "- Generated overrides: `%d` (`+%d`, `-%d`)\n\n", len(next.MarketOverrides), len(addedOverrides), len(removedOverrides))

	fmt.Fprintf(&b, "## Needs Human Review\n\n")
	if len(reviewItems.RWAAssets) == 0 && len(removedOverrides) == 0 {
		fmt.Fprintf(&b, "No high-signal review candidates. The change set is suitable for automated registry refresh.\n\n")
	} else {
		writeOverrideTable(&b, "New RWA / Commodity Overrides", reviewItems.RWAAssets, limit)
		writeOverrideTable(&b, "Removed Overrides", removedOverrides, limit)
	}

	fmt.Fprintf(&b, "## Automation Notes\n\n")
	fmt.Fprintf(&b, "- New unknown-class overrides are promoted only as exchange-explicit base/quote mappings; they do not create asset aliases or asset classes.\n")
	fmt.Fprintf(&b, "- New unknown-class override count: `%d`\n\n", reviewItems.UnknownAssetClassCount)

	fmt.Fprintf(&b, "## Added Overrides\n\n")
	writeOverrideTable(&b, "", addedOverrides, limit)
	fmt.Fprintf(&b, "## Added Asset Aliases\n\n")
	writeAssetTable(&b, addedAssets, limit)
	return b.String()
}

type registryReviewStats struct {
	Included              int
	SkippedNonCEX         int
	SkippedNonStableQuote int
	SkippedLeveraged      int
}

func discoveryStats(items []discoveryItem) registryReviewStats {
	var stats registryReviewStats
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.VenueType), "cex") {
			stats.SkippedNonCEX++
			continue
		}
		if discovery.IsExcludedLeveragedToken(item.PlatformID, item.BaseAsset, item.Symbol) {
			stats.SkippedLeveraged++
			continue
		}
		quote := strings.ToUpper(strings.TrimSpace(item.QuoteAsset))
		if !stableAssets[quote] {
			stats.SkippedNonStableQuote++
			continue
		}
		if normalizeGeneratedMarketType(item.MarketType) == "" {
			continue
		}
		stats.Included++
	}
	return stats
}

type reviewCandidateSet struct {
	UnknownAssetClassCount int
	RWAAssets              []identity.MarketOverride
}

func reviewCandidates(overrides []identity.MarketOverride, items []discoveryItem, limit int) reviewCandidateSet {
	byOverride := discoveryItemIndex(items)
	out := reviewCandidateSet{}
	for _, override := range overrides {
		base, _ := splitCanonicalSymbol(override.CanonicalSymbol)
		if stableAssets[base] {
			continue
		}
		item := byOverride[overrideKey(override)]
		class := classifyGeneratedAsset(item, base)
		switch class {
		case "":
			out.UnknownAssetClassCount++
		case "rwa_stock", "rwa_commodity":
			out.RWAAssets = append(out.RWAAssets, override)
		}
	}
	sortOverrides(out.RWAAssets)
	if len(out.RWAAssets) > limit {
		out.RWAAssets = out.RWAAssets[:limit]
	}
	return out
}

func discoveryItemIndex(items []discoveryItem) map[string]discoveryItem {
	out := map[string]discoveryItem{}
	for _, item := range items {
		if !shouldInclude(item) {
			continue
		}
		override := identity.MarketOverride{
			Exchange:        strings.ToLower(strings.TrimSpace(item.PlatformID)),
			RawSymbol:       strings.TrimSpace(item.Symbol),
			MarketType:      normalizeGeneratedMarketType(item.MarketType),
			CanonicalSymbol: strings.ToUpper(strings.TrimSpace(item.BaseAsset)) + "/" + strings.ToUpper(strings.TrimSpace(item.QuoteAsset)),
		}
		out[overrideKey(override)] = item
	}
	return out
}

func diffMarketOverrides(left []identity.MarketOverride, right []identity.MarketOverride) []identity.MarketOverride {
	rightKeys := map[string]bool{}
	for _, item := range right {
		rightKeys[overrideKey(item)] = true
	}
	out := make([]identity.MarketOverride, 0)
	for _, item := range left {
		if rightKeys[overrideKey(item)] {
			continue
		}
		out = append(out, item)
	}
	sortOverrides(out)
	return out
}

func diffAssetAliases(left []identity.AssetAliasRule, right []identity.AssetAliasRule) []identity.AssetAliasRule {
	rightKeys := map[string]bool{}
	for _, item := range right {
		rightKeys[item.Canonical] = true
	}
	out := make([]identity.AssetAliasRule, 0)
	for _, item := range left {
		if rightKeys[item.Canonical] {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Canonical < out[j].Canonical
	})
	return out
}

func sortOverrides(items []identity.MarketOverride) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Exchange == items[j].Exchange {
			if items[i].RawSymbol == items[j].RawSymbol {
				return items[i].MarketType < items[j].MarketType
			}
			return items[i].RawSymbol < items[j].RawSymbol
		}
		return items[i].Exchange < items[j].Exchange
	})
}

func writeOverrideTable(b *bytes.Buffer, title string, rows []identity.MarketOverride, limit int) {
	if title != "" {
		fmt.Fprintf(b, "### %s\n\n", title)
	}
	if len(rows) == 0 {
		fmt.Fprintf(b, "None.\n\n")
		return
	}
	total := len(rows)
	if total > limit {
		rows = rows[:limit]
	}
	fmt.Fprintf(b, "| Exchange | Raw Symbol | Market Type | Canonical Symbol |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(b, "| `%s` | `%s` | `%s` | `%s` |\n", row.Exchange, row.RawSymbol, row.MarketType, row.CanonicalSymbol)
	}
	if total > len(rows) {
		fmt.Fprintf(b, "\n_%d more omitted by --review-limit._\n", total-len(rows))
	}
	fmt.Fprintf(b, "\n")
}

func writeAssetTable(b *bytes.Buffer, rows []identity.AssetAliasRule, limit int) {
	if len(rows) == 0 {
		fmt.Fprintf(b, "None.\n\n")
		return
	}
	total := len(rows)
	if total > limit {
		rows = rows[:limit]
	}
	fmt.Fprintf(b, "| Canonical | Asset Class |\n")
	fmt.Fprintf(b, "| --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(b, "| `%s` | `%s` |\n", row.Canonical, row.AssetClass)
	}
	if total > len(rows) {
		fmt.Fprintf(b, "\n_%d more omitted by --review-limit._\n", total-len(rows))
	}
	fmt.Fprintf(b, "\n")
}

func normalizeGeneratedMarketType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "spot":
		return "spot"
	case "perp", "perpetual", "swap", "linear":
		return "perpetual"
	case "future", "futures", "delivery":
		return "future"
	default:
		return ""
	}
}

func ensureAsset(target map[string]identity.AssetAliasRule, canonical string, assetClass string) {
	canonical = strings.ToUpper(strings.TrimSpace(canonical))
	assetClass = strings.TrimSpace(assetClass)
	if canonical == "" || assetClass == "" {
		return
	}
	if _, exists := target[canonical]; exists {
		return
	}
	target[canonical] = identity.AssetAliasRule{
		Canonical:  canonical,
		AssetClass: assetClass,
		Aliases:    []string{},
	}
}

func classifyGeneratedAsset(item discoveryItem, asset string) string {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if asset == "" {
		return ""
	}
	if stableAssets[asset] {
		return "fiat_stable"
	}
	if fromExchange := inferAssetClassFromExchangeMetadata(item, asset); fromExchange != "" {
		return fromExchange
	}
	return inferAssetClassFallback(asset)
}

func inferAssetClassFromExchangeMetadata(item discoveryItem, asset string) string {
	hints := []string{
		item.AssetClass,
		item.AssetClassHint,
		item.Category,
		item.UnderlyingCategory,
		item.Sector,
	}
	hints = append(hints, item.Tags...)

	for _, hint := range hints {
		if assetClass := normalizeAssetClassHint(hint); assetClass != "" {
			return assetClass
		}
	}
	return ""
}

func normalizeAssetClassHint(value string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	raw = strings.ReplaceAll(raw, "_", " ")
	raw = strings.ReplaceAll(raw, "-", " ")
	raw = strings.Join(strings.Fields(raw), " ")

	switch {
	case raw == "":
		return ""
	case strings.Contains(raw, "stable"):
		return "fiat_stable"
	case strings.Contains(raw, "stock"),
		strings.Contains(raw, "equity"),
		strings.Contains(raw, "share"),
		strings.Contains(raw, "security"),
		strings.Contains(raw, "etf"),
		strings.Contains(raw, "index"):
		return "rwa_stock"
	case strings.Contains(raw, "commodity"),
		strings.Contains(raw, "metal"),
		strings.Contains(raw, "gold"),
		strings.Contains(raw, "silver"),
		strings.Contains(raw, "oil"),
		strings.Contains(raw, "crude"),
		strings.Contains(raw, "gas"),
		strings.Contains(raw, "energy"):
		return "rwa_commodity"
	case strings.Contains(raw, "crypto"),
		strings.Contains(raw, "blockchain"),
		strings.Contains(raw, "defi"),
		strings.Contains(raw, "meme"),
		strings.Contains(raw, "layer 1"),
		strings.Contains(raw, "layer1"),
		strings.Contains(raw, "token"),
		strings.Contains(raw, "coin"):
		return "crypto"
	default:
		return ""
	}
}

func inferAssetClassFallback(asset string) string {
	switch {
	case stableAssets[asset]:
		return "fiat_stable"
	case rwaStockAssets[asset]:
		return "rwa_stock"
	case rwaCommodityAssets[asset]:
		return "rwa_commodity"
	default:
		return ""
	}
}

func overrideKey(item identity.MarketOverride) string {
	return item.Exchange + "|" + item.RawSymbol + "|" + item.MarketType
}

func splitCanonicalSymbol(value string) (base string, quote string) {
	parts := strings.SplitN(strings.ToUpper(strings.TrimSpace(value)), "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
