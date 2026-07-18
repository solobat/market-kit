import baseRegistry from "../../../identity/default_registry.json";

const quoteSuffixes = ["USDT", "USDC", "USD"];
let registry = normalizeRegistry(baseRegistry);

export function loadRegistry() {
  return registry;
}

export function setRuntimeRegistry(input) {
  registry = normalizeRegistry(input);
  return registry;
}

export function upsertRuntimeAssetAlias(input) {
  const asset = normalizeAssetAliasEntry(input);
  if (!asset?.canonical) return registry;

  const items = [...(registry.asset_aliases || [])];
  const index = items.findIndex((item) => String(item.canonical || "").toUpperCase() === asset.canonical);
  if (index >= 0) {
    items[index] = mergeAssetAliasEntry(items[index], asset);
    items[index].asset_class = asset.asset_class || items[index].asset_class;
    items[index].aliases = normalizeAliasList(items[index].canonical, items[index].aliases);
    items[index].unit_aliases = normalizeUnitAliasList(items[index].canonical, items[index].unit_aliases);
  } else {
    items.push({
      ...asset,
      aliases: normalizeAliasList(asset.canonical, asset.aliases),
      unit_aliases: normalizeUnitAliasList(asset.canonical, asset.unit_aliases)
    });
  }

  registry = {
    ...registry,
    asset_aliases: items.sort((left, right) => String(left.canonical || "").localeCompare(String(right.canonical || "")))
  };
  return registry;
}

export function upsertRuntimeMarketOverride(input) {
  const override = normalizeMarketOverrideItem(input, new Map());
  if (!override) return registry;

  const key = marketOverrideKey(override);
  const items = (registry.market_overrides || []).filter((item) => marketOverrideKey(item) !== key);
  items.push(override);
  registry = { ...registry, market_overrides: items };
  return registry;
}

function normalizeRegistry(input) {
  const exchange_aliases = {};
  for (const [source, target] of Object.entries(input?.exchange_aliases || {})) {
    const normalizedSource = String(source || "").trim().toLowerCase();
    const normalizedTarget = String(target || "").trim().toLowerCase();
    if (!normalizedSource || !normalizedTarget) continue;
    exchange_aliases[normalizedSource] = normalizedTarget;
  }

  const [asset_aliases, scaledAliases] = normalizeAssetAliases(input?.asset_aliases || []);
  const market_overrides = normalizeMarketOverrides(input?.market_overrides || [], scaledAliases);
  return { exchange_aliases, asset_aliases, market_overrides };
}

function normalizeAssetAliases(items) {
  const merged = new Map();
  for (const item of items || []) {
    const normalized = normalizeAssetAliasEntry(item);
    if (!normalized?.canonical) continue;
    merged.set(normalized.canonical, mergeAssetAliasEntry(merged.get(normalized.canonical), normalized));
  }

  const canonicalSet = new Set(merged.keys());
  const scaledAliases = new Map();
  for (const canonical of Array.from(merged.keys())) {
    const current = merged.get(canonical);
    if (!current) continue;
    const scaled = inferScaledUnitAlias(canonical, canonicalSet);
    if (!scaled || !merged.has(scaled.base)) continue;

    const baseRule = merged.get(scaled.base);
    const incoming = {
      canonical: scaled.base,
      asset_class: current.asset_class,
      aliases: [],
      unit_aliases: [
        { alias: canonical, multiplier: scaled.multiplier },
        ...(current.aliases || []).map((alias) => ({ alias, multiplier: scaled.multiplier })),
        ...(current.unit_aliases || []).map((alias) => ({
          alias: alias.alias,
          multiplier: Number(alias.multiplier || scaled.multiplier) || scaled.multiplier
        }))
      ]
    };

    merged.set(scaled.base, mergeAssetAliasEntry(baseRule, incoming));
    merged.delete(canonical);
    canonicalSet.delete(canonical);
    scaledAliases.set(canonical, scaled.base);
  }

  const normalized = Array.from(merged.values())
    .map((item) => ({
      canonical: item.canonical,
      asset_class: item.asset_class,
      aliases: normalizeAliasList(item.canonical, item.aliases),
      unit_aliases: normalizeUnitAliasList(item.canonical, item.unit_aliases)
    }))
    .sort((left, right) => String(left.canonical || "").localeCompare(String(right.canonical || "")));

  return [normalized, scaledAliases];
}

