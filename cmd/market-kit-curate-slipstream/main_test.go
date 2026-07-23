package main

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/identity"
)

func TestBuildGeneratedRegistryFiltersToStableQuotedMarkets(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "okx",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "DRAM-USDT-SWAP",
			BaseAsset:  "DRAM",
			QuoteAsset: "USDT",
			Status:     "live",
		},
		{
			PlatformID: "hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "DRAM",
			BaseAsset:  "DRAM",
			QuoteAsset: "USDT",
			Status:     "live",
		},
		{
			PlatformID: "okx",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC-EUR",
			BaseAsset:  "BTC",
			QuoteAsset: "EUR",
			Status:     "live",
		},
	})

	if len(registry.MarketOverrides) != 2 {
		t.Fatalf("expected 2 generated overrides, got %d", len(registry.MarketOverrides))
	}
	expectedOverrides := map[identity.MarketOverride]bool{
		{Exchange: "okx", RawSymbol: "DRAM-USDT-SWAP", MarketType: "perpetual", CanonicalSymbol: "DRAM/USDT"}: true,
		{Exchange: "hyperliquid", RawSymbol: "DRAM", MarketType: "perpetual", CanonicalSymbol: "DRAM/USDT"}:   true,
	}
	for _, override := range registry.MarketOverrides {
		if !expectedOverrides[override] {
			t.Fatalf("unexpected generated override: %+v", override)
		}
	}
	if len(registry.AssetAliases) != 2 {
		t.Fatalf("expected DRAM and USDT asset aliases, got %d", len(registry.AssetAliases))
	}
	if registry.GeneratedVersion != curation.GeneratedRegistryVersion {
		t.Fatalf("expected generated registry version %d, got %d", curation.GeneratedRegistryVersion, registry.GeneratedVersion)
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

func TestInferAssetClass(t *testing.T) {
	cases := map[string]string{
		"USDT":   "fiat_stable",
		"AEUR":   "fiat_stable",
		"PYUSD":  "fiat_stable",
		"AAPL":   "rwa_stock",
		"AAPLON": "",
		"AMZNX":  "",
		"TSM":    "rwa_stock",
		"SOXL":   "rwa_stock",
		"XLE":    "rwa_stock",
		"CL":     "rwa_commodity",
		"NATGAS": "rwa_commodity",
		"XAUT":   "rwa_commodity",
		"CVX":    "",
		"BTC":    "",
	}

	for asset, expected := range cases {
		if actual := inferAssetClassFallback(asset); actual != expected {
			t.Fatalf("asset %s expected %s got %s", asset, expected, actual)
		}
	}
}

func TestInferAssetClassDoesNotGuessWrappedStocksFromSuffix(t *testing.T) {
	for _, asset := range []string{"DON", "HDX", "METAX", "SPYON"} {
		if actual := inferAssetClassFallback(asset); actual != "" {
			t.Fatalf("expected %s to require explicit classification, got %s", asset, actual)
		}
	}
}

func TestClassifyGeneratedAssetPrefersExchangeMetadata(t *testing.T) {
	item := discoveryItem{
		BaseAsset:      "CL",
		AssetClassHint: "commodity",
		Category:       "energy",
		Tags:           []string{"oil"},
	}

	if actual := classifyGeneratedAsset(item, "CL"); actual != "rwa_commodity" {
		t.Fatalf("expected exchange metadata to classify CL as rwa_commodity, got %s", actual)
	}
}

func TestClassifyGeneratedAssetTreatsTradFiEquityMetadataAsRWAStock(t *testing.T) {
	item := discoveryItem{
		BaseAsset:          "KORU",
		AssetClassHint:     "equity",
		Category:           "tradifi_perpetual",
		UnderlyingCategory: "equity",
	}

	if actual := classifyGeneratedAsset(item, "KORU"); actual != "rwa_stock" {
		t.Fatalf("expected KORU TradFi equity metadata to classify as rwa_stock, got %s", actual)
	}
}

func TestClassifyGeneratedAssetKeepsExplicitRWATickerAheadOfGenericCryptoMetadata(t *testing.T) {
	item := discoveryItem{
		BaseAsset:      "WMT",
		AssetClassHint: "crypto",
		Category:       "token",
		Tags:           []string{"coin"},
	}

	if actual := classifyGeneratedAsset(item, "WMT"); actual != "rwa_stock" {
		t.Fatalf("expected WMT to classify as rwa_stock despite generic crypto metadata, got %s", actual)
	}
}

func TestClassifyGeneratedAssetPrefersStockHintOverGenericTokenHint(t *testing.T) {
	item := discoveryItem{
		BaseAsset:          "NEWSTOCK",
		Category:           "token",
		UnderlyingCategory: "stock",
		Tags:               []string{"tokenized-stock"},
	}

	if actual := classifyGeneratedAsset(item, "NEWSTOCK"); actual != "rwa_stock" {
		t.Fatalf("expected stock hint to beat generic token hint, got %s", actual)
	}
}

func TestMergeGeneratedRegistryPromotesExistingCryptoWhenCurrentRWAHasEvidence(t *testing.T) {
	existing := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "KORU", AssetClass: "crypto"},
			{Canonical: "USDT", AssetClass: "fiat_stable"},
		},
	}
	current := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID:         "binance",
			VenueType:          "cex",
			MarketType:         "perp",
			Symbol:             "KORUUSDT",
			BaseAsset:          "KORU",
			QuoteAsset:         "USDT",
			AssetClassHint:     "equity",
			Category:           "tradifi_perpetual",
			UnderlyingCategory: "equity",
			Status:             "live",
		},
	})

	merged := sanitizeExistingGeneratedRegistry(existing, current).Merge(current)
	for _, asset := range merged.AssetAliases {
		if asset.Canonical == "KORU" {
			if asset.AssetClass != "rwa_stock" {
				t.Fatalf("expected current RWA evidence to correct KORU, got %+v", asset)
			}
			return
		}
	}
	t.Fatalf("expected KORU asset alias, got %+v", merged.AssetAliases)
}

