package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/solobat/market-kit/discovery"
)

const (
	BuiltInSourceID    = "market-kit-bootstrap"
	BuiltInSourceLabel = "交易所直连启动数据"
	BuiltInSourceURL   = "builtin:bootstrap"
)

type collector struct {
	id      string
	label   string
	fetch   func(context.Context, *http.Client) ([]discovery.ImportedMarket, error)
	enabled bool
}

func FetchDefault(ctx context.Context, client *http.Client) (discovery.ImportEnvelope, error) {
	return Fetch(ctx, client, nil)
}

func Fetch(ctx context.Context, client *http.Client, sourceIDs []string) (discovery.ImportEnvelope, error) {
	selected := map[string]bool{}
	for _, sourceID := range sourceIDs {
		sourceID = strings.ToLower(strings.TrimSpace(sourceID))
		if sourceID != "" {
			selected[sourceID] = true
		}
	}

	items := make([]discovery.ImportedMarket, 0, 2048)
	for _, source := range collectors() {
		if !source.enabled {
			continue
		}
		if len(selected) > 0 && !selected[source.id] {
			continue
		}
		fetched, err := source.fetch(ctx, client)
		if err != nil {
			return discovery.ImportEnvelope{}, fmt.Errorf("%s: %w", source.id, err)
		}
		items = append(items, fetched...)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].PlatformID == items[j].PlatformID {
			if items[i].MarketType == items[j].MarketType {
				return items[i].Symbol < items[j].Symbol
			}
			return items[i].MarketType < items[j].MarketType
		}
		return items[i].PlatformID < items[j].PlatformID
	})

	return discovery.ImportEnvelope{
		Source:      discovery.SourceKindBootstrap,
		GeneratedAt: time.Now().UTC(),
		Items:       items,
	}, nil
}

func collectors() []collector {
	return []collector{
		{id: "binance", label: "Binance", fetch: fetchBinance, enabled: true},
		{id: "bybit", label: "Bybit", fetch: fetchBybit, enabled: true},
		{id: "okx", label: "OKX", fetch: fetchOKX, enabled: true},
		{id: "bitget", label: "Bitget", fetch: fetchBitget, enabled: true},
		{id: "gate", label: "Gate", fetch: fetchGate, enabled: true},
		{id: "hyperliquid", label: "Hyperliquid", fetch: fetchHyperliquid, enabled: true},
	}
}

func fetchBinance(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	type symbol struct {
		Symbol     string `json:"symbol"`
		Status     string `json:"status"`
		BaseAsset  string `json:"baseAsset"`
		QuoteAsset string `json:"quoteAsset"`
	}
	type response struct {
		Symbols []symbol `json:"symbols"`
	}

	var spot response
	if err := fetchJSON(ctx, client, http.MethodGet, "https://api.binance.com/api/v3/exchangeInfo", nil, &spot); err != nil {
		return nil, err
	}

	var perp response
	if err := fetchJSON(ctx, client, http.MethodGet, "https://fapi.binance.com/fapi/v1/exchangeInfo", nil, &perp); err != nil {
		return nil, err
	}

	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(spot.Symbols)+len(perp.Symbols))
	for _, item := range spot.Symbols {
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "binance",
			Platform:    "Binance",
			VenueType:   "cex",
			MarketType:  "spot",
			Symbol:      strings.TrimSpace(item.Symbol),
			BaseAsset:   strings.TrimSpace(item.BaseAsset),
			QuoteAsset:  strings.TrimSpace(item.QuoteAsset),
			Status:      normalizeBinanceStatus(item.Status),
			ExternalURL: "https://www.binance.com/en/trade/" + strings.ToUpper(item.BaseAsset) + "_" + strings.ToUpper(item.QuoteAsset),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}
	for _, item := range perp.Symbols {
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "binance",
			Platform:    "Binance",
			VenueType:   "cex",
			MarketType:  "perp",
			Symbol:      strings.TrimSpace(item.Symbol),
			BaseAsset:   strings.TrimSpace(item.BaseAsset),
			QuoteAsset:  strings.TrimSpace(item.QuoteAsset),
			Status:      normalizeBinanceStatus(item.Status),
			ExternalURL: "https://www.binance.com/en/futures/" + strings.ToUpper(item.Symbol),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}
	return out, nil
}

func fetchBybit(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	spot, err := fetchBybitCategory(ctx, client, "spot", "spot")
	if err != nil {
		return nil, err
	}
	perp, err := fetchBybitCategory(ctx, client, "linear", "perp")
	if err != nil {
		return nil, err
	}
	return append(spot, perp...), nil
}

