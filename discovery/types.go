package discovery

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/solobat/market-kit/identity"
)

type SourceKind string

const (
	SourceKindSlipstream SourceKind = "slipstream"
	SourceKindBootstrap  SourceKind = "market-kit-bootstrap"
)

type ImportEnvelope struct {
	Source      SourceKind       `json:"source"`
	GeneratedAt time.Time        `json:"generatedAt"`
	Items       []ImportedMarket `json:"items"`
}

type ImportedMarket struct {
	SourceID           string    `json:"sourceId"`
	PlatformID         string    `json:"platformId"`
	Platform           string    `json:"platform"`
	VenueType          string    `json:"venueType"`
	MarketType         string    `json:"marketType"`
	Symbol             string    `json:"symbol"`
	BaseAsset          string    `json:"baseAsset"`
	QuoteAsset         string    `json:"quoteAsset"`
	AssetClass         string    `json:"assetClass"`
	AssetClassHint     string    `json:"assetClassHint"`
	Category           string    `json:"category"`
	UnderlyingCategory string    `json:"underlyingCategory"`
	Tags               []string  `json:"tags"`
	Chain              string    `json:"chain"`
	Status             string    `json:"status"`
	ST                 bool      `json:"st,omitempty"`
	PreDelisting       bool      `json:"preDelisting,omitempty"`
	Flags              []string  `json:"flags,omitempty"`
	ExternalURL        string    `json:"externalUrl"`
	FirstSeenAt        time.Time `json:"firstSeenAt"`
	LastSeenAt         time.Time `json:"lastSeenAt"`
}

func (market *ImportedMarket) UnmarshalJSON(payload []byte) error {
	type importedMarketJSON struct {
		SourceID           string    `json:"sourceId"`
		PlatformID         string    `json:"platformId"`
		Platform           string    `json:"platform"`
		VenueType          string    `json:"venueType"`
		MarketType         string    `json:"marketType"`
		Symbol             string    `json:"symbol"`
		BaseAsset          string    `json:"baseAsset"`
		QuoteAsset         string    `json:"quoteAsset"`
		AssetClass         string    `json:"assetClass"`
		AssetClassHint     string    `json:"assetClassHint"`
		Category           string    `json:"category"`
		UnderlyingCategory string    `json:"underlyingCategory"`
		Tags               []string  `json:"tags"`
		Chain              string    `json:"chain"`
		Status             string    `json:"status"`
		ST                 any       `json:"st"`
		IsST               any       `json:"isST"`
		IsSTSnake          any       `json:"is_st"`
		PreDelisting       any       `json:"preDelisting"`
		PreDelistingSnake  any       `json:"pre_delisting"`
		InDelisting        any       `json:"in_delisting"`
		Flags              []string  `json:"flags"`
		StatusFlags        []string  `json:"statusFlags"`
		ExternalURL        string    `json:"externalUrl"`
		FirstSeenAt        time.Time `json:"firstSeenAt"`
		LastSeenAt         time.Time `json:"lastSeenAt"`
	}

	var raw importedMarketJSON
	if err := json.Unmarshal(payload, &raw); err != nil {
		return err
	}

	st := boolish(raw.ST) || boolish(raw.IsST) || boolish(raw.IsSTSnake)
	preDelisting := boolish(raw.PreDelisting) || boolish(raw.PreDelistingSnake) || boolish(raw.InDelisting)
	flags := NormalizeMarketFlags(append(raw.Flags, raw.StatusFlags...), st, preDelisting)
	st = hasMarketFlag(flags, MarketFlagST)
	preDelisting = hasMarketFlag(flags, MarketFlagPreDelisting)

	*market = ImportedMarket{
		SourceID:           raw.SourceID,
		PlatformID:         raw.PlatformID,
		Platform:           raw.Platform,
		VenueType:          raw.VenueType,
		MarketType:         raw.MarketType,
		Symbol:             raw.Symbol,
		BaseAsset:          raw.BaseAsset,
		QuoteAsset:         raw.QuoteAsset,
		AssetClass:         raw.AssetClass,
		AssetClassHint:     raw.AssetClassHint,
		Category:           raw.Category,
		UnderlyingCategory: raw.UnderlyingCategory,
		Tags:               raw.Tags,
		Chain:              raw.Chain,
		Status:             raw.Status,
		ST:                 st,
		PreDelisting:       preDelisting,
		Flags:              flags,
		ExternalURL:        raw.ExternalURL,
		FirstSeenAt:        raw.FirstSeenAt,
		LastSeenAt:         raw.LastSeenAt,
	}
	return nil
}

