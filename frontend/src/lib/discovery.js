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

  const exchange = normalizeExchange(item.platformId || item.platform || item.exchange);
  const rawSymbol = String(item.symbol || item.rawSymbol || "").trim();
  const marketTypeHint = String(item.marketType || item.market_type || "").trim();
  const explicitBase = String(item.baseAsset || item.base_asset || "").trim().toUpperCase();
  const explicitQuote = String(item.quoteAsset || item.quote_asset || "").trim().toUpperCase();
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
  const evidence = ["imported from slipstream market inventory"];
  if (explicitBase && explicitQuote) {
    evidence.push("used explicit base/quote from discovery source");
  }
  if (resolution.reason) {
    evidence.push(resolution.reason);
  }

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
    assetClass: market.assetClass || "unknown",
    status: String(item.status || "").trim().toLowerCase(),
    chain: String(item.chain || "").trim(),
    externalUrl: String(item.externalUrl || item.external_url || "").trim(),
    confidence: Number(resolution.confidence || (explicitBase && explicitQuote ? 0.9 : 0.7)),
    evidence: dedupeStrings(evidence),
    resolutionStatus: resolution.status || "unresolved"
  };
}

function normalizeDiscoveryItem(item, source) {
  if (!item || typeof item !== "object") return null;
  return {
    sourceId: String(item.sourceId || item.source_id || source || "slipstream").trim(),
    platformId: String(item.platformId || item.platform_id || item.exchange || item.platform || "").trim(),
    platform: String(item.platform || item.platformName || item.platformId || item.platform_id || "").trim(),
    venueType: String(item.venueType || item.venue_type || "").trim(),
    marketType: String(item.marketType || item.market_type || "").trim(),
    symbol: String(item.symbol || item.rawSymbol || item.raw_symbol || "").trim(),
    baseAsset: String(item.baseAsset || item.base_asset || "").trim(),
    quoteAsset: String(item.quoteAsset || item.quote_asset || "").trim(),
    chain: String(item.chain || item.chainName || item.chain_name || "").trim(),
    status: String(item.status || "").trim(),
    externalUrl: String(item.externalUrl || item.external_url || "").trim(),
    firstSeenAt: String(item.firstSeenAt || item.first_seen_at || "").trim(),
    lastSeenAt: String(item.lastSeenAt || item.last_seen_at || "").trim()
  };
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
  const evidence = [];
  let minConfidence = 1;
  let needsReview = false;

  for (const row of rows) {
    if (row.exchange) exchanges.add(row.exchange);
    if (row.marketType) marketTypes.add(row.marketType);
    if (row.venueType) venueTypes.add(row.venueType);
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

function envelopeSource(item) {
  return String(item.source || item.project || "").trim();
}
