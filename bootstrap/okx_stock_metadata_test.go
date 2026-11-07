package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/curation"
	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
	"net/http"
	"testing"
)

func TestOKXNativeStockContractUnitsStayVenueScoped(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(`{"data":[{"instId":"XLV-USDT-SWAP","instCategory":"3","ctType":"linear","ctVal":"0.1","ctMult":"1","ctValCcy":"XLV","settleCcy":"USDT","state":"live"},{"instId":"XLV-USD-SWAP","instCategory":"3","ctType":"inverse","ctVal":"100","ctMult":"1","ctValCcy":"USD","settleCcy":"XLV","state":"live"},{"instId":"BTC-USDT-SWAP","instCategory":"1","ctType":"linear","settleCcy":"USDT","state":"live"},{"instId":"UNKNOWN-USDT-SWAP","instCategory":"3","ctType":"linear","ctVal":"1","state":"live"}]}`), nil
	})}
	rows, err := fetchOKXType(context.Background(), client, "SWAP", "perp")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || rows[0].InstrumentMetadata.ContractValue != "0.1" || rows[0].InstrumentMetadata.ContractValueCurrency != "XLV" || rows[2].InstrumentMetadata != nil {
		t.Fatal(rows)
	}
	generic := rows[0]
	generic.SourceID = "consumer"
	generic.InstrumentMetadata = nil
	generic.AssetClassHint = "crypto"
	for _, input := range [][]discovery.ImportedMarket{append(append([]discovery.ImportedMarket{}, rows...), generic), append([]discovery.ImportedMarket{generic}, rows...)} {
		resolver := identity.NewResolver(curation.BuildGeneratedRegistry(input))
		for _, sym := range []string{"XLV-USDT-SWAP", "XLV-USD-SWAP", "UNKNOWN-USDT-SWAP"} {
			got := resolver.Resolve(identity.ResolveRequest{Exchange: "okx", Symbol: sym, MarketTypeHint: "perpetual"}).Market
			if got == nil || got.AssetClass != "rwa_stock" || got.ComparisonStatus != identity.ComparisonAmbiguous || got.AssetID == "equity:ticker:xlv" {
				t.Fatal(got)
			}
			if sym == "XLV-USD-SWAP" && (got.ContractType != "inverse_swap" || got.SettlementAsset != "XLV") {
				t.Fatal(got)
			}
			if sym == "XLV-USDT-SWAP" && (got.ContractType != "linear_swap" || got.UnitMultiplier == 0.1) {
				t.Fatal("contract count was treated as price alias", got)
			}
			if sym == "UNKNOWN-USDT-SWAP" && got.SettlementAsset != "" {
				t.Fatal("missing collateral inferred from quote", got)
			}
		}
	}
}

func TestConflictingOKXContractUnitsRemainUnverified(t *testing.T) {
	a := discovery.ImportedMarket{SourceID: "a", PlatformID: "okx", VenueType: "cex", MarketType: "perp", Symbol: "XLV-USDT-SWAP", BaseAsset: "XLV", QuoteAsset: "USDT", AssetClassHint: "stock", InstrumentMetadata: &discovery.InstrumentMetadata{Source: "okx", SymbolType: "3", ContractType: "linear_swap", SettlementAsset: "USDT", ContractValue: "1", ContractMultiplier: "1", ContractValueCurrency: "XLV"}}
	b := a
	m := *a.InstrumentMetadata
	b.InstrumentMetadata = &m
	b.InstrumentMetadata.ContractValue = "0.1"
	b.SourceID = "b"
	for _, rows := range [][]discovery.ImportedMarket{{a, b}, {b, a}} {
		got := identity.NewResolver(curation.BuildGeneratedRegistry(rows)).Resolve(identity.ResolveRequest{Exchange: "okx", Symbol: a.Symbol, MarketTypeHint: "perpetual"}).Market
		if got == nil || got.AssetClass != "unknown" || got.ComparisonStatus != identity.ComparisonAmbiguous || got.SettlementAsset != "" || got.ContractType != "stock_swap_unverified" {
			t.Fatal(got)
		}
	}
}