type CandidateMarket struct {
	SourceID        string              `json:"sourceId"`
	PlatformID      string              `json:"platformId"`
	Platform        string              `json:"platform"`
	Exchange        string              `json:"exchange"`
	VenueType       string              `json:"venueType"`
	MarketType      identity.MarketType `json:"marketType"`
	RawSymbol       string              `json:"rawSymbol"`
	VenueSymbol     string              `json:"venueSymbol"`
	BaseAsset       string              `json:"baseAsset"`
	QuoteAsset      string              `json:"quoteAsset"`
	CanonicalSymbol string              `json:"canonicalSymbol"`
	AssetClass      string              `json:"assetClass"`
	Chain           string              `json:"chain"`
	Status          string              `json:"status"`
	ST              bool                `json:"st,omitempty"`
	PreDelisting    bool                `json:"preDelisting,omitempty"`
	Flags           []string            `json:"flags,omitempty"`
	ExternalURL     string              `json:"externalUrl"`
	Confidence      float64             `json:"confidence"`
	Evidence        []string            `json:"evidence"`
	FirstSeenAt     time.Time           `json:"firstSeenAt"`
	LastSeenAt      time.Time           `json:"lastSeenAt"`
}

type AssetCandidateGroup struct {
	GroupKey          string                `json:"groupKey"`
	CanonicalAsset    string                `json:"canonicalAsset"`
	CanonicalSymbol   string                `json:"canonicalSymbol"`
	QuoteAsset        string                `json:"quoteAsset"`
	AssetClass        string                `json:"assetClass"`
	Exchanges         []string              `json:"exchanges"`
	MarketTypes       []identity.MarketType `json:"marketTypes"`
	VenueTypes        []string              `json:"venueTypes"`
	Chains            []string              `json:"chains"`
	Flags             []string              `json:"flags,omitempty"`
	NeedsReview       bool                  `json:"needsReview"`
	PrimaryConfidence float64               `json:"primaryConfidence"`
	Evidence          []string              `json:"evidence"`
	Markets           []CandidateMarket     `json:"markets"`
}

const (
	MarketFlagST           = "st"
	MarketFlagPreDelisting = "pre_delisting"
)

func NormalizeMarketFlags(flags []string, st bool, preDelisting bool) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(flags)+2)
	for _, flag := range flags {
		normalized := normalizeMarketFlag(flag)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	if st && !seen[MarketFlagST] {
		seen[MarketFlagST] = true
		out = append(out, MarketFlagST)
	}
	if preDelisting && !seen[MarketFlagPreDelisting] {
		out = append(out, MarketFlagPreDelisting)
	}
	return out
}

func normalizeMarketFlag(value string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	raw = strings.ReplaceAll(raw, "-", "_")
	raw = strings.ReplaceAll(raw, " ", "_")
	raw = strings.Join(strings.Fields(raw), "_")
	switch raw {
	case "", "false", "0", "no", "none":
		return ""
	case "st", "special_treatment", "specialtreatment", "special":
		return MarketFlagST
	case "pre_delisting", "predelisting", "pre_delist", "predelist", "in_delisting", "indelisting", "delisting":
		return MarketFlagPreDelisting
	default:
		return raw
	}
}

func hasMarketFlag(flags []string, target string) bool {
	target = normalizeMarketFlag(target)
	for _, flag := range flags {
		if normalizeMarketFlag(flag) == target {
			return true
		}
	}
	return false
}

func boolish(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case float64:
		return typed != 0
	case string:
		raw := strings.ToLower(strings.TrimSpace(typed))
		if parsed, err := strconv.ParseBool(raw); err == nil {
			return parsed
		}
		switch raw {
		case "1", "y", "yes", "on", "enabled", "st", "special_treatment", "special treatment", "pre_delisting", "pre-delisting", "in_delisting":
			return true
		default:
			return false
		}
	default:
		return boolish(fmt.Sprint(typed))
	}
}