function normalizeAssetAliasEntry(item) {
  const canonical = String(item?.canonical || "").trim().toUpperCase();
  if (!canonical) return null;
  return {
    canonical,
    asset_class: String(item?.asset_class || item?.assetClass || "").trim(),
    aliases: (item?.aliases || []).map((alias) => String(alias || "").trim().toUpperCase()),
    unit_aliases: (item?.unit_aliases || item?.unitAliases || []).map((alias) => ({
      alias: String(alias?.alias || "").trim().toUpperCase(),
      multiplier: Number(alias?.multiplier || 0)
    }))
  };
}

function normalizeMarketOverrides(items, scaledAliases) {
  const seen = new Set();
  const normalized = [];
  for (const item of items || []) {
    const override = normalizeMarketOverrideItem(item, scaledAliases);
    if (!override) continue;
    const key = marketOverrideKey(override);
    if (seen.has(key)) continue;
    seen.add(key);
    normalized.push(override);
  }
  return normalized;
}

function normalizeMarketOverrideItem(item, scaledAliases) {
  const exchange = String(item?.exchange || "").trim().toLowerCase();
  const raw_symbol = String(item?.raw_symbol || item?.rawSymbol || "").trim();
  const market_type = normalizeMarketType(item?.market_type || item?.marketType);
  let canonical_symbol = String(item?.canonical_symbol || item?.canonicalSymbol || "").trim().toUpperCase();
  const unit_alias = String(item?.unit_alias || item?.unitAlias || "").trim().toUpperCase();
  const unit_multiplier = Number(item?.unit_multiplier || item?.unitMultiplier || 0);
  const [base, quote] = splitCanonical(canonical_symbol);
  if (base && quote && scaledAliases?.has?.(base)) {
    canonical_symbol = `${scaledAliases.get(base)}/${quote}`;
  }
  if (!exchange || !raw_symbol || !market_type || !canonical_symbol) return null;
  return {
    exchange,
    raw_symbol,
    market_type,
    canonical_symbol,
    ...(Number.isFinite(unit_multiplier) && unit_multiplier > 0 ? { unit_alias, unit_multiplier } : {})
  };
}

function marketOverrideKey(item) {
  const exchange = String(item?.exchange || "").trim().toLowerCase();
  return `${exchange}|${String(item?.raw_symbol || item?.rawSymbol || "").trim().toUpperCase()}|${normalizeMarketType(item?.market_type || item?.marketType)}`;
}

function mergeAssetAliasEntry(left, right) {
  const current = left || { canonical: right?.canonical || "", asset_class: "", aliases: [], unit_aliases: [] };
  const merged = {
    canonical: current.canonical || right?.canonical || "",
    asset_class: current.asset_class || String(right?.asset_class || "").trim(),
    aliases: [...(current.aliases || [])],
    unit_aliases: [...(current.unit_aliases || [])]
  };

  const aliasSet = new Set(merged.aliases.filter((alias) => alias && alias !== merged.canonical));
  for (const alias of right?.aliases || []) {
    if (!alias || alias === merged.canonical || aliasSet.has(alias)) continue;
    merged.aliases.push(alias);
    aliasSet.add(alias);
  }

  const unitAliasIndex = new Map();
  for (const [index, alias] of (merged.unit_aliases || []).entries()) {
    if (!alias?.alias || alias.alias === merged.canonical) continue;
    unitAliasIndex.set(alias.alias, index);
  }
  for (const alias of right?.unit_aliases || []) {
    if (!alias?.alias || alias.alias === merged.canonical) continue;
    if (unitAliasIndex.has(alias.alias)) {
      const existing = merged.unit_aliases[unitAliasIndex.get(alias.alias)];
      if (!Number(existing.multiplier) && Number(alias.multiplier) > 0) {
        existing.multiplier = Number(alias.multiplier);
      }
      continue;
    }
    merged.unit_aliases.push({ alias: alias.alias, multiplier: Number(alias.multiplier || 0) });
    unitAliasIndex.set(alias.alias, merged.unit_aliases.length - 1);
  }

  return merged;
}