func TestBuildGeneratedRegistrySkipsUnknownAutoAliases(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "binance",
			VenueType:  "cex",
			MarketType: "perpetual",
			Symbol:     "FOOUSDT",
			BaseAsset:  "FOO",
			QuoteAsset: "USDT",
			Status:     "live",
		},
	})

	if len(registry.MarketOverrides) != 1 {
		t.Fatalf("expected generated override for FOOUSDT, got %d", len(registry.MarketOverrides))
	}
	if len(registry.AssetAliases) != 1 {
		t.Fatalf("expected only stable quote alias to be generated, got %d", len(registry.AssetAliases))
	}
	if registry.AssetAliases[0].Canonical != "USDT" || registry.AssetAliases[0].AssetClass != "fiat_stable" {
		t.Fatalf("unexpected generated alias: %+v", registry.AssetAliases[0])
	}
}

func TestBuildGeneratedRegistryInfersHyperliquidHIP3RWAStockAlias(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "binance",
			VenueType:  "cex",
			MarketType: "perp",
			Symbol:     "SKHYNIXUSDT",
			BaseAsset:  "SKHYNIX",
			QuoteAsset: "USDT",
			Status:     "live",
		},
		{
			PlatformID: "hyperliquid",
			VenueType:  "dex",
			MarketType: "perp",
			Symbol:     "xyz:SKHX",
			BaseAsset:  "SKHX",
			QuoteAsset: "USDC",
			Status:     "live",
		},
	})

	var foundAlias bool
	for _, item := range registry.AssetAliases {
		if item.Canonical != "SKHYNIX" {
			continue
		}
		if item.AssetClass != "rwa_stock" {
			t.Fatalf("expected SKHYNIX to classify as rwa_stock, got %+v", item)
		}
		for _, alias := range item.Aliases {
			if alias == "SKHX" {
				foundAlias = true
			}
		}
	}
	if !foundAlias {
		t.Fatalf("expected SKHYNIX alias SKHX, got %+v", registry.AssetAliases)
	}

	for _, override := range registry.MarketOverrides {
		if override.Exchange == "hyperliquid" && override.RawSymbol == "xyz:SKHX" {
			if override.CanonicalSymbol != "SKHYNIX/USDT" {
				t.Fatalf("expected xyz:SKHX to canonicalize to SKHYNIX/USDT, got %+v", override)
			}
			return
		}
	}
	t.Fatalf("expected hyperliquid xyz:SKHX override, got %+v", registry.MarketOverrides)
}

