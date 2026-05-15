package discovery

import (
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
	ExternalURL        string    `json:"externalUrl"`
	FirstSeenAt        time.Time `json:"firstSeenAt"`
	LastSeenAt         time.Time `json:"lastSeenAt"`
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
	NeedsReview       bool                  `json:"needsReview"`
	PrimaryConfidence float64               `json:"primaryConfidence"`
	Evidence          []string              `json:"evidence"`
	Markets           []CandidateMarket     `json:"markets"`
}
