package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/solobat/market-kit/discovery"
)

const arcusMarketsURL = "https://api.arcus.xyz/v1/markets"
const qfexRefdataURL = "https://api.qfex.com/refdata"
const qfexContractsURL = "https://api.qfex.com/md/contracts"

// Metadata discovery runs at the existing bootstrap cadence; prices are not identity evidence.
func fetchArcus(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	var response struct {
		Markets []struct {
			Symbol   string `json:"marketDisplayName"`
			Base     string `json:"baseAsset"`
			Quote    string `json:"quoteAsset"`
			Type     string `json:"type"`
			Status   string `json:"status"`
			Category string `json:"category"`
		} `json:"markets"`
	}
	if err := fetchJSON(ctx, client, http.MethodGet, arcusMarketsURL, nil, &response); err != nil {
		return nil, err
	}
	if response.Markets == nil {
		return nil, fmt.Errorf("Arcus missing markets")
	}
	rows := make([]discovery.ImportedMarket, 0, len(response.Markets))
	now := time.Now().UTC()
	for _, m := range response.Markets {
		if m.Type != "PERPETUAL" || m.Base == "" || m.Quote == "" || m.Symbol == "" {
			continue
		}
		hint := ""
		switch m.Category {
		case "CRYPTO":
			hint = "crypto"
		case "EQUITIES":
			hint = "stock"
		case "INDICES":
			hint = "index"
		}
		// Arcus commodity listings are fund shares, not spot commodities.
		switch m.Base {
		case "GLD", "SLV", "USO", "CPER", "SPY", "QQQ", "VT", "SGOV", "DRAM":
			hint = "etf"
		}
		status := "paused"
		if m.Status == "ONLINE" {
			status = "live"
		}
		rows = append(rows, discovery.ImportedMarket{SourceID: BuiltInSourceID, PlatformID: "arcus", Platform: "Arcus", VenueType: "dex", MarketType: "perp", Symbol: m.Symbol, BaseAsset: m.Base, QuoteAsset: m.Quote, AssetClassHint: hint, Category: m.Category, Chain: "robinhood", Status: status, ExternalURL: "https://app.arcus.xyz/trade/perpetuals/" + url.PathEscape(m.Symbol), FirstSeenAt: now, LastSeenAt: now})
	}
	return rows, nil
}

func fetchQfex(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	type reference struct {
		Symbol   string `json:"symbol"`
		Status   string `json:"status"`
		Category string `json:"product_category"`
	}
	var refs struct {
		Data []reference `json:"data"`
	}
	if err := fetchJSON(ctx, client, http.MethodGet, qfexRefdataURL, nil, &refs); err != nil {
		return nil, err
	}
	if refs.Data == nil {
		return nil, fmt.Errorf("QFEX missing refdata")
	}
	var contracts struct {
		Data []struct {
			Symbol string `json:"ticker_id"`
			Base   string `json:"base_currency"`
			Quote  string `json:"target_currency"`
			Type   string `json:"product_type"`
		} `json:"data"`
	}
	if err := fetchJSON(ctx, client, http.MethodGet, qfexContractsURL, nil, &contracts); err != nil {
		return nil, err
	}
	if contracts.Data == nil {
		return nil, fmt.Errorf("QFEX missing contracts")
	}
	metadata := make(map[string]reference, len(refs.Data))
	for _, r := range refs.Data {
		metadata[r.Symbol] = r
	}
	rows := make([]discovery.ImportedMarket, 0, len(contracts.Data))
	now := time.Now().UTC()
	for _, m := range contracts.Data {
		if m.Type != "Perpetual" || m.Symbol == "" || m.Base == "" || m.Quote == "" {
			continue
		}
		ref, ok := metadata[m.Symbol]
		if !ok {
			return nil, fmt.Errorf("QFEX missing metadata for %s", m.Symbol)
		}
		status := "paused"
		if ref.Status == "ACTIVE" {
			status = "live"
		}
		if ref.Status == "DELISTED" {
			status = "delisted"
		}
		hint := ""
		switch ref.Category {
		case "EQUITY":
			hint = "stock"
		case "INDEX":
			hint = "index"
		case "COMMODITY":
			hint = "commodity"
		}
		rows = append(rows, discovery.ImportedMarket{SourceID: BuiltInSourceID, PlatformID: "qfex", Platform: "QFEX", VenueType: "cex", MarketType: "perp", Symbol: m.Symbol, BaseAsset: strings.ToUpper(m.Base), QuoteAsset: strings.ToUpper(m.Quote), AssetClassHint: hint, Category: ref.Category, Status: status, ExternalURL: "https://www.qfex.com/trade", FirstSeenAt: now, LastSeenAt: now})
	}
	return rows, nil
}