func TestBuildGeneratedRegistryDoesNotClassifySuffixLookalikes(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "DON_USDT",
			BaseAsset:  "DON",
			QuoteAsset: "USDT",
			Status:     "live",
		},
	})

	if len(registry.MarketOverrides) != 1 {
		t.Fatalf("expected DON market override to be retained, got %d", len(registry.MarketOverrides))
	}
	for _, item := range registry.AssetAliases {
		if item.Canonical == "DON" {
			t.Fatalf("expected DON not to get a formal asset class alias, got %+v", item)
		}
	}
}

func TestBuildGeneratedRegistryCollapsesScaledUnitAssets(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "binance",
			VenueType:  "cex",
			MarketType: "perpetual",
			Symbol:     "1000PEPEUSDT",
			BaseAsset:  "1000PEPE",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
		{
			PlatformID: "okx",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "PEPE-USDT",
			BaseAsset:  "PEPE",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
	})

	assets := map[string]identity.AssetAliasRule{}
	for _, item := range registry.AssetAliases {
		assets[item.Canonical] = item
	}
	pepe, ok := assets["PEPE"]
	if !ok {
		t.Fatalf("expected PEPE asset alias to exist after collapse")
	}
	if len(pepe.UnitAliases) != 1 || pepe.UnitAliases[0].Alias != "1000PEPE" || pepe.UnitAliases[0].Multiplier != 1000 {
		t.Fatalf("expected 1000PEPE unit alias, got %+v", pepe.UnitAliases)
	}
	if _, exists := assets["1000PEPE"]; exists {
		t.Fatalf("expected scaled asset canonical to be collapsed away")
	}

	for _, item := range registry.MarketOverrides {
		if item.RawSymbol == "1000PEPEUSDT" && item.CanonicalSymbol != "PEPE/USDT" {
			t.Fatalf("expected scaled market override to rewrite to PEPE/USDT, got %+v", item)
		}
	}
}

func TestBuildGeneratedRegistrySkipsGateLeveragedTokens(t *testing.T) {
	registry := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC3L_USDT",
			BaseAsset:  "BTC3L",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
		{
			PlatformID: "gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "BTC_USDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			AssetClass: "crypto",
			Status:     "live",
		},
	})

	if len(registry.MarketOverrides) != 1 {
		t.Fatalf("expected leveraged gate token to be filtered, got %d overrides", len(registry.MarketOverrides))
	}
	if registry.MarketOverrides[0].RawSymbol != "BTC_USDT" {
		t.Fatalf("unexpected override retained: %+v", registry.MarketOverrides[0])
	}
}

func TestDecodeDiscoveryEnvelopeAcceptsServerSyncWrapper(t *testing.T) {
	envelope, err := decodeDiscoveryEnvelope([]byte(`{
		"source":{"id":"slipstream-prod"},
		"payload":{
			"source":"slipstream",
			"items":[
				{"platformId":"gate","venueType":"cex","marketType":"spot","symbol":"SPCX_USDT","baseAsset":"SPCX","quoteAsset":"USDT"}
			]
		}
	}`))
	if err != nil {
		t.Fatalf("decode wrapped envelope: %v", err)
	}
	if len(envelope.Items) != 1 || envelope.Items[0].Symbol != "SPCX_USDT" {
		t.Fatalf("unexpected wrapped envelope: %+v", envelope)
	}
}

func TestLoadDiscoveryPayloadRejectsMultipleSources(t *testing.T) {
	_, err := loadDiscoveryPayload(discoveryLoadOptions{
		InputPath:    "input.json",
		UseBootstrap: true,
	})
	if err == nil || !strings.Contains(err.Error(), "only one discovery source") {
		t.Fatalf("expected multiple source error, got %v", err)
	}
}

