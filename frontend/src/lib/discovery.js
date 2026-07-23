import payload from "./mock-discovery.json";
import { normalizeExchange, normalizeMarketType, resolveIdentity } from "./identity.js";

export function loadDiscoveryEnvelope() {
  return payload;
}

export function normalizeDiscoveryEnvelope(input, source = "slipstream") {
  const items = extractDiscoveryItems(input);
  return {
    source,
    generatedAt:
      String(input?.generatedAt || input?.generated_at || input?.timestamp || new Date().toISOString()).trim(),
    items: items
      .map((item) => normalizeDiscoveryItem(item, source))
      .filter(Boolean)
  };
}

export function buildCandidateGroups(envelope = payload) {
  const items = Array.isArray(envelope?.items) ? envelope.items : [];
  const markets = items.map((item, index) => normalizeImportedMarket(item, index)).filter(Boolean);
  const grouped = new Map();

  for (const market of markets) {
    const key = `${market.baseAsset || "UNKNOWN"}/${market.quoteAsset || "UNKNOWN"}`;
    if (!grouped.has(key)) {
      grouped.set(key, []);
    }
    grouped.get(key).push(market);
  }

  return Array.from(grouped.entries())
    .map(([key, rows]) => summarizeGroup(key, rows))
    .sort((left, right) => left.groupKey.localeCompare(right.groupKey));
}

function normalizeImportedMarket(item, index) {
  if (!item || typeof item !== "object") return null;
  if (shouldIgnoreImportedMarket(item)) return null;

  const exchange = normalizeExchange(item.platformId || item.platform || item.exchange);
  const rawSymbol = String(item.symbol || item.rawSymbol || "").trim();
  const marketTypeHint = String(item.marketType || item.market_type || "").trim();
  const explicitBase = String(item.baseAsset || item.base_asset || "").trim().toUpperCase();
  const explicitQuote = String(item.quoteAsset || item.quote_asset || "").trim().toUpperCase();
  const explicitAssetClass = normalizeImportedAssetClassHints(
    item.assetClass,
    item.asset_class,
    item.assetClassHint,
    item.asset_class_hint,
    item.category,
    item.underlyingCategory,
    item.underlying_category,
    item.sector,
    item.tags
  );
  const resolution = resolveIdentity({
    exchange,
    symbol: rawSymbol,
    marketTypeHint,
    canonicalSymbolHint: explicitBase && explicitQuote ? `${explicitBase}/${explicitQuote}` : "",
    instType: String(item.instType || item.inst_type || "").trim(),
    productType: String(item.productType || item.product_type || "").trim()
  });

  const market = resolution.market || {};
  const baseAsset = explicitBase || market.baseAsset || "";
  const quoteAsset = explicitQuote || market.quoteAsset || "";
  let assetClass = explicitAssetClass || market.assetClass || "unknown";
  if (assetClass === "crypto" && isRwaAssetClass(market.assetClass)) {
    assetClass = market.assetClass;
  }
  const evidence = ["imported from slipstream market inventory"];
  if (explicitBase && explicitQuote) {
    evidence.push("used explicit base/quote from discovery source");
  }
  if (explicitAssetClass) {
    evidence.push("used exchange-provided asset classification");
  }
  if (resolution.reason) {
    evidence.push(resolution.reason);
  }
  const flags = normalizeMarketStatusFlags(item);

  return {
    id: String(item.id || `${exchange}:${rawSymbol}:${index}`),
    sourceId: String(item.sourceId || envelopeSource(item) || "slipstream"),
    exchange,
    platform: String(item.platform || item.platformId || exchange).trim(),
    venueType: String(item.venueType || item.venue_type || "").trim().toLowerCase(),
    marketType: normalizeMarketType(marketTypeHint) || market.marketType || "",
    rawSymbol,
    venueSymbol: market.venueSymbol || rawSymbol,
    baseAsset,
    quoteAsset,
    canonicalSymbol: baseAsset && quoteAsset ? `${baseAsset}/${quoteAsset}` : market.canonicalSymbol || "",
    assetClass,
    status: String(item.status || "").trim().toLowerCase(),
    st: flags.includes("st"),
    preDelisting: flags.includes("pre_delisting"),
    flags,
    chain: String(item.chain || "").trim(),
    externalUrl: String(item.externalUrl || item.external_url || "").trim(),
    confidence: Number(explicitAssetClass ? 0.95 : (resolution.confidence || (explicitBase && explicitQuote ? 0.9 : 0.7))),
    evidence: dedupeStrings(evidence),
    resolutionStatus: resolution.status || "unresolved"
  };
}

