package discovery

import (
	"fmt"
	"strings"

	"github.com/solobat/market-kit/identity"
)

// SecurityDescriptionKey normalizes native security identifiers only within an
// explicit market. It is never used to create asset IDs or comparison keys.
func SecurityDescriptionKey(region, ticker string) string {
	region = strings.ToUpper(strings.TrimSpace(region))
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	width := 0
	switch region {
	case "HK":
		width = 5
		ticker = strings.TrimPrefix(ticker, "HK")
		ticker = strings.TrimSuffix(ticker, "HK")
	case "CN":
		width = 6
		ticker = strings.TrimPrefix(ticker, "CN")
	case "KR":
		width = 6
		ticker = strings.TrimPrefix(ticker, "KR")
	case "US":
		if ticker == "" {
			return ""
		}
		for _, c := range ticker {
			if !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '.' && c != '-' {
				return ""
			}
		}
		return region + ":" + ticker
	default:
		return ""
	}
	if ticker == "" || len(ticker) > width {
		return ""
	}
	nonzero := false
	for _, c := range ticker {
		if c < '0' || c > '9' {
			return ""
		}
		if c != '0' {
			nonzero = true
		}
	}
	if !nonzero {
		return ""
	}
	return region + ":" + strings.Repeat("0", width-len(ticker)) + ticker
}

func descriptorSignature(m *InstrumentMetadata) string {
	return fmt.Sprintf("%q/%q/%q/%q", strings.TrimSpace(m.FullName), strings.ToUpper(m.MarketRegion), SecurityDescriptionKey(m.MarketRegion, m.UnderlyingTicker), strings.ToLower(m.SymbolType))
}

func enrichInstrumentDescriptions(rows []CandidateMarket) {
	byIdentity := map[string][]*InstrumentMetadata{}
	bySecurity := map[string][]*InstrumentMetadata{}
	for i := range rows {
		row := &rows[i]
		m := row.InstrumentMetadata
		if row.ComparisonStatus != identity.ComparisonEligible || row.AssetClass != "rwa_stock" ||
			row.Exchange != "bybit" || m == nil || m.Source != "bybit" || strings.TrimSpace(m.FullName) == "" ||
			(strings.ToLower(m.SymbolType) != "stock" && strings.ToLower(m.SymbolType) != "etf") {
			continue
		}
		if row.UnderlyingID != "" {
			byIdentity[row.UnderlyingID] = append(byIdentity[row.UnderlyingID], m)
		}
		if key := SecurityDescriptionKey(m.MarketRegion, m.UnderlyingTicker); key != "" {
			bySecurity[key] = append(bySecurity[key], m)
		}
	}
	for i := range rows {
		row := &rows[i]
		if row.ComparisonStatus != identity.ComparisonEligible || row.AssetClass != "rwa_stock" {
			continue
		}
		own := row.InstrumentMetadata
		if own != nil && strings.TrimSpace(own.FullName) != "" {
			continue
		}
		donors := byIdentity[row.UnderlyingID]
		method := "reviewed_underlying_id"
		if own != nil {
			key := SecurityDescriptionKey(own.MarketRegion, own.UnderlyingTicker)
			if key != "" {
				donors = append(append([]*InstrumentMetadata{}, donors...), bySecurity[key]...)
				method = "market_and_security_code"
			}
		}
		var selected *InstrumentMetadata
		conflict := false
		for _, donor := range donors {
			if own != nil {
				ownKey := SecurityDescriptionKey(own.MarketRegion, own.UnderlyingTicker)
				donorKey := SecurityDescriptionKey(donor.MarketRegion, donor.UnderlyingTicker)
				if ownKey != "" && donorKey != "" && ownKey != donorKey {
					conflict = true
					break
				}
			}
			if own != nil && own.MarketRegion != "" && !strings.EqualFold(own.MarketRegion, donor.MarketRegion) {
				conflict = true
				break
			}
			if selected != nil && descriptorSignature(selected) != descriptorSignature(donor) {
				conflict = true
				break
			}
			selected = donor
		}
		if selected == nil || conflict {
			continue
		}
		merged := InstrumentMetadata{}
		if own != nil {
			merged = *own
		}
		merged.FullName = selected.FullName
		merged.DisplayName = "" // A venue's contract display symbol is not portable.
		merged.MarketRegion = selected.MarketRegion
		merged.UnderlyingTicker = selected.UnderlyingTicker
		merged.SymbolType = selected.SymbolType
		merged.DescriptionSource = selected.Source
		merged.DescriptionSourceURL = selected.SourceURL
		merged.DescriptionObservedAt = selected.ObservedAt
		merged.DescriptionMatchMethod = method
		row.InstrumentMetadata = &merged
	}
}
