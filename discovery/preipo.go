package discovery

import "strings"

const XYZPreIPOURL = "https://docs.trade.xyz/perpetuals/specifications-and-schedules/pre-ipo-specification-index.md"

// Issuer association never establishes equal contract units across venues.
func OfficialPreIPOIssuer(item ImportedMarket) (string, bool) {
	m := item.InstrumentMetadata
	if m == nil || item.PlatformID != "hyperliquid" || !strings.HasPrefix(strings.ToLower(item.Symbol), "xyz:") || m.PreIPOEvidenceURL != XYZPreIPOURL || m.PreIPOIssuer == "" {
		return "", false
	}
	return m.PreIPOIssuer, true
}
