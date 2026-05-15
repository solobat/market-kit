import baseRegistry from "../../../identity/default_registry.json";
import generatedRegistry from "../../../identity/generated_registry.json";

const quoteSuffixes = ["USDT", "USDC", "USD"];
const registry = mergeRegistries(baseRegistry, generatedRegistry);

export function loadRegistry() {
  return registry;
}

function mergeRegistries(left, right) {
  const merged = {
    exchange_aliases: { ...(left?.exchange_aliases || {}) },
    asset_aliases: [...(left?.asset_aliases || [])],
    market_overrides: [...(left?.market_overrides || [])]
  };

  for (const [key, value] of Object.entries(right?.exchange_aliases || {})) {
    if (!(key in merged.exchange_aliases)) {
      merged.exchange_aliases[key] = value;
    }
  }

  const assetKeys = new Set(merged.asset_aliases.map((item) => String(item.canonical || "").trim().toUpperCase()));
  for (const item of right?.asset_aliases || []) {
    const key = String(item.canonical || "").trim().toUpperCase();
    if (!key || assetKeys.has(key)) continue;
    merged.asset_aliases.push(item);
    assetKeys.add(key);
  }

  const overrideKeys = new Set(
    merged.market_overrides.map(
      (item) =>
        `${normalizeExchange(item.exchange)}|${String(item.raw_symbol || item.rawSymbol || "").trim().toUpperCase()}|${normalizeMarketType(item.market_type || item.marketType)}`
    )
  );
  for (const item of right?.market_overrides || []) {
    const key = `${normalizeExchange(item.exchange)}|${String(item.raw_symbol || item.rawSymbol || "").trim().toUpperCase()}|${normalizeMarketType(item.market_type || item.marketType)}`;
    if (!key || overrideKeys.has(key)) continue;
    merged.market_overrides.push(item);
    overrideKeys.add(key);
  }

  return merged;
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
      market: toIdentity(exchange, symbol, overrideMatches[0].market_type, overrideMatches[0].canonical_symbol)
    };
  }
  if (overrideMatches.length > 1) {
    return {
      status: "ambiguous",
      confidence: 0.5,
      reason: "multiple explicit market overrides matched",
      candidates: overrideMatches.map((item) => toIdentity(exchange, symbol, item.market_type, item.canonical_symbol))
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
      assetClass: alias.assetClass || "unknown"
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

function toIdentity(exchange, symbol, marketType, canonicalSymbol) {
  const [base, quote] = splitCanonical(canonicalSymbol);
  const alias = resolveAssetAlias(base);
  return {
    exchange,
    marketType,
    rawSymbol: symbol,
    venueSymbol: normalizeVenueSymbol(exchange, symbol, marketType),
    canonicalSymbol,
    baseAsset: alias.canonical || base,
    quoteAsset: quote,
    assetClass: alias.assetClass || "unknown"
  };
}

function resolveAssetAlias(base) {
  const normalized = String(base || "").trim().toUpperCase();
  const matches = registry.asset_aliases.filter((item) => {
    if (String(item.canonical || "").trim().toUpperCase() === normalized) return true;
    return (item.aliases || []).some((alias) => String(alias || "").trim().toUpperCase() === normalized);
  });
  if (matches.length === 1) {
    return {
      canonical: matches[0].canonical,
      assetClass: matches[0].asset_class,
      ambiguous: false
    };
  }
  if (matches.length > 1) {
    return { canonical: "", assetClass: "", ambiguous: true };
  }
  return { canonical: "", assetClass: "", ambiguous: false };
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

export function registryStats() {
  const exchangeSet = new Set([
    ...Object.keys(registry.exchange_aliases || {}),
    ...(registry.market_overrides || []).map((item) => normalizeExchange(item.exchange))
  ]);
  const assetClasses = new Set((registry.asset_aliases || []).map((item) => item.asset_class).filter(Boolean));
  return {
    assets: registry.asset_aliases.length,
    overrides: registry.market_overrides.length,
    exchangeAliases: Object.keys(registry.exchange_aliases || {}).length,
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
