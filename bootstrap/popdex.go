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

const popdexSymbolsURL = "https://api.popdex.xyz/api/v1/config/symbols?category=Futures"

func fetchPopdex(ctx context.Context, client *http.Client) ([]discovery.ImportedMarket, error) {
	var response struct {
		Code string `json:"code"`
		Data []struct {
			Symbol     string `json:"symbol"`
			Base       string `json:"baseToken"`
			Quote      string `json:"quoteToken"`
			Category   string `json:"category"`
			Status     string `json:"status"`
			AssetClass string `json:"assetClass"`
		} `json:"data"`
	}
	if err := fetchJSON(ctx, client, http.MethodGet, popdexSymbolsURL, nil, &response); err != nil {
		return nil, err
	}
	if response.Code != "200" || response.Data == nil {
		return nil, fmt.Errorf("Popdex invalid symbol metadata (%s)", response.Code)
	}
	rows := make([]discovery.ImportedMarket, 0, len(response.Data))
	now := time.Now().UTC()
	for _, m := range response.Data {
		if !strings.EqualFold(m.Category, "Futures") || m.Symbol == "" || m.Base == "" || m.Quote == "" {
			continue
		}
		hint := ""
		if m.AssetClass == "Crypto" {
			hint = "crypto"
		}
		// Rwa includes stocks, ETFs, commodities and indices. Do not infer all RWA as stocks.
		if m.AssetClass == "Rwa" {
			switch m.Base {
			case "BZ", "CL", "XAU", "XAG":
				hint = "commodity"
			case "US500", "TECH100", "JP225", "KR200":
				hint = "index"
			case "SOXL", "DRAM", "EWY", "QQQ":
				hint = "etf"
			case "SPCX", "SNDK", "MU", "NVDA", "INTC", "SKHY", "CRCL", "META", "MSTR", "ORCL", "SAMSUNG", "SKHYNIX", "SOFTBANK", "CXMT", "UNITREE", "ZHIPU", "MINIMAX", "AAOI", "TSLA", "GOOGL", "SMCI", "HOOD", "COIN":
				hint = "stock"
			default:
				return nil, fmt.Errorf("Popdex unclassified RWA market %s", m.Symbol)
			}
		}
		status := "paused"
		if m.Status == "Trading" {
			status = "live"
		} else if m.Status == "Delisted" {
			status = "delisted"
		}
		rows = append(rows, discovery.ImportedMarket{SourceID: BuiltInSourceID, PlatformID: "popdex", Platform: "Popdex", VenueType: "dex", MarketType: "perp", Symbol: m.Symbol, BaseAsset: m.Base, QuoteAsset: m.Quote, AssetClassHint: hint, Category: m.AssetClass, Chain: "morph-tachyon", Status: status, ExternalURL: "https://app.popdex.xyz/en/trade/futures/" + url.PathEscape(m.Symbol), FirstSeenAt: now, LastSeenAt: now})
	}
	return rows, nil
}
