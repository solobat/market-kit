package identity

// Clone isolates mutable registry construction from published resolver snapshots.
// Use on registry writes/builds, never on ticker or Resolve paths.
func (r Registry) Clone() Registry {
	out := r
	out.ExchangeAliases = make(map[string]string, len(r.ExchangeAliases))
	for k, v := range r.ExchangeAliases {
		out.ExchangeAliases[k] = v
	}
	out.AssetAliases = append([]AssetAliasRule(nil), r.AssetAliases...)
	for i := range out.AssetAliases {
		out.AssetAliases[i].Aliases = append([]string(nil), out.AssetAliases[i].Aliases...)
		out.AssetAliases[i].UnitAliases = append([]AssetUnitAlias(nil), out.AssetAliases[i].UnitAliases...)
	}
	out.MarketOverrides = append([]MarketOverride(nil), r.MarketOverrides...)
	out.AssetCollisions = append([]AssetCollision(nil), r.AssetCollisions...)
	for i := range out.AssetCollisions {
		out.AssetCollisions[i].AssetClasses = append([]string(nil), out.AssetCollisions[i].AssetClasses...)
	}
	return out
}

// Overlay applies reviewed entries last, retaining unrelated published rules.
func (r Registry) Overlay(reviewed Registry) Registry {
	r = r.Clone()
	reviewed = reviewed.Clone()
	reviewed.Normalize()
	assets := map[string]AssetAliasRule{}
	for _, v := range reviewed.AssetAliases {
		assets[v.Canonical] = v
	}
	for i, v := range r.AssetAliases {
		if next, ok := assets[v.Canonical]; ok {
			merged := mergeAssetAliasRule(v, next)
			if next.AssetClass != "" {
				merged.AssetClass = next.AssetClass
			}
			if next.AssetID != "" {
				merged.AssetID = next.AssetID
			}
			if next.UnderlyingID != "" {
				merged.UnderlyingID = next.UnderlyingID
			}
			r.AssetAliases[i] = merged
			delete(assets, v.Canonical)
		}
	}
	for _, v := range assets {
		r.AssetAliases = append(r.AssetAliases, v)
	}
	overrides := map[string]MarketOverride{}
	for _, v := range reviewed.MarketOverrides {
		overrides[marketOverrideKey(v)] = v
	}
	for i, v := range r.MarketOverrides {
		k := marketOverrideKey(v)
		if next, ok := overrides[k]; ok {
			r.MarketOverrides[i] = next
			delete(overrides, k)
		}
	}
	for _, v := range overrides {
		r.MarketOverrides = append(r.MarketOverrides, v)
	}
	for k, v := range reviewed.ExchangeAliases {
		r.ExchangeAliases[k] = v
	}
	r.AssetCollisions = append(r.AssetCollisions, reviewed.AssetCollisions...)
	if reviewed.GeneratedVersion > r.GeneratedVersion {
		r.GeneratedVersion = reviewed.GeneratedVersion
	}
	r.Normalize()
	return r
}