function normalizeAliasList(canonical, aliases) {
  return Array.from(
    new Set(
      (aliases || [])
        .map((alias) => String(alias || "").trim().toUpperCase())
        .filter((alias) => alias && alias !== canonical)
    )
  ).sort((left, right) => left.localeCompare(right));
}

function normalizeUnitAliasList(canonical, aliases) {
  const map = new Map();
  for (const item of aliases || []) {
    const alias = String(item?.alias || "").trim().toUpperCase();
    const multiplier = Number(item?.multiplier || 0);
    if (!alias || alias === canonical || multiplier <= 0 || map.has(alias)) continue;
    map.set(alias, { alias, multiplier });
  }
  return Array.from(map.values()).sort((left, right) => {
    if (left.multiplier === right.multiplier) return left.alias.localeCompare(right.alias);
    return left.multiplier - right.multiplier;
  });
}

function inferScaledUnitAlias(value, canonicalSet) {
  const normalized = String(value || "").trim().toUpperCase();
  if (!normalized) return null;

  const prefixes = [
    { token: "1000000", multiplier: 1000000 },
    { token: "10000", multiplier: 10000 },
    { token: "1000", multiplier: 1000 },
    { token: "100", multiplier: 100 },
    { token: "1M", multiplier: 1000000 }
  ];

  for (const prefix of prefixes) {
    if (normalized.startsWith(prefix.token) && normalized.length > prefix.token.length) {
      const base = normalized.slice(prefix.token.length);
      if (isValidScaledBase(base) && canonicalSet.has(base)) {
        return { base, multiplier: prefix.multiplier };
      }
    }
    if (normalized.endsWith(prefix.token) && normalized.length > prefix.token.length) {
      const base = normalized.slice(0, normalized.length - prefix.token.length);
      if (isValidScaledBase(base) && canonicalSet.has(base)) {
        return { base, multiplier: prefix.multiplier };
      }
    }
  }
  return null;
}

function isValidScaledBase(value) {
  return /^[A-Z0-9]{2,}$/.test(String(value || "").trim().toUpperCase());
}

export function normalizeExchange(value) {
  const raw = String(value || "").trim().toLowerCase();
  return registry.exchange_aliases?.[raw] || raw;
}

export function normalizeMarketType(value) {
  const raw = String(value || "").trim().toLowerCase();
  if (raw === "spot") return "spot";
  if (["perpetual", "perp", "swap", "linear"].includes(raw)) return "perpetual";
  if (["future", "futures", "delivery"].includes(raw)) return "future";
  return "";
}