func fetchBybitCategory(ctx context.Context, client *http.Client, category, marketType string) ([]discovery.ImportedMarket, error) {
	type item struct {
		Symbol    string `json:"symbol"`
		Status    string `json:"status"`
		BaseCoin  string `json:"baseCoin"`
		QuoteCoin string `json:"quoteCoin"`
	}
	type response struct {
		Result struct {
			List []item `json:"list"`
		} `json:"result"`
	}

	url := fmt.Sprintf("https://api.bybit.com/v5/market/instruments-info?category=%s&limit=1000", category)
	var payload response
	if err := fetchJSON(ctx, client, http.MethodGet, url, nil, &payload); err != nil {
		return nil, err
	}

	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(payload.Result.List))
	for _, item := range payload.Result.List {
		base := strings.TrimSpace(item.BaseCoin)
		quote := strings.TrimSpace(item.QuoteCoin)
		if base == "" || quote == "" {
			base, quote = splitSymbol(item.Symbol)
		}
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "bybit",
			Platform:    "Bybit",
			VenueType:   "cex",
			MarketType:  marketType,
			Symbol:      strings.TrimSpace(item.Symbol),
			BaseAsset:   base,
			QuoteAsset:  quote,
			Category:    category,
			Status:      normalizeBybitStatus(item.Status),
			ExternalURL: "https://www.bybit.com/trade/usdt/" + strings.ToUpper(item.Symbol),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}
	return out, nil
}

func fetchOKX(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	spot, err := fetchOKXType(ctx, client, "SPOT", "spot")
	if err != nil {
		return nil, err
	}
	perp, err := fetchOKXType(ctx, client, "SWAP", "perp")
	if err != nil {
		return nil, err
	}
	return append(spot, perp...), nil
}

func fetchOKXType(ctx context.Context, client *http.Client, instType, marketType string) ([]discovery.ImportedMarket, error) {
	type item struct {
		InstID   string `json:"instId"`
		BaseCcy  string `json:"baseCcy"`
		QuoteCcy string `json:"quoteCcy"`
		State    string `json:"state"`
	}
	type response struct {
		Data []item `json:"data"`
	}

	url := fmt.Sprintf("https://www.okx.com/api/v5/public/instruments?instType=%s", instType)
	var payload response
	if err := fetchJSON(ctx, client, http.MethodGet, url, nil, &payload); err != nil {
		return nil, err
	}

	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(payload.Data))
	for _, item := range payload.Data {
		base := strings.TrimSpace(item.BaseCcy)
		quote := strings.TrimSpace(item.QuoteCcy)
		if base == "" || quote == "" {
			base, quote = splitDelimitedSymbol(item.InstID)
		}
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "okx",
			Platform:    "OKX",
			VenueType:   "cex",
			MarketType:  marketType,
			Symbol:      strings.TrimSpace(item.InstID),
			BaseAsset:   base,
			QuoteAsset:  quote,
			Category:    strings.ToLower(instType),
			Status:      normalizeSimpleLiveState(item.State),
			ExternalURL: "https://www.okx.com/trade-swap/" + strings.ToLower(item.InstID),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}
	return out, nil
}

func fetchBitget(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	type spotSymbol struct {
		Symbol       string `json:"symbol"`
		BaseCoin     string `json:"baseCoin"`
		QuoteCoin    string `json:"quoteCoin"`
		SymbolStatus string `json:"symbolStatus"`
	}
	type spotResponse struct {
		Data []spotSymbol `json:"data"`
	}

	var spot spotResponse
	if err := fetchJSON(ctx, client, http.MethodGet, "https://api.bitget.com/api/v2/spot/public/symbols", nil, &spot); err != nil {
		return nil, err
	}

	type contract struct {
		Symbol       string `json:"symbol"`
		BaseCoin     string `json:"baseCoin"`
		QuoteCoin    string `json:"quoteCoin"`
		SymbolStatus string `json:"symbolStatus"`
	}
	type contractResponse struct {
		Data []contract `json:"data"`
	}

	var perp contractResponse
	if err := fetchJSON(ctx, client, http.MethodGet, "https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES", nil, &perp); err != nil {
		return nil, err
	}

	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(spot.Data)+len(perp.Data))
	for _, item := range spot.Data {
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "bitget",
			Platform:    "Bitget",
			VenueType:   "cex",
			MarketType:  "spot",
			Symbol:      strings.TrimSpace(item.Symbol),
			BaseAsset:   strings.TrimSpace(item.BaseCoin),
			QuoteAsset:  strings.TrimSpace(item.QuoteCoin),
			Status:      normalizeSimpleLiveState(item.SymbolStatus),
			ExternalURL: "https://www.bitget.com/spot/" + strings.ToUpper(item.Symbol),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}
	for _, item := range perp.Data {
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "bitget",
			Platform:    "Bitget",
			VenueType:   "cex",
			MarketType:  "perp",
			Symbol:      strings.TrimSpace(item.Symbol),
			BaseAsset:   strings.TrimSpace(item.BaseCoin),
			QuoteAsset:  strings.TrimSpace(item.QuoteCoin),
			Category:    "usdt-futures",
			Status:      normalizeSimpleLiveState(item.SymbolStatus),
			ExternalURL: "https://www.bitget.com/futures/usdt/" + strings.ToUpper(item.Symbol),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}
	return out, nil
}

