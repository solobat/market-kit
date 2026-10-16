package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/solobat/market-kit/discovery"
)

const extendedReferenceURL = "https://api.starknet.extended.exchange/api/v1/info/markets"
const gtradeReferenceURL = "https://backend-arbitrum.gains.trade/trading-variables"

// Official metadata closes the classification gaps in generic downstream feeds.
// Preserve the venue's actual contract name, including 24/5 and ADR distinctions.
func fetchExtendedReference(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	var response struct {
		Status string `json:"status"`
		Data   []struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Base        string `json:"assetName"`
			Quote       string `json:"collateralAssetName"`
			Category    string `json:"category"`
			SubCategory string `json:"subCategory"`
			Active      bool   `json:"active"`
		} `json:"data"`
	}
	if err := fetchJSON(ctx, client, http.MethodGet, extendedReferenceURL, nil, &response); err != nil {
		return nil, err
	}
	if response.Status != "OK" || response.Data == nil {
		return nil, fmt.Errorf("Extended invalid market metadata (%s)", response.Status)
	}
	now := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(response.Data))
	for _, v := range response.Data {
		if v.Type != "PERPETUAL" || v.Name == "" || v.Base == "" || v.Quote == "" {
			continue
		}
		hint := ""
		underlying := ""
		switch v.SubCategory {
		case "Commodity":
			hint = "commodity"
		case "Equity":
			hint = "stock"
		case "ETF/Index":
			hint = "index"
		}
		if hint == "" && v.Category == "Crypto" {
			hint = "crypto"
			underlying = "coin"
		}
		// Forex and pre-market products cannot be inferred to be crypto from RWA.
		status := "paused"
		if v.Active {
			status = "live"
		}
		out = append(out, discovery.ImportedMarket{SourceID: BuiltInSourceID, PlatformID: "extended", Platform: "Extended", VenueType: "dex", MarketType: "perp", Symbol: v.Name, BaseAsset: v.Base, QuoteAsset: v.Quote, AssetClassHint: hint, Category: v.SubCategory, UnderlyingCategory: underlying, Status: status, FirstSeenAt: now, LastSeenAt: now, ExternalURL: "https://app.extended.exchange"})
	}
	return out, nil
}

// gTrade's official catalog includes non-crypto groups omitted by older exports.
// Catalog membership alone does not prove that an individual market is tradable.
func fetchGTradeReference(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	var response struct {
		Pairs []struct {
			From       string `json:"from"`
			To         string `json:"to"`
			GroupIndex string `json:"groupIndex"`
		} `json:"pairs"`
		Groups []struct {
			Name string `json:"name"`
		} `json:"groups"`
	}
	if err := fetchJSON(ctx, client, http.MethodGet, gtradeReferenceURL, nil, &response); err != nil {
		return nil, err
	}
	if response.Pairs == nil || response.Groups == nil {
		return nil, fmt.Errorf("gTrade missing market groups")
	}
	now := time.Now().UTC()
	out := make([]discovery.ImportedMarket, 0, len(response.Pairs))
	for _, v := range response.Pairs {
		idx, err := strconv.Atoi(v.GroupIndex)
		if err != nil || idx < 0 || idx >= len(response.Groups) || v.From == "" || v.To == "" {
			continue
		}
		group := strings.ToLower(response.Groups[idx].Name)
		hint := ""
		underlying := ""
		switch {
		case strings.HasPrefix(group, "stocks"):
			hint = "stock"
		case group == "indices":
			hint = "index"
		case strings.HasPrefix(group, "commodities"):
			hint = "commodity"
		case strings.Contains(group, "crypto") || group == "altcoins":
			hint = "crypto"
			underlying = "coin"
		}
		out = append(out, discovery.ImportedMarket{SourceID: BuiltInSourceID, PlatformID: "gtrade", Platform: "gTrade", VenueType: "dex", MarketType: "perp", Symbol: strings.ToUpper(v.From + "/" + v.To), BaseAsset: strings.ToUpper(v.From), QuoteAsset: strings.ToUpper(v.To), AssetClassHint: hint, Category: group, UnderlyingCategory: underlying, Status: "unknown", FirstSeenAt: now, LastSeenAt: now, ExternalURL: "https://gains.trade/trade"})
	}
	return out, nil
}