export function resolveIdentity(request) {
  const exchange = normalizeExchange(request.exchange);
  const symbol = String(request.symbol || "").trim();
  if (!exchange) {
    return unresolved("exchange is required");
  }
  if (!symbol) {
    return unresolved("symbol is required");
  }

  const hintedMarketType = normalizeMarketType(request.marketTypeHint);
  const overrideMatches = registry.market_overrides.filter((item) => {
    if (normalizeExchange(item.exchange) !== exchange) return false;
    if (String(item.raw_symbol || "").trim().toUpperCase() !== symbol.toUpperCase()) return false;
    if (hintedMarketType && normalizeMarketType(item.market_type) !== hintedMarketType) return false;
    return true;
  });
  if (overrideMatches.length === 1) {
    return {
      status: "resolved",
      confidence: 1,
      reason: "matched explicit market override",
      market: toIdentity(exchange, symbol, overrideMatches[0].market_type, overrideMatches[0].canonical_symbol, overrideMatches[0])
    };
  }
  if (overrideMatches.length > 1) {
    return {
      status: "ambiguous",
      confidence: 0.5,
      reason: "multiple explicit market overrides matched",
      candidates: overrideMatches.map((item) => toIdentity(exchange, symbol, item.market_type, item.canonical_symbol, item))
    };
  }

  const inferred = inferMarketType({
    exchange,
    symbol,
    marketTypeHint: request.marketTypeHint,
    instType: request.instType,
    productType: request.productType
  });
  if (!inferred.marketType) {
    return unresolved("market type could not be inferred");
  }

  const pair = derivePair(exchange, symbol, inferred.marketType, request.canonicalSymbolHint);
  if (!pair) {
    return unresolved("base/quote could not be derived");
  }

  const alias = resolveAssetAlias(pair.base);
  if (alias.ambiguous) {
    return {
      status: "ambiguous",
      confidence: 0.55,
      reason: "base asset alias matched multiple candidates"
    };
  }

  const base = alias.canonical || pair.base;
  return {
    status: "resolved",
    confidence: inferred.confident ? 0.95 : 0.82,
    reason: inferred.confident ? "resolved using exchange-specific market inference" : "resolved using heuristics",
    market: {
      exchange,
      marketType: inferred.marketType,
      rawSymbol: symbol,
      venueSymbol: normalizeVenueSymbol(exchange, symbol, inferred.marketType),
      canonicalSymbol: `${base}/${pair.quote}`,
      baseAsset: base,
      quoteAsset: pair.quote,
      assetClass: alias.assetClass || "unknown",
      ...unitConversionFields(alias)
    }
  };
}

function unresolved(reason) {
  return {
    status: "unresolved",
    confidence: 0,
    reason
  };
}

function toIdentity(exchange, symbol, marketType, canonicalSymbol, override = null) {
  const [base, quote] = splitCanonical(canonicalSymbol);
  const alias = resolveAssetAlias(base);
  const canonicalBase = alias.canonical || base;
  const rawPair = derivePair(exchange, symbol, marketType, "");
  const rawAlias = rawPair ? resolveAssetAlias(rawPair.base) : null;
  const overrideUnitMultiplier = Number(override?.unit_multiplier || override?.unitMultiplier || 0);
  const overrideUnitAlias = String(override?.unit_alias || override?.unitAlias || rawPair?.base || canonicalBase || "").trim().toUpperCase();
  const unitAlias =
    Number.isFinite(overrideUnitMultiplier) && overrideUnitMultiplier > 0
      ? { unitAlias: overrideUnitAlias, unitMultiplier: overrideUnitMultiplier }
      : rawAlias?.unitMultiplier > 0 && (rawAlias.canonical || rawPair.base) === canonicalBase ? rawAlias : alias;
  return {
    exchange,
    marketType,
    rawSymbol: symbol,
    venueSymbol: normalizeVenueSymbol(exchange, symbol, marketType),
    canonicalSymbol,
    baseAsset: canonicalBase,
    quoteAsset: quote,
    assetClass: alias.assetClass || "unknown",
    ...unitConversionFields(unitAlias)
  };
}

function resolveAssetAlias(base) {
  const normalized = String(base || "").trim().toUpperCase();
  const matches = registry.asset_aliases.filter((item) => {
    if (String(item.canonical || "").trim().toUpperCase() === normalized) return true;
    if ((item.aliases || []).some((alias) => String(alias || "").trim().toUpperCase() === normalized)) return true;
    return (item.unit_aliases || []).some((alias) => String(alias?.alias || "").trim().toUpperCase() === normalized);
  });
  if (matches.length === 1) {
    const unitAlias = (matches[0].unit_aliases || []).find(
      (alias) => String(alias?.alias || "").trim().toUpperCase() === normalized
    );
    return {
      canonical: matches[0].canonical,
      assetClass: matches[0].asset_class,
      unitAlias: unitAlias?.alias || "",
      unitMultiplier: Number(unitAlias?.multiplier || 0),
      ambiguous: false
    };
  }
  if (matches.length > 1) {
    return { canonical: "", assetClass: "", ambiguous: true };
  }
  return { canonical: "", assetClass: "", ambiguous: false };
}

