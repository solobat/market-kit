package identity

import "strings"

func CanonicalAssetID(assetClass string, symbol string) string {
	assetClass = strings.ToLower(strings.TrimSpace(assetClass))
	symbol = strings.ToLower(strings.TrimSpace(symbol))
	if symbol == "" {
		return ""
	}
	prefix := "unknown"
	switch assetClass {
	case "crypto":
		prefix = "crypto"
	case "rwa_stock":
		prefix = "equity"
	case "rwa_commodity":
		prefix = "commodity"
	case "fiat_stable":
		prefix = "fiat"
	}
	return prefix + ":ticker:" + symbol
}

func DefaultComparisonKey(assetID string, quoteAsset string) string {
	assetID = strings.ToLower(strings.TrimSpace(assetID))
	quoteAsset = strings.ToUpper(strings.TrimSpace(quoteAsset))
	if assetID == "" || strings.HasPrefix(assetID, "unknown:") || quoteAsset == "" {
		return ""
	}
	return assetID + "/" + quoteAsset
}

func NormalizeComparisonStatus(value ComparisonStatus) ComparisonStatus {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case string(ComparisonEligible):
		return ComparisonEligible
	case string(ComparisonProhibited):
		return ComparisonProhibited
	default:
		return ComparisonAmbiguous
	}
}

func DefaultInstrumentKind(marketType MarketType, assetClass string) string {
	assetClass = strings.ToLower(strings.TrimSpace(assetClass))
	switch marketType {
	case MarketTypeSpot:
		return "spot"
	case MarketTypePerpetual:
		switch assetClass {
		case "rwa_stock":
			return "equity_perpetual"
		case "rwa_commodity":
			return "commodity_perpetual"
		default:
			return "perpetual"
		}
	case MarketTypeFuture:
		return "dated_future"
	default:
		return "unknown"
	}
}

// DefaultSettlementAsset preserves unknown collateral for contract types whose
// reference quote is not proof of settlement. Explicit venue evidence wins.
func DefaultSettlementAsset(contractType, quote string) string {
	switch strings.ToLower(strings.TrimSpace(contractType)) {
	case "multi_collateral_cfd", "dated_future_unverified", "linear_futures", "inverse_futures":
		return ""
	default:
		return quote
	}
}
