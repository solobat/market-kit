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

	binanceWeb3OndoStockURL = "https://www.binance.com/bapi/defi/v1/public/wallet-direct/buw/wallet/market/token/rwa/stock/detail/list/ai"
)

type collector struct {
	id      string
	label   string
	fetch   func(context.Context, *http.Client) ([]discovery.ImportedMarket, error)
	enabled bool
}

type binanceWeb3StockToken struct {
	Ticker             string `json:"ticker"`
	UnderlyingSymbol   string `json:"underlyingSymbol"`
	StockSymbol        string `json:"stockSymbol"`
	AssetSymbol        string `json:"assetSymbol"`
	Symbol             string `json:"symbol"`
	TokenSymbol        string `json:"tokenSymbol"`
	Name               string `json:"name"`
	TokenName          string `json:"tokenName"`
	QuoteAsset         string `json:"quoteAsset"`
	QuoteSymbol        string `json:"quoteSymbol"`
	ChainID            string `json:"chainId"`
	ChainName          string `json:"chainName"`
	ContractAddress    string `json:"contractAddress"`
	Contract           string `json:"contract"`
	Type               any    `json:"type"`
	Status             string `json:"status"`
	TradingStatus      string `json:"tradingStatus"`
	ExternalURL        string `json:"externalUrl"`
	URL                string `json:"url"`
	Provider           string `json:"provider"`
	UnderlyingCategory string `json:"underlyingCategory"`
	Category           string `json:"category"`
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
	var failed []string
	for _, source := range collectors() {
		if !source.enabled {
			continue
		}
		if len(selected) > 0 && !selected[source.id] {
			continue
		}
		fetched, err := source.fetch(ctx, client)
		if err != nil {
			if len(selected) > 0 {
				return discovery.ImportEnvelope{}, fmt.Errorf("%s: %w", source.id, err)
			}
			failed = append(failed, fmt.Sprintf("%s: %v", source.id, err))
			continue
		}
		items = append(items, fetched...)
	}
	if len(items) == 0 && len(failed) > 0 {
		return discovery.ImportEnvelope{}, fmt.Errorf("all bootstrap collectors failed: %s", strings.Join(failed, "; "))
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
		{id: "binance-web3", label: "Binance Web3", fetch: fetchBinanceWeb3OndoStocks, enabled: true},
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

func fetchBinanceWeb3OndoStocks(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	type response struct {
		Data json.RawMessage `json:"data"`
	}

	var payload response
	if err := fetchJSONWithHeaders(ctx, client, http.MethodGet, binanceWeb3OndoStockURL, nil, binanceWeb3Headers(), &payload); err != nil {
		return nil, err
	}

	tokens := decodeBinanceWeb3StockTokens(payload.Data)
	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(tokens))
	for _, item := range tokens {
		if !isOndoProviderType(item.Type) {
			continue
		}

		base := strings.ToUpper(strings.TrimSpace(firstNonEmpty(
			item.Ticker,
			item.UnderlyingSymbol,
			item.StockSymbol,
			item.AssetSymbol,
		)))
		symbol := strings.TrimSpace(firstNonEmpty(item.Symbol, item.TokenSymbol, base))
		if base == "" && symbol != "" {
			base = strings.ToUpper(symbol)
		}
		if base == "" || symbol == "" {
			continue
		}

		chain := strings.TrimSpace(firstNonEmpty(item.ChainName, item.ChainID))
		contract := strings.TrimSpace(firstNonEmpty(item.ContractAddress, item.Contract))
		tags := []string{"binance-web3", "ondo", "tokenized-stock"}
		if contract != "" {
			tags = append(tags, "contract:"+contract)
		}

		out = append(out, discovery.ImportedMarket{
			SourceID:           BuiltInSourceID,
			PlatformID:         "binance-web3",
			Platform:           "Binance Web3",
			VenueType:          "web3",
			MarketType:         "spot",
			Symbol:             symbol,
			BaseAsset:          base,
			QuoteAsset:         strings.ToUpper(strings.TrimSpace(firstNonEmpty(item.QuoteAsset, item.QuoteSymbol, "USD"))),
			AssetClassHint:     "stock",
			Category:           firstNonEmpty(item.Category, "tokenized_stock"),
			UnderlyingCategory: firstNonEmpty(item.UnderlyingCategory, "stock"),
			Tags:               tags,
			Chain:              chain,
			Status:             normalizeBinanceWeb3Status(firstNonEmpty(item.TradingStatus, item.Status)),
			ExternalURL:        firstNonEmpty(item.ExternalURL, item.URL, "https://www.binance.com/en/web3"),
			FirstSeenAt:        seenAt,
			LastSeenAt:         seenAt,
		})
	}
	return out, nil
}

func decodeBinanceWeb3StockTokens(raw json.RawMessage) []binanceWeb3StockToken {
	var direct []binanceWeb3StockToken
	if err := json.Unmarshal(raw, &direct); err == nil && len(direct) > 0 {
		return direct
	}

	var wrapped struct {
		List      []binanceWeb3StockToken `json:"list"`
		Items     []binanceWeb3StockToken `json:"items"`
		Rows      []binanceWeb3StockToken `json:"rows"`
		TokenList []binanceWeb3StockToken `json:"tokenList"`
		StockList []binanceWeb3StockToken `json:"stockList"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		out := make(
			[]binanceWeb3StockToken,
			0,
			len(wrapped.List)+len(wrapped.Items)+len(wrapped.Rows)+len(wrapped.TokenList)+len(wrapped.StockList),
		)
		out = append(out, wrapped.List...)
		out = append(out, wrapped.Items...)
		out = append(out, wrapped.Rows...)
		out = append(out, wrapped.TokenList...)
		out = append(out, wrapped.StockList...)
		if len(out) > 0 {
			return out
		}
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		var out []binanceWeb3StockToken
		collectBinanceWeb3StockTokens(decoded, &out)
		return out
	}

	return nil
}

func collectBinanceWeb3StockTokens(value any, out *[]binanceWeb3StockToken) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectBinanceWeb3StockTokens(item, out)
		}
	case map[string]any:
		if looksLikeBinanceWeb3StockToken(typed) {
			payload, err := json.Marshal(typed)
			if err == nil {
				var token binanceWeb3StockToken
				if json.Unmarshal(payload, &token) == nil {
					*out = append(*out, token)
					return
				}
			}
		}
		for _, item := range typed {
			collectBinanceWeb3StockTokens(item, out)
		}
	}
}

func looksLikeBinanceWeb3StockToken(item map[string]any) bool {
	for _, key := range []string{
		"ticker",
		"underlyingSymbol",
		"stockSymbol",
		"assetSymbol",
		"tokenSymbol",
		"contractAddress",
	} {
		if strings.TrimSpace(fmt.Sprint(item[key])) != "" && fmt.Sprint(item[key]) != "<nil>" {
			return true
		}
	}
	return false
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
	type dexEntry struct {
		Name string `json:"name"`
	}

	var payload response
	if err := fetchJSON(ctx, client, http.MethodPost, "https://api.hyperliquid.xyz/info", map[string]any{"type": "meta"}, &payload); err != nil {
		return nil, err
	}

	seenAt := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(payload.Universe))
	seen := map[string]bool{}
	for _, item := range payload.Universe {
		symbol := strings.TrimSpace(item.Name)
		if symbol == "" || seen[symbol] {
			continue
		}
		seen[symbol] = true
		out = append(out, discovery.ImportedMarket{
			SourceID:    BuiltInSourceID,
			PlatformID:  "hyperliquid",
			Platform:    "Hyperliquid",
			VenueType:   "dex",
			MarketType:  "perp",
			Symbol:      symbol,
			BaseAsset:   symbol,
			QuoteAsset:  "USDC",
			Chain:       "Hyperliquid L1",
			Status:      "live",
			ExternalURL: "https://app.hyperliquid.xyz/trade/" + symbol,
			FirstSeenAt: seenAt,
			LastSeenAt:  seenAt,
		})
	}

	var dexes []dexEntry
	if err := fetchJSON(ctx, client, http.MethodPost, "https://api.hyperliquid.xyz/info", map[string]any{"type": "perpDexs"}, &dexes); err != nil {
		return nil, err
	}
	for _, dex := range dexes {
		dexName := strings.TrimSpace(dex.Name)
		if dexName == "" {
			continue
		}
		var payload []json.RawMessage
		if err := fetchJSON(ctx, client, http.MethodPost, "https://api.hyperliquid.xyz/info", map[string]any{"type": "metaAndAssetCtxs", "dex": dexName}, &payload); err != nil {
			return nil, err
		}
		if len(payload) == 0 {
			continue
		}

		var meta response
		if err := json.Unmarshal(payload[0], &meta); err != nil {
			continue
		}
		for _, item := range meta.Universe {
			base := strings.TrimSpace(item.Name)
			if base == "" {
				continue
			}
			symbol := dexName + ":" + base
			if strings.Contains(base, ":") {
				symbol = base
			}
			if seen[symbol] {
				continue
			}
			seen[symbol] = true
			out = append(out, discovery.ImportedMarket{
				SourceID:    BuiltInSourceID,
				PlatformID:  "hyperliquid",
				Platform:    "Hyperliquid",
				VenueType:   "dex",
				MarketType:  "perp",
				Symbol:      symbol,
				BaseAsset:   base,
				QuoteAsset:  "USDC",
				Chain:       "Hyperliquid L1",
				Status:      "live",
				ExternalURL: "https://app.hyperliquid.xyz/trade/" + symbol,
				FirstSeenAt: seenAt,
				LastSeenAt:  seenAt,
			})
		}
	}
	return out, nil
}

func fetchJSON[T any](ctx context.Context, client *http.Client, method, url string, body any, target *T) error {
	return fetchJSONWithHeaders(ctx, client, method, url, body, nil, target)
}

func fetchJSONWithHeaders[T any](ctx context.Context, client *http.Client, method, url string, body any, headers map[string]string, target *T) error {
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
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "market-kit/0.1")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
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

func binanceWeb3Headers() map[string]string {
	return map[string]string{
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "en-US,en;q=0.9",
		"Origin":          "https://www.binance.com",
		"Referer":         "https://www.binance.com/en/web3",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	}
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

func normalizeBinanceWeb3Status(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "trading", "live", "listed", "online", "enabled", "active":
		return "live"
	case "prelaunch", "pre_launch", "pre trading", "pre_trading", "pending":
		return "prelaunch"
	case "paused", "halt", "halted", "disabled", "offline", "delisted":
		return "paused"
	default:
		return "unknown"
	}
}

func isOndoProviderType(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case float64:
		return typed == 1
	case int:
		return typed == 1
	case string:
		typed = strings.ToLower(strings.TrimSpace(typed))
		return typed == "" || typed == "1" || typed == "ondo" || strings.Contains(typed, "ondo")
	default:
		return false
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