function unitConversionFields(alias) {
  const multiplier = Number(alias?.unitMultiplier || 0);
  if (!alias?.unitAlias || !Number.isFinite(multiplier) || multiplier <= 0) return {};
  return {
    unitAlias: alias.unitAlias,
    unitMultiplier: multiplier,
    canonicalPriceMultiplier: 1 / multiplier,
    canonicalQuantityMultiplier: multiplier
  };
}

function inferMarketType({ exchange, symbol, marketTypeHint, instType, productType }) {
  const hinted = normalizeMarketType(marketTypeHint);
  if (hinted) return { marketType: hinted, confident: true };

  const inst = String(instType || "").trim().toLowerCase();
  const product = String(productType || "").trim().toLowerCase();
  const raw = String(symbol || "").trim().toUpperCase();

  if (inst.includes("spot") || product.includes("spot")) return { marketType: "spot", confident: true };
  if (inst.includes("swap") || inst.includes("perp") || product.includes("swap") || product.includes("perp")) {
    return { marketType: "perpetual", confident: true };
  }
  if (inst.includes("future") || product.includes("future")) return { marketType: "future", confident: true };

  if (exchange === "okx") {
    if (raw.endsWith("-SWAP")) return { marketType: "perpetual", confident: true };
    if ((raw.match(/-/g) || []).length === 1) return { marketType: "spot", confident: true };
  }

  if (exchange === "hyperliquid") {
    if (raw.includes("/")) return { marketType: "spot", confident: true };
    return { marketType: "perpetual", confident: true };
  }

  if (["binance", "bybit", "bitget", "gate", "aster"].includes(exchange)) {
    if (/[\/\-_]/.test(raw)) return { marketType: "spot", confident: false };
    return { marketType: "perpetual", confident: false };
  }

  return { marketType: "", confident: false };
}

function derivePair(exchange, symbol, marketType, canonicalHint) {
  if (canonicalHint) {
    const split = splitCanonical(canonicalHint);
    if (split[0] && split[1]) return { base: split[0], quote: split[1] };
  }

  let raw = String(symbol || "").trim().toUpperCase();
  if (exchange === "okx" && raw.endsWith("-SWAP")) raw = raw.slice(0, -5);
  if (exchange === "hyperliquid" && marketType === "perpetual" && !raw.includes("/")) {
    return { base: raw.split(":")[0], quote: "USDT" };
  }

  for (const sep of ["/", "-", "_"]) {
    if (raw.includes(sep)) {
      const [base, quote] = raw.split(sep);
      if (base && quote) return { base, quote };
    }
  }

  for (const quote of quoteSuffixes) {
    if (raw.endsWith(quote) && raw.length > quote.length) {
      return { base: raw.slice(0, -quote.length), quote };
    }
  }
  return null;
}

function normalizeVenueSymbol(exchange, symbol, marketType) {
  let raw = String(symbol || "").trim().toUpperCase();
  if (exchange === "okx") {
    if (marketType === "perpetual" && !raw.endsWith("-SWAP")) {
      const pair = derivePair(exchange, raw, "spot", "");
      if (pair) return `${pair.base}-${pair.quote}-SWAP`;
    }
    if (marketType === "spot" && raw.endsWith("-SWAP")) {
      return raw.slice(0, -5);
    }
    return raw;
  }
  if (exchange === "gate") {
    return raw.replaceAll("/", "_").replaceAll("-", "_");
  }
  if (["binance", "bybit", "bitget", "aster"].includes(exchange)) {
    return raw.replaceAll("/", "").replaceAll("-", "").replaceAll("_", "");
  }
  if (exchange === "hyperliquid") {
    if (marketType === "spot") return raw;
    return raw.split(/[/:_-]/)[0];
  }
  return raw;
}