func TestLoadDiscoveryPayloadFetchesBootstrapCollectors(t *testing.T) {
	restore := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		payloads := map[string]string{
			"GET https://api.gateio.ws/api/v4/spot/currency_pairs":    `[{"id":"SPCX_USDT","base":"SPCX","quote":"USDT","trade_status":"tradable"}]`,
			"GET https://api.gateio.ws/api/v4/futures/usdt/contracts": `[]`,
		}
		key := req.Method + " " + req.URL.String()
		body, ok := payloads[key]
		if !ok {
			t.Fatalf("unexpected bootstrap request: %s", key)
		}
		return jsonResponse(body), nil
	})
	defer func() { http.DefaultTransport = restore }()

	payload, err := loadDiscoveryPayload(discoveryLoadOptions{
		UseBootstrap:     true,
		BootstrapSources: []string{"gate"},
	})
	if err != nil {
		t.Fatalf("load bootstrap discovery: %v", err)
	}

	envelope, err := decodeDiscoveryEnvelope(payload)
	if err != nil {
		t.Fatalf("decode bootstrap payload: %v", err)
	}
	if envelope.Source != "market-kit-bootstrap" {
		t.Fatalf("unexpected source: %s", envelope.Source)
	}
	if len(envelope.Items) != 1 || envelope.Items[0].Symbol != "SPCX_USDT" {
		t.Fatalf("unexpected bootstrap items: %+v", envelope.Items)
	}
}

func TestBuildReviewReportHighlightsRWANewAssets(t *testing.T) {
	items := []discoveryItem{
		{
			PlatformID: "gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "SPCX_USDT",
			BaseAsset:  "SPCX",
			QuoteAsset: "USDT",
			Status:     "live",
		},
	}
	next := buildGeneratedRegistry(items)
	report := buildReviewReport(identity.Registry{}, next, items, "test-source", 20)

	if !strings.Contains(report, "New RWA / Commodity Overrides") {
		t.Fatalf("expected RWA review section, got:\n%s", report)
	}
	if !strings.Contains(report, "`SPCX_USDT`") || !strings.Contains(report, "`SPCX/USDT`") {
		t.Fatalf("expected SPCX override in review report, got:\n%s", report)
	}
}

func TestSanitizeExistingGeneratedRegistryDropsUnsupportedRWAClassifications(t *testing.T) {
	existing := identity.Registry{
		AssetAliases: []identity.AssetAliasRule{
			{Canonical: "SPCX", AssetClass: "rwa_stock"},
			{Canonical: "DON", AssetClass: "rwa_stock"},
			{Canonical: "BTC", AssetClass: "crypto"},
		},
		MarketOverrides: []identity.MarketOverride{
			{Exchange: "gate", RawSymbol: "DON_USDT", MarketType: "spot", CanonicalSymbol: "DON/USDT"},
		},
	}
	current := buildGeneratedRegistry([]discoveryItem{
		{
			PlatformID: "gate",
			VenueType:  "cex",
			MarketType: "spot",
			Symbol:     "SPCX_USDT",
			BaseAsset:  "SPCX",
			QuoteAsset: "USDT",
			Status:     "live",
		},
	})

	sanitized := sanitizeExistingGeneratedRegistry(existing, current)
	assets := map[string]string{}
	for _, item := range sanitized.AssetAliases {
		assets[item.Canonical] = item.AssetClass
	}
	if assets["SPCX"] != "rwa_stock" {
		t.Fatalf("expected explicitly supported SPCX to remain, got %+v", assets)
	}
	if _, ok := assets["DON"]; ok {
		t.Fatalf("expected unsupported DON RWA alias to be removed, got %+v", assets)
	}
	if assets["BTC"] != "crypto" {
		t.Fatalf("expected crypto alias to remain, got %+v", assets)
	}
	if len(sanitized.MarketOverrides) != 1 {
		t.Fatalf("expected market overrides to be preserved, got %+v", sanitized.MarketOverrides)
	}
}