function normalizeDiscoveryItem(item, source) {
  if (!item || typeof item !== "object") return null;
  const flags = normalizeMarketStatusFlags(item);
  const normalized = {
    sourceId: String(item.sourceId || item.source_id || source || "slipstream").trim(),
    platformId: String(item.platformId || item.platform_id || item.exchange || item.platform || "").trim(),
    platform: String(item.platform || item.platformName || item.platformId || item.platform_id || "").trim(),
    venueType: String(item.venueType || item.venue_type || "").trim(),
    marketType: String(item.marketType || item.market_type || "").trim(),
    symbol: String(item.symbol || item.rawSymbol || item.raw_symbol || "").trim(),
    baseAsset: String(item.baseAsset || item.base_asset || "").trim(),
    quoteAsset: String(item.quoteAsset || item.quote_asset || "").trim(),
    assetClass: String(item.assetClass || item.asset_class || "").trim(),
    assetClassHint: String(item.assetClassHint || item.asset_class_hint || "").trim(),
    category: String(item.category || "").trim(),
    underlyingCategory: String(item.underlyingCategory || item.underlying_category || "").trim(),
    sector: String(item.sector || "").trim(),
    tags: Array.isArray(item.tags) ? item.tags.map((value) => String(value || "").trim()).filter(Boolean) : [],
    chain: String(item.chain || item.chainName || item.chain_name || "").trim(),
    status: String(item.status || "").trim(),
    st: flags.includes("st"),
    preDelisting: flags.includes("pre_delisting"),
    flags,
    externalUrl: String(item.externalUrl || item.external_url || "").trim(),
    firstSeenAt: String(item.firstSeenAt || item.first_seen_at || "").trim(),
    lastSeenAt: String(item.lastSeenAt || item.last_seen_at || "").trim()
  };
  return shouldIgnoreImportedMarket(normalized) ? null : normalized;
}

function extractDiscoveryItems(payload) {
  if (Array.isArray(payload)) return payload;
  if (!payload || typeof payload !== "object") return [];
  if (Array.isArray(payload.items)) return payload.items;
  if (Array.isArray(payload.data)) return payload.data;
  if (payload.data && Array.isArray(payload.data.items)) return payload.data.items;
  return [];
}

function summarizeGroup(groupKey, rows) {
  const exchanges = new Set();
  const marketTypes = new Set();
  const venueTypes = new Set();
  const flags = new Set();
  const evidence = [];
  let minConfidence = 1;
  let needsReview = false;

  for (const row of rows) {
    if (row.exchange) exchanges.add(row.exchange);
    if (row.marketType) marketTypes.add(row.marketType);
    if (row.venueType) venueTypes.add(row.venueType);
    for (const flag of row.flags || []) flags.add(flag);
    if (row.resolutionStatus !== "resolved" || row.assetClass === "unknown") needsReview = true;
    minConfidence = Math.min(minConfidence, Number(row.confidence || 0));
    evidence.push(...(row.evidence || []));
  }

  const [baseAsset, quoteAsset] = groupKey.split("/");
  return {
    groupKey,
    canonicalAsset: baseAsset || "",
    canonicalSymbol: groupKey,
    quoteAsset: quoteAsset || "",
    assetClass: rows.find((row) => row.assetClass && row.assetClass !== "unknown")?.assetClass || "unknown",
    exchanges: Array.from(exchanges).sort(),
    marketTypes: Array.from(marketTypes).sort(),
    venueTypes: Array.from(venueTypes).sort(),
    flags: Array.from(flags).sort(),
    needsReview,
    primaryConfidence: Number.isFinite(minConfidence) ? minConfidence : 0,
    evidence: dedupeStrings(evidence),
    markets: rows.slice().sort((left, right) => left.exchange.localeCompare(right.exchange))
  };
}

function dedupeStrings(values) {
  const seen = new Set();
  const out = [];
  for (const value of values) {
    const normalized = String(value || "").trim();
    if (!normalized || seen.has(normalized)) continue;
    seen.add(normalized);
    out.push(normalized);
  }
  return out;
}