func fetchGate(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	type pair struct {
		ID          string `json:"id"`
		Base        string `json:"base"`
		Quote       string `json:"quote"`
		TradeStatus string `json:"trade_status"`
	}
	var spot []pair
	if err := fetchJSON(ctx, client, http.MethodGet, "https://api.gateio.ws/api/v4/spot/currency_pairs", nil, &spot); err != nil {
		return nil, err
	}

	type contract struct {
		Name        string `json:"name"`
		QuantoBase  string `json:"quanto_base"`
		Settle      string `json:"settle"`
		InDelisting bool   `json:"in_delisting"`
	}
	var perp []contract
	if err := fetchJSON(ctx, client, http.MethodGet, "https://api.gateio.ws/api/v4/futures/usdt/contracts", nil, &perp); err != nil {
		return nil, err
	}

	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(spot)+len(perp))
	for _, item := range spot {
		market := discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "gate",
			Platform:    "Gate",
			VenueType:   "cex",
			MarketType:  "spot",
			Symbol:      strings.TrimSpace(item.ID),
			BaseAsset:   strings.TrimSpace(item.Base),
			QuoteAsset:  strings.TrimSpace(item.Quote),
			Status:      normalizeGateTradeStatus(item.TradeStatus),
			ExternalURL: "https://www.gate.com/trade/" + strings.ToUpper(item.ID),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		}
		if discovery.ShouldIgnoreImportedMarket(market) {
			continue
		}
		out = append(out, market)
	}
	for _, item := range perp {
		status := "live"
		if item.InDelisting {
			status = "paused"
		}
		market := discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "gate",
			Platform:    "Gate",
			VenueType:   "cex",
			MarketType:  "perp",
			Symbol:      strings.TrimSpace(item.Name),
			BaseAsset:   strings.TrimSpace(item.QuantoBase),
			QuoteAsset:  strings.TrimSpace(item.Settle),
			Category:    "usdt-futures",
			Status:      status,
			ExternalURL: "https://www.gate.com/futures/" + strings.ToUpper(item.Name),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		}
		if discovery.ShouldIgnoreImportedMarket(market) {
			continue
		}
		out = append(out, market)
	}
	return out, nil
}

func fetchHyperliquid(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	type market struct {
		Name string `json:"name"`
	}
	type response struct {
		Universe []market `json:"universe"`
	}

	var payload response
	if err := fetchJSON(ctx, client, http.MethodPost, "https://api.hyperliquid.xyz/info", map[string]any{"type": "meta"}, &payload); err != nil {
		return nil, err
	}

	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(payload.Universe))
	for _, item := range payload.Universe {
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "hyperliquid",
			Platform:    "Hyperliquid",
			VenueType:   "dex",
			MarketType:  "perp",
			Symbol:      strings.TrimSpace(item.Name),
			BaseAsset:   strings.TrimSpace(item.Name),
			QuoteAsset:  "USDC",
			Chain:       "Hyperliquid L1",
			Status:      "live",
			ExternalURL: "https://app.hyperliquid.xyz/trade/" + strings.TrimSpace(item.Name),
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}
	return out, nil
}

func fetchJSON[T any](ctx context.Context, client *http.Client, method, url string, body any, target *T) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		content, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("%s %s failed: %s - %s", method, url, resp.Status, strings.TrimSpace(string(content)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func splitSymbol(symbol string) (string, string) {
	symbol = strings.TrimSpace(symbol)
	for _, quote := range []string{"USDT", "USDC", "USD", "BTC", "ETH"} {
		if strings.HasSuffix(symbol, quote) && len(symbol) > len(quote) {
			return strings.TrimSuffix(symbol, quote), quote
		}
	}
	return symbol, ""
}

func splitDelimitedSymbol(symbol string) (string, string) {
	parts := strings.Split(strings.TrimSpace(symbol), "-")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return symbol, ""
}

func normalizeBinanceStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "TRADING":
		return "live"
	case "PRE_TRADING", "PENDING_TRADING":
		return "prelaunch"
	case "BREAK", "HALT":
		return "paused"
	default:
		return "unknown"
	}
}

func normalizeBybitStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "trading":
		return "live"
	case "prelaunch", "settling":
		return "prelaunch"
	case "closed", "suspend":
		return "paused"
	default:
		return "unknown"
	}
}

func normalizeSimpleLiveState(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "live", "online", "listed", "normal":
		return "live"
	case "prelaunch", "pre_market":
		return "prelaunch"
	case "offline", "suspend", "paused":
		return "paused"
	default:
		return "unknown"
	}
}

func normalizeGateTradeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "tradable", "buyable", "sellable":
		return "live"
	case "untradable":
		return "paused"
	default:
		return "unknown"
	}
}