function splitCanonical(value) {
  const raw = String(value || "").trim().toUpperCase();
  if (!raw.includes("/")) return ["", ""];
  const [base, quote] = raw.split("/");
  return [base || "", quote || ""];
}

export function registryStats(input = registry) {
  const source = input || registry;
  const exchangeSet = new Set([
    ...Object.keys(source.exchange_aliases || {}),
    ...(source.market_overrides || []).map((item) => normalizeExchange(item.exchange))
  ]);
  const assetClasses = new Set((source.asset_aliases || []).map((item) => item.asset_class).filter(Boolean));
  return {
    assets: source.asset_aliases.length,
    overrides: source.market_overrides.length,
    exchangeAliases: Object.keys(source.exchange_aliases || {}).length,
    exchanges: exchangeSet.size,
    assetClasses: assetClasses.size
  };
}

export function normalizeImportedCases(payload) {
  const rows = extractRows(payload);
  return rows
    .map((item, index) => normalizeImportedCase(item, index))
    .filter(Boolean);
}

function extractRows(payload) {
  if (Array.isArray(payload)) return payload;
  if (!payload || typeof payload !== "object") return [];
  if (Array.isArray(payload.cases)) return payload.cases;
  if (Array.isArray(payload.items)) return payload.items;
  if (Array.isArray(payload.data)) return payload.data;
  if (payload.data && Array.isArray(payload.data.items)) return payload.data.items;
  if (payload.data && Array.isArray(payload.data.cases)) return payload.data.cases;
  return [];
}

function normalizeImportedCase(item, index) {
  if (!item || typeof item !== "object") return null;

  const exchange = normalizeExchange(
    item.exchange ||
      item.platform ||
      item.market?.exchange ||
      item.input?.exchange ||
      item.primary?.exchange ||
      item.secondary?.exchange
  );

  const symbol =
    String(
      item.symbol ||
        item.rawSymbol ||
        item.raw_symbol ||
        item.market?.rawSymbol ||
        item.input?.symbol ||
        item.input?.rawSymbol ||
        item.primary?.rawSymbol ||
        item.primary?.symbol ||
        ""
    ).trim() || "";

  const marketTypeHint =
    String(
      item.marketTypeHint ||
        item.market_type_hint ||
        item.marketType ||
        item.market_type ||
        item.market?.marketType ||
        item.input?.marketTypeHint ||
        item.input?.marketType ||
        ""
    ).trim() || "";

  const canonicalSymbolHint =
    String(
      item.canonicalSymbolHint ||
        item.canonical_symbol_hint ||
        item.canonicalSymbol ||
        item.canonical_symbol ||
        item.market?.canonicalSymbol ||
        item.input?.canonicalSymbolHint ||
        item.input?.canonicalSymbol ||
        ""
    ).trim() || "";

  const instType = String(item.instType || item.inst_type || item.input?.instType || "").trim() || "";
  const productType =
    String(item.productType || item.product_type || item.input?.productType || "").trim() || "";

  const status = String(item.status || item.identityStatus || "").trim().toLowerCase();
  const source = String(item.source || item.project || item.system || "unknown").trim();
  const reason = String(item.reason || item.message || item.error || "").trim();
  const firstSeenAt = String(item.firstSeenAt || item.first_seen_at || item.createdAt || item.created_at || "").trim();
  const lastSeenAt = String(item.lastSeenAt || item.last_seen_at || item.updatedAt || item.updated_at || "").trim();
  const count = Number(item.count || item.hits || item.frequency || 1) || 1;

  const request = {
    exchange,
    symbol,
    marketTypeHint,
    canonicalSymbolHint,
    instType,
    productType
  };

  const resolution = resolveIdentity(request);

  return {
    id: String(item.id || item.caseId || item.case_id || `${source}:${exchange}:${symbol}:${index}`),
    source,
    status: status === "ambiguous" || status === "unresolved" ? status : resolution.status,
    reason,
    firstSeenAt,
    lastSeenAt,
    count,
    request,
    resolution
  };
}