function normalizeMarketStatusFlags(item) {
  const flags = [];
  const rawFlags = Array.isArray(item?.flags) ? item.flags : [];
  const rawStatusFlags = Array.isArray(item?.statusFlags)
    ? item.statusFlags
    : Array.isArray(item?.status_flags)
      ? item.status_flags
      : [];

  for (const flag of [...rawFlags, ...rawStatusFlags]) {
    const normalized = normalizeStatusFlag(flag);
    if (normalized) flags.push(normalized);
  }
  if (boolish(item?.st) || boolish(item?.isST) || boolish(item?.is_st)) flags.push("st");
  if (
    boolish(item?.preDelisting) ||
    boolish(item?.pre_delisting) ||
    boolish(item?.inDelisting) ||
    boolish(item?.in_delisting)
  ) {
    flags.push("pre_delisting");
  }
  return dedupeStrings(flags);
}

function normalizeStatusFlag(value) {
  const raw = String(value || "")
    .trim()
    .toLowerCase()
    .replaceAll("-", "_")
    .replaceAll(" ", "_")
    .replace(/_+/g, "_");
  if (!raw || raw === "false" || raw === "0" || raw === "no" || raw === "none") return "";
  if (["st", "special_treatment", "specialtreatment", "special"].includes(raw)) return "st";
  if (["pre_delisting", "predelisting", "pre_delist", "predelist", "in_delisting", "indelisting", "delisting"].includes(raw)) {
    return "pre_delisting";
  }
  return raw;
}

function boolish(value) {
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;
  const raw = String(value || "").trim().toLowerCase();
  return ["1", "true", "t", "yes", "y", "on", "enabled", "st", "special_treatment", "special treatment", "pre_delisting", "pre-delisting", "in_delisting"].includes(raw);
}

function envelopeSource(item) {
  return String(item.source || item.project || "").trim();
}

function normalizeImportedAssetClassHints(...hints) {
  const seen = new Set();
  for (const hint of hints) {
    if (Array.isArray(hint)) {
      for (const value of hint) {
        const assetClass = normalizeImportedAssetClassHint(value);
        if (assetClass) seen.add(assetClass);
      }
      continue;
    }
    const assetClass = normalizeImportedAssetClassHint(hint);
    if (assetClass) seen.add(assetClass);
  }
  for (const assetClass of ["fiat_stable", "rwa_stock", "rwa_commodity", "crypto"]) {
    if (seen.has(assetClass)) return assetClass;
  }
  return "";
}

function isRwaAssetClass(value) {
  return value === "rwa_stock" || value === "rwa_commodity";
}

function normalizeImportedAssetClassHint(value) {
  const raw = String(value || "")
    .trim()
    .toLowerCase()
    .replaceAll("_", " ")
    .replaceAll("-", " ")
    .replace(/\s+/g, " ");

  if (!raw) return "";
  if (raw.includes("stable")) return "fiat_stable";
  if (
    raw.includes("stock") ||
    raw.includes("equity") ||
    raw.includes("share") ||
    raw.includes("security") ||
    raw.includes("etf") ||
    raw.includes("index")
  ) {
    return "rwa_stock";
  }
  if (
    raw.includes("commodity") ||
    raw.includes("metal") ||
    raw.includes("gold") ||
    raw.includes("silver") ||
    raw.includes("oil") ||
    raw.includes("crude") ||
    raw.includes("gas") ||
    raw.includes("energy")
  ) {
    return "rwa_commodity";
  }
  if (
    raw.includes("crypto") ||
    raw.includes("blockchain") ||
    raw.includes("defi") ||
    raw.includes("meme") ||
    raw.includes("layer 1") ||
    raw.includes("layer1") ||
    raw.includes("token") ||
    raw.includes("coin")
  ) {
    return "crypto";
  }
  return "";
}

function shouldIgnoreImportedMarket(item) {
  const platformId = String(item?.platformId || item?.platform_id || item?.exchange || item?.platform || "").trim().toLowerCase();
  if (platformId !== "gate") return false;

  const baseAsset = String(item?.baseAsset || item?.base_asset || "").trim().toUpperCase();
  const rawSymbol = String(item?.symbol || item?.rawSymbol || item?.raw_symbol || "").trim().toUpperCase();
  return /[0-9]+[LS]$/.test(baseAsset) || /[0-9]+[LS]$/.test(rawSymbol);
}
