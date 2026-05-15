package discovery

import (
	"regexp"
	"strings"
)

var gateLeveragedTokenPattern = regexp.MustCompile(`[0-9]+[LS]$`)

func ShouldIgnoreImportedMarket(item ImportedMarket) bool {
	return IsExcludedLeveragedToken(item.PlatformID, item.BaseAsset, item.Symbol)
}

func IsExcludedLeveragedToken(platformID string, baseAsset string, rawSymbol string) bool {
	platformID = strings.ToLower(strings.TrimSpace(platformID))
	if platformID != "gate" {
		return false
	}

	base := strings.ToUpper(strings.TrimSpace(baseAsset))
	symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	return gateLeveragedTokenPattern.MatchString(base) || gateLeveragedTokenPattern.MatchString(symbol)
}
