package identity

type ResolveStatus string

const (
	ResolveResolved   ResolveStatus = "resolved"
	ResolveAmbiguous  ResolveStatus = "ambiguous"
	ResolveUnresolved ResolveStatus = "unresolved"
)

type ComparisonStatus string

const (
	ComparisonEligible   ComparisonStatus = "eligible"
	ComparisonAmbiguous  ComparisonStatus = "ambiguous"
	ComparisonProhibited ComparisonStatus = "prohibited"
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
	Exchange                    string           `json:"exchange"`
	MarketType                  MarketType       `json:"marketType"`
	RawSymbol                   string           `json:"rawSymbol"`
	VenueSymbol                 string           `json:"venueSymbol"`
	CanonicalSymbol             string           `json:"canonicalSymbol"`
	BaseAsset                   string           `json:"baseAsset"`
	QuoteAsset                  string           `json:"quoteAsset"`
	AssetClass                  string           `json:"assetClass"`
	AssetID                     string           `json:"assetId,omitempty"`
	UnderlyingID                string           `json:"underlyingId,omitempty"`
	ComparisonKey               string           `json:"comparisonKey,omitempty"`
	ComparisonStatus            ComparisonStatus `json:"comparisonStatus,omitempty"`
	RegistryVersion             int              `json:"registryVersion,omitempty"`
	InstrumentKind              string           `json:"instrumentKind,omitempty"`
	ContractType                string           `json:"contractType,omitempty"`
	SettlementAsset             string           `json:"settlementAsset,omitempty"`
	ExpiryAtMs                  int64            `json:"expiryAtMs,omitempty"`
	IdentitySource              string           `json:"identitySource,omitempty"`
	UnitAlias                   string           `json:"unitAlias,omitempty"`
	UnitMultiplier              float64          `json:"unitMultiplier,omitempty"`
	CanonicalPriceMultiplier    float64          `json:"canonicalPriceMultiplier,omitempty"`
	CanonicalQuantityMultiplier float64          `json:"canonicalQuantityMultiplier,omitempty"`
}

type ResolveResult struct {
	Status     ResolveStatus    `json:"status"`
	Confidence float64          `json:"confidence"`
	Reason     string           `json:"reason"`
	Market     *MarketIdentity  `json:"market,omitempty"`
	Candidates []MarketIdentity `json:"candidates,omitempty"`
}

type Registry struct {
	GeneratedVersion int               `json:"generated_version,omitempty"`
	ExchangeAliases  map[string]string `json:"exchange_aliases"`
	AssetAliases     []AssetAliasRule  `json:"asset_aliases"`
	MarketOverrides  []MarketOverride  `json:"market_overrides"`
	AssetCollisions  []AssetCollision  `json:"asset_collisions,omitempty"`
}

type AssetCollision struct {
	Symbol       string   `json:"symbol"`
	AssetClasses []string `json:"asset_classes"`
	Reason       string   `json:"reason,omitempty"`
}

type AssetUnitAlias struct {
	Alias      string  `json:"alias"`
	Multiplier float64 `json:"multiplier"`
}

type AssetAliasRule struct {
	Canonical    string           `json:"canonical"`
	AssetClass   string           `json:"asset_class"`
	AssetID      string           `json:"asset_id,omitempty"`
	UnderlyingID string           `json:"underlying_id,omitempty"`
	Aliases      []string         `json:"aliases"`
	UnitAliases  []AssetUnitAlias `json:"unit_aliases,omitempty"`
}

type MarketOverride struct {
	Exchange         string           `json:"exchange"`
	RawSymbol        string           `json:"raw_symbol"`
	MarketType       string           `json:"market_type"`
	CanonicalSymbol  string           `json:"canonical_symbol"`
	AssetClass       string           `json:"asset_class,omitempty"`
	AssetID          string           `json:"asset_id,omitempty"`
	UnderlyingID     string           `json:"underlying_id,omitempty"`
	ComparisonKey    string           `json:"comparison_key,omitempty"`
	ComparisonStatus ComparisonStatus `json:"comparison_status,omitempty"`
	InstrumentKind   string           `json:"instrument_kind,omitempty"`
	ContractType     string           `json:"contract_type,omitempty"`
	SettlementAsset  string           `json:"settlement_asset,omitempty"`
	ExpiryAtMs       int64            `json:"expiry_at_ms,omitempty"`
	UnitAlias        string           `json:"unit_alias,omitempty"`
	UnitMultiplier   float64          `json:"unit_multiplier,omitempty"`
}

