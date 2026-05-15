package identity

type ResolveStatus string

const (
	ResolveResolved   ResolveStatus = "resolved"
	ResolveAmbiguous  ResolveStatus = "ambiguous"
	ResolveUnresolved ResolveStatus = "unresolved"
)

type MarketType string

const (
	MarketTypeSpot      MarketType = "spot"
	MarketTypePerpetual MarketType = "perpetual"
	MarketTypeFuture    MarketType = "future"
	MarketTypeUnknown   MarketType = "unknown"
)

type ResolveRequest struct {
	Exchange            string
	Symbol              string
	CanonicalSymbolHint string
	MarketTypeHint      string
	InstType            string
	ProductType         string
}

type MarketIdentity struct {
	Exchange        string     `json:"exchange"`
	MarketType      MarketType `json:"marketType"`
	RawSymbol       string     `json:"rawSymbol"`
	VenueSymbol     string     `json:"venueSymbol"`
	CanonicalSymbol string     `json:"canonicalSymbol"`
	BaseAsset       string     `json:"baseAsset"`
	QuoteAsset      string     `json:"quoteAsset"`
	AssetClass      string     `json:"assetClass"`
}

type ResolveResult struct {
	Status     ResolveStatus    `json:"status"`
	Confidence float64          `json:"confidence"`
	Reason     string           `json:"reason"`
	Market     *MarketIdentity  `json:"market,omitempty"`
	Candidates []MarketIdentity `json:"candidates,omitempty"`
}

type Registry struct {
	ExchangeAliases map[string]string `json:"exchange_aliases"`
	AssetAliases    []AssetAliasRule  `json:"asset_aliases"`
	MarketOverrides []MarketOverride  `json:"market_overrides"`
}

type AssetAliasRule struct {
	Canonical  string   `json:"canonical"`
	AssetClass string   `json:"asset_class"`
	Aliases    []string `json:"aliases"`
}

type MarketOverride struct {
	Exchange        string `json:"exchange"`
	RawSymbol       string `json:"raw_symbol"`
	MarketType      string `json:"market_type"`
	CanonicalSymbol string `json:"canonical_symbol"`
}