func (r Registry) Merge(other Registry) Registry {
	r.Normalize()
	other.Normalize()

	out := Registry{
		GeneratedVersion: maxInt(r.GeneratedVersion, other.GeneratedVersion),
		ExchangeAliases:  make(map[string]string, len(r.ExchangeAliases)+len(other.ExchangeAliases)),
		AssetAliases:     append([]AssetAliasRule(nil), r.AssetAliases...),
		MarketOverrides:  append([]MarketOverride(nil), r.MarketOverrides...),
		AssetCollisions:  append([]AssetCollision(nil), r.AssetCollisions...),
	}

	for key, value := range r.ExchangeAliases {
		out.ExchangeAliases[key] = value
	}
	for key, value := range other.ExchangeAliases {
		if _, exists := out.ExchangeAliases[key]; !exists {
			out.ExchangeAliases[key] = value
		}
	}

	assetIndex := map[string]int{}
	for idx, item := range out.AssetAliases {
		assetIndex[item.Canonical] = idx
	}
	for _, item := range other.AssetAliases {
		if existing, ok := assetIndex[item.Canonical]; ok {
			out.AssetAliases[existing] = mergeAssetAliasRule(out.AssetAliases[existing], item)
			continue
		}
		out.AssetAliases = append(out.AssetAliases, item)
		assetIndex[item.Canonical] = len(out.AssetAliases) - 1
	}

	overrideKeys := map[string]bool{}
	for _, item := range out.MarketOverrides {
		overrideKeys[marketOverrideKey(item)] = true
	}
	for _, item := range other.MarketOverrides {
		key := marketOverrideKey(item)
		if overrideKeys[key] {
			continue
		}
		out.MarketOverrides = append(out.MarketOverrides, item)
		overrideKeys[key] = true
	}

	collisionIndex := map[string]int{}
	for idx, item := range out.AssetCollisions {
		collisionIndex[item.Symbol] = idx
	}
	for _, item := range other.AssetCollisions {
		if existing, ok := collisionIndex[item.Symbol]; ok {
			out.AssetCollisions[existing] = mergeAssetCollision(out.AssetCollisions[existing], item)
			continue
		}
		out.AssetCollisions = append(out.AssetCollisions, item)
		collisionIndex[item.Symbol] = len(out.AssetCollisions) - 1
	}

	out.Normalize()
	return out
}

func maxInt(left int, right int) int {
	if right > left {
		return right
	}
	return left
}

func marketOverrideKey(item MarketOverride) string {
	return item.Exchange + "|" + item.RawSymbol + "|" + item.MarketType
}

func mergeAssetAliasRule(left AssetAliasRule, right AssetAliasRule) AssetAliasRule {
	out := AssetAliasRule{
		Canonical:    firstNonEmpty(left.Canonical, right.Canonical),
		AssetClass:   firstNonEmpty(left.AssetClass, right.AssetClass),
		AssetID:      firstNonEmpty(left.AssetID, right.AssetID),
		UnderlyingID: firstNonEmpty(left.UnderlyingID, right.UnderlyingID),
		Aliases:      append([]string(nil), left.Aliases...),
		UnitAliases:  append([]AssetUnitAlias(nil), left.UnitAliases...),
	}

	aliasSet := map[string]bool{}
	for _, alias := range out.Aliases {
		if alias == "" || alias == out.Canonical {
			continue
		}
		aliasSet[alias] = true
	}
	for _, alias := range right.Aliases {
		if alias == "" || alias == out.Canonical || aliasSet[alias] {
			continue
		}
		out.Aliases = append(out.Aliases, alias)
		aliasSet[alias] = true
	}

	unitAliasIndex := map[string]int{}
	for idx, alias := range out.UnitAliases {
		if alias.Alias == "" || alias.Alias == out.Canonical {
			continue
		}
		unitAliasIndex[alias.Alias] = idx
	}
	for _, alias := range right.UnitAliases {
		if alias.Alias == "" || alias.Alias == out.Canonical {
			continue
		}
		if existing, ok := unitAliasIndex[alias.Alias]; ok {
			if out.UnitAliases[existing].Multiplier == 0 && alias.Multiplier > 0 {
				out.UnitAliases[existing].Multiplier = alias.Multiplier
			}
			continue
		}
		out.UnitAliases = append(out.UnitAliases, alias)
		unitAliasIndex[alias.Alias] = len(out.UnitAliases) - 1
	}

	return out
}

func mergeAssetCollision(left AssetCollision, right AssetCollision) AssetCollision {
	out := AssetCollision{
		Symbol:       firstNonEmpty(left.Symbol, right.Symbol),
		AssetClasses: append([]string(nil), left.AssetClasses...),
		Reason:       firstNonEmpty(left.Reason, right.Reason),
	}
	seen := map[string]bool{}
	for _, value := range out.AssetClasses {
		seen[value] = true
	}
	for _, value := range right.AssetClasses {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out.AssetClasses = append(out.AssetClasses, value)
	}
	return out
}
