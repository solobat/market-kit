<script>
  import { onMount } from "svelte";
  import { buildCandidateGroups, loadDiscoveryEnvelope, normalizeDiscoveryEnvelope } from "./lib/discovery.js";
  import { loadRegistry, normalizeExchange, normalizeImportedCases, normalizeMarketType, registryStats, resolveIdentity, setRuntimeRegistry } from "./lib/identity.js";

  let registry = loadRegistry();
  let stats = registryStats(registry);
  const allDiscoverySourceId = "all";
  const defaultDiscoveryEnvelope = loadDiscoveryEnvelope();
  const syncConfigKey = "market-kit.sync-config";
  const syncCasesKey = "market-kit.sync-cases";

  let theme = "dark";
  let assetQuery = "";
  let assetClassFilter = "all";
  let overrideQuery = "";
  let groupQuery = "";
  let syncQuery = "";
  let page = "registry";
  let request = {
    exchange: "okx",
    symbol: "DRAM-USDT-SWAP",
    marketTypeHint: "",
    canonicalSymbolHint: "",
    instType: "",
    productType: ""
  };
  let syncConfig = {
    source: "veridex",
    url: "",
    authHeader: "",
    authValue: ""
  };
  let syncedCases = [];
  let syncState = "idle";
  let syncMessage = "";
  let registryState = "embedded";
  let registryMessage = "正在使用前端内置 registry。";
  let selectedAssetCanonical = "";
  let proxyAvailable = false;
  let remoteSources = [];
  let selectedSourceId = "";
  let discoveryEnvelope = defaultDiscoveryEnvelope;
  let discoveryState = "idle";
  let discoveryMessage = "";
  let selectedDiscoverySourceId = "";
  let groupStatusFilter = "all";
  let groupAssetClassFilter = "all";
  let groupMarketTypeFilter = "all";
  let groupExchangeFilter = "all";
  let expandedGroups = {};
  let symbolQuery = "";
  let symbolPlatformFilter = ["binance"];
  let symbolMarketTypeFilter = ["perpetual"];
  let symbolAssetClassFilter = ["rwa_stock"];
  let symbolLayerFilter = ["registry", "discovery"];
  let symbolPresenceEnabled = true;
  let symbolPresencePlatformFilter = ["binance-web3"];
  let symbolPresenceMarketTypeFilter = ["spot"];
  let symbolPresenceAssetClassFilter = ["rwa_stock"];
  let symbolOutputField = "rawSymbol";
  let symbolOutputFormat = "quotedCsv";
  let selectedSymbolIds = new Set();
  let symbolCopyMessage = "";
  $: stats = registryStats(registry);
  $: discoverySources = remoteSources.filter((item) => sourceKind(item) === "discovery");
  $: sampleSources = remoteSources.filter((item) => sourceKind(item) !== "discovery");

  $: assetClassOptions = Array.from(new Set((registry.asset_aliases || []).map((item) => item.asset_class).filter(Boolean))).sort();
  $: assetRows = (registry.asset_aliases || [])
    .filter((item) => {
      const query = assetQuery.trim().toUpperCase();
      const matchesQuery = !query || (
        String(item.canonical || "").toUpperCase().includes(query) ||
        String(item.asset_class || "").toUpperCase().includes(query) ||
        (item.aliases || []).some((alias) => String(alias || "").toUpperCase().includes(query)) ||
        (item.unit_aliases || []).some((alias) => String(alias?.alias || "").toUpperCase().includes(query))
      );
      const matchesClass = assetClassFilter === "all" || item.asset_class === assetClassFilter;
      return matchesQuery && matchesClass;
    })
    .sort((left, right) => {
      const query = assetQuery.trim().toUpperCase();
      if (query) {
        const rankCompare = assetSearchRank(left, query) - assetSearchRank(right, query);
        if (rankCompare !== 0) return rankCompare;
      }
      const classCompare = String(left.asset_class || "").localeCompare(String(right.asset_class || ""));
      if (classCompare !== 0) return classCompare;
      const aliasCompare = totalAssetAliasCount(right) - totalAssetAliasCount(left);
      if (aliasCompare !== 0) return aliasCompare;
      return String(left.canonical || "").localeCompare(String(right.canonical || ""));
    });
  $: visibleAliasCount = assetRows.reduce((sum, item) => sum + totalAssetAliasCount(item), 0);
  $: registryAssetRows = registry.asset_aliases || [];
  $: effectiveSelectedAssetCanonical =
    selectedAssetCanonical ||
    assetRows[0]?.canonical ||
    registryAssetRows[0]?.canonical ||
    "";
  $: selectedAsset = registryAssetRows.find((item) => item.canonical === effectiveSelectedAssetCanonical) || null;

  $: overrideRows = (registry.market_overrides || []).filter((item) => {
    const query = overrideQuery.trim().toUpperCase();
    if (!query) return true;
    return (
      String(item.exchange || "").toUpperCase().includes(query) ||
      String(item.raw_symbol || "").toUpperCase().includes(query) ||
      String(item.canonical_symbol || "").toUpperCase().includes(query) ||
      String(item.market_type || "").toUpperCase().includes(query)
    );
  });
  $: allCandidateGroups = buildCandidateGroups(discoveryEnvelope);
  $: groupAssetClassOptions = Array.from(new Set(allCandidateGroups.map((group) => group.assetClass).filter(Boolean))).sort();
  $: groupMarketTypeOptions = Array.from(
    new Set(allCandidateGroups.flatMap((group) => group.marketTypes || []).filter(Boolean))
  ).sort();
  $: groupExchangeOptions = Array.from(
    new Set(allCandidateGroups.flatMap((group) => group.exchanges || []).filter(Boolean))
  ).sort();
  $: candidateGroups = allCandidateGroups.filter((group) => {
    const query = groupQuery.trim().toUpperCase();
    const matchesQuery = !query || (
      String(group.groupKey || "").toUpperCase().includes(query) ||
      String(group.assetClass || "").toUpperCase().includes(query) ||
      (group.exchanges || []).some((value) => String(value || "").toUpperCase().includes(query)) ||
      (group.markets || []).some((market) =>
        [market.rawSymbol, market.platform, market.marketType, market.venueType]
          .some((value) => String(value || "").toUpperCase().includes(query))
      )
    );
    const matchesStatus =
      groupStatusFilter === "all" ||
      (groupStatusFilter === "review" && group.needsReview) ||
      (groupStatusFilter === "ready" && !group.needsReview);
    const matchesAssetClass = groupAssetClassFilter === "all" || group.assetClass === groupAssetClassFilter;
    const matchesMarketType =
      groupMarketTypeFilter === "all" || (group.marketTypes || []).includes(groupMarketTypeFilter);
    const matchesExchange =
      groupExchangeFilter === "all" || (group.exchanges || []).includes(groupExchangeFilter);
    return matchesQuery && matchesStatus && matchesAssetClass && matchesMarketType && matchesExchange;
  });

  $: resolution = resolveIdentity(request);
  $: syncRows = syncedCases.filter((item) => {
    const query = syncQuery.trim().toUpperCase();
    if (!query) return true;
    return (
      String(item.source || "").toUpperCase().includes(query) ||
      String(item.request.exchange || "").toUpperCase().includes(query) ||
      String(item.request.symbol || "").toUpperCase().includes(query) ||
      String(item.request.marketTypeHint || "").toUpperCase().includes(query) ||
      String(item.reason || "").toUpperCase().includes(query) ||
      String(item.status || "").toUpperCase().includes(query) ||
      String(item.resolution?.market?.canonicalSymbol || "").toUpperCase().includes(query)
    );
  });
  $: unresolvedCount = syncedCases.filter((item) => item.status === "unresolved").length;
  $: ambiguousCount = syncedCases.filter((item) => item.status === "ambiguous").length;
  $: discoveryMarketCount = Array.isArray(discoveryEnvelope?.items) ? discoveryEnvelope.items.length : 0;
  $: reviewGroupCount = candidateGroups.filter((group) => group.needsReview).length;
  $: readyGroupCount = candidateGroups.filter((group) => !group.needsReview).length;
  $: selectedAssetOverrideRows = selectedAsset
    ? (registry.market_overrides || [])
      .filter((item) => String(item.canonical_symbol || "").toUpperCase().startsWith(`${selectedAsset.canonical}/`))
      .sort((left, right) => {
        if (left.exchange === right.exchange) {
          if (left.market_type === right.market_type) {
            return String(left.raw_symbol || "").localeCompare(String(right.raw_symbol || ""));
          }
          return String(left.market_type || "").localeCompare(String(right.market_type || ""));
        }
        return String(left.exchange || "").localeCompare(String(right.exchange || ""));
      })
    : [];
  $: selectedAssetGroups = selectedAsset
    ? allCandidateGroups
      .filter((group) => group.canonicalAsset === selectedAsset.canonical)
      .sort((left, right) => String(left.quoteAsset || "").localeCompare(String(right.quoteAsset || "")))
    : [];
  $: selectedAssetMarkets = selectedAssetGroups.flatMap((group) => group.markets || []);
  $: selectedAssetQuotes = Array.from(new Set(selectedAssetOverrideRows.map((item) => String(item.canonical_symbol || "").split("/")[1]).filter(Boolean))).sort();
  $: selectedAssetOverrideExchanges = Array.from(new Set(selectedAssetOverrideRows.map((item) => item.exchange).filter(Boolean))).sort();
  $: selectedAssetDiscoveryExchanges = Array.from(new Set(selectedAssetMarkets.map((item) => item.exchange).filter(Boolean))).sort();
  $: selectedAssetDiscoveryMarketTypes = Array.from(new Set(selectedAssetMarkets.map((item) => item.marketType).filter(Boolean))).sort();
  $: selectedAssetDiscoveryLoading = selectedAsset && (discoveryState === "loading" || discoveryEnvelope === defaultDiscoveryEnvelope);
  $: symbolRowsAll = buildSymbolRows(registry, allCandidateGroups);
  $: symbolPlatformOptions = uniqueSymbolOptions(symbolRowsAll.map((item) => item.platform));
  $: symbolMarketTypeOptions = uniqueSymbolOptions(symbolRowsAll.map((item) => item.marketType));
  $: symbolAssetClassOptions = uniqueSymbolOptions(symbolRowsAll.map((item) => item.assetClass));
  $: symbolLayerOptions = uniqueSymbolOptions(symbolRowsAll.map((item) => item.layer));
  $: symbolPresenceRows = symbolRowsAll.filter((item) =>
    matchesSymbolFilters(item, {
      platforms: symbolPresencePlatformFilter,
      marketTypes: symbolPresenceMarketTypeFilter,
      assetClasses: symbolPresenceAssetClassFilter,
      layers: []
    })
  );
  $: symbolPresenceBases = new Set(symbolPresenceRows.map((item) => item.baseAsset).filter(Boolean));
  $: symbolRows = symbolRowsAll
    .filter((item) =>
      matchesSymbolFilters(item, {
        platforms: symbolPlatformFilter,
        marketTypes: symbolMarketTypeFilter,
        assetClasses: symbolAssetClassFilter,
        layers: symbolLayerFilter
      })
    )
    .filter((item) => !symbolPresenceEnabled || symbolPresenceBases.has(item.baseAsset))
    .filter((item) => {
      const query = symbolQuery.trim().toUpperCase();
      if (!query) return true;
      return [
        item.rawSymbol,
        item.canonicalSymbol,
        item.baseAsset,
        item.quoteAsset,
        item.platform,
        item.marketType,
        item.assetClass,
        item.layer
      ].some((value) => String(value || "").toUpperCase().includes(query));
    })
    .sort((left, right) => {
      const baseCompare = String(left.baseAsset || "").localeCompare(String(right.baseAsset || ""));
      if (baseCompare !== 0) return baseCompare;
      const platformCompare = String(left.platform || "").localeCompare(String(right.platform || ""));
      if (platformCompare !== 0) return platformCompare;
      return String(left.rawSymbol || "").localeCompare(String(right.rawSymbol || ""));
    });
  $: {
    const visibleIds = new Set(symbolRows.map((item) => item.id));
    const nextSelected = Array.from(selectedSymbolIds).filter((id) => visibleIds.has(id));
    if (nextSelected.length !== selectedSymbolIds.size) {
      selectedSymbolIds = new Set(nextSelected);
    }
  }
  $: selectedSymbolRows = symbolRows.filter((item) => selectedSymbolIds.has(item.id));
  $: generatedSymbolText = buildGeneratedSymbolText(selectedSymbolRows, symbolOutputField, symbolOutputFormat);
  $: symbolSelectedCount = selectedSymbolIds.size;
  $: symbolBaseCount = new Set(symbolRows.map((item) => item.baseAsset).filter(Boolean)).size;
  $: symbolPresenceBaseCount = symbolPresenceBases.size;

  function applyTheme(next) {
    theme = next;
    document.documentElement.dataset.theme = next;
  }

  function toggleTheme() {
    applyTheme(theme === "dark" ? "light" : "dark");
  }

  function statusLabel(status) {
    if (status === "resolved") return "已解析";
    if (status === "ambiguous") return "有歧义";
    return "未解析";
  }

  function assetClassLabel(value) {
    if (value === "crypto") return "Crypto";
    if (value === "rwa_stock") return "RWA Stock";
    if (value === "fiat_stable") return "Stablecoin";
    return value || "Unknown";
  }

  function totalAssetAliasCount(asset) {
    return (asset?.aliases?.length || 0) + (asset?.unit_aliases?.length || 0);
  }

  function assetSearchRank(asset, query) {
    const canonical = String(asset?.canonical || "").trim().toUpperCase();
    const aliases = [
      ...(asset?.aliases || []),
      ...(asset?.unit_aliases || []).map((alias) => alias?.alias)
    ].map((value) => String(value || "").trim().toUpperCase()).filter(Boolean);
    const assetClass = String(asset?.asset_class || "").trim().toUpperCase();

    if (canonical === query) return 0;
    if (aliases.some((alias) => alias === query)) return 1;
    if (canonical.startsWith(query)) return 2;
    if (aliases.some((alias) => alias.startsWith(query))) return 3;
    if (canonical.includes(query)) return 4;
    if (aliases.some((alias) => alias.includes(query))) return 5;
    if (assetClass.includes(query)) return 6;
    return 7;
  }

  function formatUnitMultiplier(value) {
    const numeric = Number(value || 0);
    if (!Number.isFinite(numeric) || numeric <= 0) return "";
    return Number.isInteger(numeric) ? numeric.toLocaleString("en-US") : numeric.toLocaleString("en-US", { maximumFractionDigits: 6 });
  }

  function selectAsset(canonical) {
    selectedAssetCanonical = String(canonical || "").trim().toUpperCase();
  }

  function openAssetDetail(canonical) {
    selectAsset(canonical);
    navigate("asset-detail", { asset: canonical });
  }

  function handleAssetCardKeydown(event, canonical) {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openAssetDetail(canonical);
    }
  }

  function backToRegistry() {
    navigate("registry");
  }

  function openSelectedAssetGroups() {
    if (!selectedAsset?.canonical) return;
    groupQuery = selectedAsset.canonical;
    navigate("groups");
  }

  function routeFromPath(pathname) {
    const segments = String(pathname || "/")
      .split("/")
      .map((part) => part.trim())
      .filter(Boolean)
      .map((part) => decodeURIComponent(part));
    const [section, asset] = segments;

    if (section === "rules") return { page: "rules" };
    if (section === "groups") return { page: "groups" };
    if (section === "symbols") return { page: "symbols" };
    if (section === "samples") return { page: "samples" };
    if (section === "playground") return { page: "playground" };
    if (section === "registry" && asset) return { page: "asset-detail", asset };
    return { page: "registry" };
  }

  function pathForPage(nextPage, options = {}) {
    if (nextPage === "asset-detail") {
      const asset = String(options.asset || selectedAsset?.canonical || selectedAssetCanonical || "").trim().toUpperCase();
      return asset ? `/registry/${encodeURIComponent(asset)}` : "/registry";
    }
    if (nextPage === "rules") return "/rules";
    if (nextPage === "groups") return "/groups";
    if (nextPage === "symbols") return "/symbols";
    if (nextPage === "samples") return "/samples";
    if (nextPage === "playground") return "/playground";
    return "/registry";
  }

  function applyRoute(route) {
    if (route.asset) {
      selectAsset(route.asset);
    }
    page = route.page || "registry";
  }

  function syncRouteFromLocation() {
    if (typeof window === "undefined") return;
    const route = routeFromPath(window.location.pathname);
    applyRoute(route);
    if (window.location.pathname === "/" || window.location.pathname === "") {
      navigate(route.page, { asset: route.asset }, true);
    }
  }

  function navigate(nextPage, options = {}, replace = false) {
    if (nextPage === "asset-detail" && options.asset) {
      selectAsset(options.asset);
    }
    page = nextPage || "registry";
    if (typeof window === "undefined") return;

    const nextPath = pathForPage(nextPage, options);
    if (window.location.pathname === nextPath) return;
    const method = replace ? "replaceState" : "pushState";
    window.history[method]({}, "", nextPath);
  }

  function pageLabel(value) {
    if (value === "asset-detail") return selectedAsset?.canonical ? `${selectedAsset.canonical} 详情` : "标的详情";
    if (value === "groups") return "候选分组";
    if (value === "symbols") return "Symbol 生成器";
    if (value === "samples") return "待补样本";
    if (value === "playground") return "解析试验台";
    if (value === "rules") return "规则检视";
    return "Registry";
  }

  function pageDescription(value) {
    if (value === "asset-detail") return "把单个 canonical asset 的 alias、override、discovery presence 放到同一个运营视图里。";
    if (value === "groups") return "从发现市场清单里观察跨平台候选组，找出可以自动归并和需要人工复核的资产。";
    if (value === "symbols") return "按平台、市场类型和资产类别筛选 symbol，生成可交给下游项目的精确列表。";
    if (value === "samples") return "同步 downstream 项目里 unresolved / ambiguous 的样本，反向补齐 market-kit 规则。";
    if (value === "playground") return "手动输入交易所和 raw symbol，快速验证 resolver 当前会返回什么市场身份。";
    if (value === "rules") return "查看显式 market override，确认 raw venue symbol 如何折到 canonical symbol。";
    return "维护 shared registry、资产别名和交易所身份规则，让下游项目复用同一套解析判断。";
  }

  function sourceKind(source) {
    const explicit = String(source?.kind || "").trim().toLowerCase();
    if (explicit) return explicit;

    const project = String(source?.project || source?.id || "").trim().toLowerCase();
    if (project.includes("slipstream") || project.includes("bootstrap") || project.includes("discovery")) {
      return "discovery";
    }
    return "sample";
  }

  function sourceBadge(source) {
    return source?.project || source?.kind || source?.id || "source";
  }

  function preferredDiscoverySourceId(sources) {
    const rows = Array.isArray(sources) ? sources : [];
    if (rows.length > 1) return allDiscoverySourceId;
    const builtIn = rows.find((item) => String(item.id || "").trim() === "market-kit-bootstrap");
    if (builtIn?.id) return builtIn.id;
    return rows[0]?.id || "";
  }

  function shouldRefreshDiscoveryEnvelope(preferredDiscoveryId) {
    if (!preferredDiscoveryId) return false;
    const currentSource = String(discoveryEnvelope?.source || "").trim();
    if (discoveryEnvelope === defaultDiscoveryEnvelope) return true;
    if (preferredDiscoveryId === allDiscoverySourceId && currentSource !== "market-kit-all") return true;
    return false;
  }

  function formatTime(value) {
    if (!value) return "未记录";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString("zh-CN", { hour12: false });
  }

  function syncStatusLabel(value) {
    if (value === "ambiguous") return "有歧义";
    if (value === "unresolved") return "未解析";
    if (value === "resolved") return "已解析";
    return "待处理";
  }

  function toggleGroupExpanded(groupKey) {
    expandedGroups = { ...expandedGroups, [groupKey]: !expandedGroups[groupKey] };
  }

  function collapseAllGroups() {
    expandedGroups = {};
  }

  function expandVisibleGroups() {
    expandedGroups = Object.fromEntries(candidateGroups.map((group) => [group.groupKey, true]));
  }

  function resetGroupFilters() {
    groupQuery = "";
    groupStatusFilter = "all";
    groupAssetClassFilter = "all";
    groupMarketTypeFilter = "all";
    groupExchangeFilter = "all";
  }

  function buildSymbolRows(registryInput, groups) {
    const assetClassIndex = new Map(
      (registryInput.asset_aliases || []).map((item) => [
        String(item.canonical || "").trim().toUpperCase(),
        String(item.asset_class || "unknown").trim() || "unknown"
      ])
    );
    const rows = [];
    const seen = new Set();

    for (const item of registryInput.market_overrides || []) {
      const canonicalSymbol = String(item.canonical_symbol || "").trim().toUpperCase();
      const [baseAsset = "", quoteAsset = ""] = canonicalSymbol.split("/");
      const row = {
        id: `registry:${item.exchange}:${item.market_type}:${item.raw_symbol}`,
        layer: "registry",
        platform: normalizeExchange(item.exchange),
        platformLabel: platformLabel(normalizeExchange(item.exchange)),
        venueType: "registry",
        marketType: normalizeMarketType(item.market_type) || String(item.market_type || "").trim().toLowerCase(),
        rawSymbol: String(item.raw_symbol || "").trim(),
        canonicalSymbol,
        baseAsset,
        quoteAsset,
        assetClass: assetClassIndex.get(baseAsset) || "unknown",
        chain: "",
        status: "",
        sourceId: "registry"
      };
      appendUniqueSymbolRow(rows, seen, row);
    }

    for (const group of groups || []) {
      for (const market of group.markets || []) {
        const baseAsset = String(market.baseAsset || group.canonicalAsset || "").trim().toUpperCase();
        const quoteAsset = String(market.quoteAsset || group.quoteAsset || "").trim().toUpperCase();
        const platform = normalizeExchange(market.exchange || market.platform);
        const row = {
          id: `discovery:${platform}:${market.marketType}:${market.rawSymbol}:${baseAsset}:${quoteAsset}:${market.chain || ""}`,
          layer: "discovery",
          platform,
          platformLabel: platformLabel(platform),
          venueType: String(market.venueType || "").trim().toLowerCase(),
          marketType: normalizeMarketType(market.marketType) || String(market.marketType || "").trim().toLowerCase(),
          rawSymbol: String(market.rawSymbol || "").trim(),
          canonicalSymbol: String(market.canonicalSymbol || (baseAsset && quoteAsset ? `${baseAsset}/${quoteAsset}` : "")).trim().toUpperCase(),
          baseAsset,
          quoteAsset,
          assetClass: String(market.assetClass || group.assetClass || assetClassIndex.get(baseAsset) || "unknown").trim(),
          chain: String(market.chain || "").trim(),
          status: String(market.status || "").trim().toLowerCase(),
          sourceId: String(market.sourceId || "").trim()
        };
        appendUniqueSymbolRow(rows, seen, row);
      }
    }

    return rows;
  }

  function appendUniqueSymbolRow(rows, seen, row) {
    if (!row.rawSymbol || !row.baseAsset) return;
    const key = [
      row.layer,
      row.platform,
      row.marketType,
      row.rawSymbol.toUpperCase(),
      row.baseAsset,
      row.quoteAsset,
      row.chain
    ].join("|");
    if (seen.has(key)) return;
    seen.add(key);
    rows.push({ ...row, id: key });
  }

  function uniqueSymbolOptions(values) {
    return Array.from(new Set(values.map((value) => String(value || "").trim()).filter(Boolean))).sort();
  }

  function matchesSymbolFilters(item, filters) {
    return (
      matchesFilterValue(item.platform, filters.platforms) &&
      matchesFilterValue(item.marketType, filters.marketTypes) &&
      matchesFilterValue(item.assetClass, filters.assetClasses) &&
      matchesFilterValue(item.layer, filters.layers)
    );
  }

  function matchesFilterValue(value, selected) {
    if (!selected?.length) return true;
    return selected.includes(String(value || "").trim());
  }

  function toggleArrayValue(values, value) {
    const normalized = String(value || "").trim();
    if (!normalized) return values;
    return values.includes(normalized)
      ? values.filter((item) => item !== normalized)
      : [...values, normalized];
  }

  function platformLabel(value) {
    if (value === "binance") return "Binance";
    if (value === "binance-web3") return "Binance Web3 / Ondo";
    if (value === "hyperliquid") return "Hyperliquid";
    if (value === "bitget") return "Bitget";
    if (value === "bybit") return "Bybit";
    if (value === "okx") return "OKX";
    if (value === "gate") return "Gate";
    return value || "unknown";
  }

  function layerLabel(value) {
    if (value === "registry") return "Registry";
    if (value === "discovery") return "Discovery";
    return value || "Unknown";
  }

  function toggleSymbolRow(id) {
    const next = new Set(selectedSymbolIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    selectedSymbolIds = next;
  }

  function selectAllSymbolRows() {
    selectedSymbolIds = new Set(symbolRows.map((item) => item.id));
  }

  function deselectAllSymbolRows() {
    selectedSymbolIds = new Set();
  }

  function invertSymbolSelection() {
    const next = new Set();
    for (const item of symbolRows) {
      if (!selectedSymbolIds.has(item.id)) {
        next.add(item.id);
      }
    }
    selectedSymbolIds = next;
  }

  function applyStockPerpOndoPreset() {
    symbolQuery = "";
    symbolPlatformFilter = ["binance"];
    symbolMarketTypeFilter = ["perpetual"];
    symbolAssetClassFilter = ["rwa_stock"];
    symbolLayerFilter = ["registry", "discovery"];
    symbolPresenceEnabled = true;
    symbolPresencePlatformFilter = ["binance-web3"];
    symbolPresenceMarketTypeFilter = ["spot"];
    symbolPresenceAssetClassFilter = ["rwa_stock"];
    symbolOutputField = "rawSymbol";
    symbolOutputFormat = "quotedCsv";
    selectedSymbolIds = new Set();
    symbolCopyMessage = "";
  }

  function resetSymbolFilters() {
    symbolQuery = "";
    symbolPlatformFilter = [];
    symbolMarketTypeFilter = [];
    symbolAssetClassFilter = [];
    symbolLayerFilter = ["registry", "discovery"];
    symbolPresenceEnabled = false;
    symbolPresencePlatformFilter = [];
    symbolPresenceMarketTypeFilter = [];
    symbolPresenceAssetClassFilter = [];
    selectedSymbolIds = new Set();
    symbolCopyMessage = "";
  }

  function symbolOutputValue(row, field) {
    if (field === "baseAsset") return row.baseAsset;
    if (field === "canonicalSymbol") return row.canonicalSymbol;
    return row.rawSymbol;
  }

  function buildGeneratedSymbolText(rows, field, format) {
    const values = Array.from(new Set(rows.map((row) => symbolOutputValue(row, field)).filter(Boolean))).sort();
    if (format === "json") return JSON.stringify(values, null, 2);
    if (format === "lines") return values.join("\n");
    if (format === "csv") return values.join(",");
    return values.map((value) => `"${value}"`).join(",");
  }

  async function copyGeneratedSymbolText() {
    if (!generatedSymbolText) {
      symbolCopyMessage = "先选择至少一个 symbol。";
      return;
    }
    try {
      await navigator.clipboard.writeText(generatedSymbolText);
      symbolCopyMessage = `已复制 ${selectedSymbolRows.length} 项。`;
    } catch {
      symbolCopyMessage = "复制失败，请手动选中文本。";
    }
  }

  function persistSyncState() {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(syncConfigKey, JSON.stringify(syncConfig));
    window.localStorage.setItem(syncCasesKey, JSON.stringify(syncedCases));
  }

  async function fetchFromEndpoints(endpoints) {
    let lastError = null;
    for (const endpoint of endpoints) {
      try {
        const response = await fetch(endpoint);
        const payload = await response.json();
        if (!response.ok) {
          throw new Error(payload?.error || `endpoint returned ${response.status}`);
        }
        return payload;
      } catch (error) {
        lastError = error;
      }
    }
    throw lastError || new Error("all endpoints failed");
  }

  async function loadRuntimeRegistry() {
    registryState = "loading";
    registryMessage = "正在读取后端 runtime registry…";
    try {
      const payload = await fetchFromEndpoints(["/api/v1/registry", "/api/registry"]);
      const nextRegistry = payload.registry || payload;
      registry = setRuntimeRegistry(nextRegistry);
      registryState = "success";
      registryMessage = `runtime · ${registry.market_overrides.length} overrides`;
      request = { ...request };
      discoveryEnvelope = { ...discoveryEnvelope };
    } catch (error) {
      registryState = "embedded";
      registryMessage = error instanceof Error
        ? `后端 registry 读取失败，继续使用前端内置 registry：${error.message}`
        : "后端 registry 读取失败，继续使用前端内置 registry。";
    }
  }

  function mergeCases(nextRows) {
    const map = new Map();
    for (const item of [...syncedCases, ...nextRows]) {
      map.set(`${item.source}:${item.id}`, item);
    }
    syncedCases = Array.from(map.values()).sort((left, right) => {
      const l = new Date(left.lastSeenAt || left.firstSeenAt || 0).getTime();
      const r = new Date(right.lastSeenAt || right.firstSeenAt || 0).getTime();
      return r - l;
    });
  }

  async function syncRemoteCases() {
    if (!syncConfig.url.trim()) {
      syncState = "error";
      syncMessage = "请先填写远端导出地址。";
      return;
    }

    syncState = "loading";
    syncMessage = "正在拉取远端待补规则样本…";

    try {
      const headers = {};
      if (syncConfig.authHeader.trim() && syncConfig.authValue.trim()) {
        headers[syncConfig.authHeader.trim()] = syncConfig.authValue.trim();
      }
      const response = await fetch(syncConfig.url.trim(), { headers });
      if (!response.ok) {
        throw new Error(`remote responded ${response.status}`);
      }
      const payload = await response.json();
      syncedCases = normalizeImportedCases(payload).map((item) => ({
        ...item,
        source: item.source === "unknown" ? syncConfig.source : item.source
      }));
      syncState = "success";
      syncMessage = `已同步 ${syncedCases.length} 条样本。`;
      persistSyncState();
    } catch (error) {
      syncState = "error";
      syncMessage =
        error instanceof Error
          ? `同步失败：${error.message}。如果远端在 ECS 上，请确认接口允许浏览器跨域访问。`
          : "同步失败，请检查远端导出接口。";
    }
  }

  async function loadRemoteSources(options = {}) {
    const { shouldAutoBootstrap = false } = options;
    try {
      const payload = await fetchFromEndpoints(["/api/discovery/sources", "/__market-kit/sources"]);
      remoteSources = Array.isArray(payload.sources) ? payload.sources : [];
      proxyAvailable = true;
      const nextSampleSources = remoteSources.filter((item) => sourceKind(item) !== "discovery");
      const nextDiscoverySources = remoteSources.filter((item) => sourceKind(item) === "discovery");
      if (!selectedSourceId && nextSampleSources.length) {
        selectedSourceId = nextSampleSources[0].id;
      }
      const preferredDiscoveryId = preferredDiscoverySourceId(nextDiscoverySources);
      if (!selectedDiscoverySourceId && preferredDiscoveryId) {
        selectedDiscoverySourceId = preferredDiscoveryId;
      }
      if (preferredDiscoveryId && (shouldAutoBootstrap || shouldRefreshDiscoveryEnvelope(preferredDiscoveryId))) {
        await loadCurrentDiscovery({
          loadingMessage: "正在读取后端发现市场快照…",
          fallbackErrorMessage: "读取后端发现市场快照失败。"
        });
      }
    } catch {
      proxyAvailable = false;
      remoteSources = [];
    }
  }

  async function loadCurrentDiscovery(options = {}) {
    const {
      loadingMessage = "正在读取后端发现市场快照…",
      fallbackErrorMessage = "读取后端发现市场快照失败。"
    } = options;

    discoveryState = "loading";
    discoveryMessage = loadingMessage;

    try {
      const payload = await fetchFromEndpoints([
        `/api/discovery/current?source=${encodeURIComponent(allDiscoverySourceId)}`,
        `/api/discovery/sync?source=${encodeURIComponent(allDiscoverySourceId)}`
      ]);
      const project = payload.payload?.source || payload.source?.project || payload.source?.id || "market-discovery";
      discoveryEnvelope = normalizeDiscoveryEnvelope(payload.payload, project);
      discoveryState = "success";
      discoveryMessage = `已读取后端发现市场快照，共 ${discoveryEnvelope.items.length} 个市场。`;
    } catch (error) {
      discoveryState = "error";
      discoveryMessage = error instanceof Error ? `读取失败：${error.message}` : fallbackErrorMessage;
    }
  }

  async function syncDiscoverySource(sourceId = selectedDiscoverySourceId, options = {}) {
    const {
      loadingMessage = "正在同步发现市场清单…",
      fallbackErrorMessage = "同步发现源失败。"
    } = options;
    if (!sourceId) {
      discoveryState = "error";
      discoveryMessage = "请先选择一个发现源。";
      return;
    }

    discoveryState = "loading";
    discoveryMessage = loadingMessage;

    try {
      const payload = await fetchFromEndpoints([
        `/api/discovery/sync?source=${encodeURIComponent(sourceId)}`,
        `/__market-kit/sync?source=${encodeURIComponent(sourceId)}`
      ]);
      const project = payload.payload?.source || payload.source?.project || payload.source?.id || "market-discovery";
      discoveryEnvelope = normalizeDiscoveryEnvelope(payload.payload, project);
      discoveryState = "success";
      discoveryMessage = `已从 ${payload.source?.label || sourceId} 导入 ${discoveryEnvelope.items.length} 个市场。`;
    } catch (error) {
      discoveryState = "error";
      discoveryMessage = error instanceof Error ? `同步失败：${error.message}` : fallbackErrorMessage;
    }
  }

  function resetDiscoveryToMock() {
    discoveryEnvelope = defaultDiscoveryEnvelope;
    discoveryState = "success";
    discoveryMessage = "已切回仓库内置的本地示例市场清单。";
  }

  async function syncPresetSource(sourceId = selectedSourceId) {
    if (!sourceId) {
      syncState = "error";
      syncMessage = "请先选择一个远端同步源。";
      return;
    }

    syncState = "loading";
    syncMessage = "正在通过本地代理拉取远端样本…";

    try {
      const payload = await fetchFromEndpoints([
        `/api/discovery/sync?source=${encodeURIComponent(sourceId)}`,
        `/__market-kit/sync?source=${encodeURIComponent(sourceId)}`
      ]);
      const imported = normalizeImportedCases(payload.payload).map((item) => ({
        ...item,
        source: item.source === "unknown" ? payload.source?.project || payload.source?.id || sourceId : item.source
      }));
      mergeCases(imported);
      syncState = "success";
      syncMessage = `已从 ${payload.source?.label || sourceId} 同步 ${imported.length} 条样本。`;
      persistSyncState();
    } catch (error) {
      syncState = "error";
      syncMessage = error instanceof Error ? `同步失败：${error.message}` : "同步失败，请检查本地代理配置。";
    }
  }

  async function syncAllPresetSources() {
    if (!sampleSources.length) {
      syncState = "error";
      syncMessage = "当前没有可用的远端同步源。";
      return;
    }

    syncState = "loading";
    syncMessage = `正在顺序同步 ${sampleSources.length} 个远端源…`;

    let total = 0;
    try {
      for (const source of sampleSources) {
        const payload = await fetchFromEndpoints([
          `/api/discovery/sync?source=${encodeURIComponent(source.id)}`,
          `/__market-kit/sync?source=${encodeURIComponent(source.id)}`
        ]);
        const imported = normalizeImportedCases(payload.payload).map((item) => ({
          ...item,
          source: item.source === "unknown" ? payload.source?.project || payload.source?.id || source.id : item.source
        }));
        total += imported.length;
        mergeCases(imported);
      }
      syncState = "success";
      syncMessage = `已同步 ${sampleSources.length} 个远端源，共 ${total} 条样本。`;
      persistSyncState();
    } catch (error) {
      syncState = "error";
      syncMessage = error instanceof Error ? `批量同步失败：${error.message}` : "批量同步失败。";
    }
  }

  onMount(() => {
    syncRouteFromLocation();
    window.addEventListener("popstate", syncRouteFromLocation);

    const savedConfig = window.localStorage.getItem(syncConfigKey);
    const savedCases = window.localStorage.getItem(syncCasesKey);
    if (savedConfig) {
      try {
        syncConfig = { ...syncConfig, ...JSON.parse(savedConfig) };
      } catch {}
    }
    if (savedCases) {
      try {
        syncedCases = JSON.parse(savedCases);
      } catch {}
    }
    loadRuntimeRegistry();
    loadRemoteSources({ shouldAutoBootstrap: true });

    return () => {
      window.removeEventListener("popstate", syncRouteFromLocation);
    };
  });

  applyTheme(theme);
</script>

<svelte:head>
  <title>market-kit console</title>
  <meta
    name="description"
    content="Explore market-kit asset identity rules, venue symbol overrides, and market resolution heuristics."
  />
</svelte:head>

<div class="shell">
  <aside class="rail" aria-label="market-kit navigation">
    <div class="rail__brand">
      <div class="rail__mark" aria-hidden="true">MK</div>
      <div class="rail__brand-copy">
        <span>Shared Identity</span>
        <strong>market-kit</strong>
      </div>
    </div>

    <div class="rail__session">
      <span>{registryState === "success" ? "runtime registry" : "embedded registry"}</span>
      <strong>{registryState === "loading" ? "loading..." : registryMessage}</strong>
    </div>

    <nav class="rail__nav">
      <span class="rail__section-label">Views</span>
      <button class:active={page === "registry" || page === "asset-detail"} on:click={() => navigate("registry")}>
        <span class="rail__icon">R</span>
        <span>Registry</span>
      </button>
      <button class:active={page === "rules"} on:click={() => navigate("rules")}>
        <span class="rail__icon">O</span>
        <span>规则检视</span>
      </button>
      <button class:active={page === "groups"} on:click={() => navigate("groups")}>
        <span class="rail__icon">G</span>
        <span>候选分组</span>
      </button>
      <button class:active={page === "symbols"} on:click={() => navigate("symbols")}>
        <span class="rail__icon">S</span>
        <span>Symbol 生成器</span>
      </button>
      <button class:active={page === "samples"} on:click={() => navigate("samples")}>
        <span class="rail__icon">C</span>
        <span>待补样本</span>
      </button>
      <button class:active={page === "playground"} on:click={() => navigate("playground")}>
        <span class="rail__icon">P</span>
        <span>解析试验台</span>
      </button>
    </nav>

    <div class="rail__tools">
      <span class="rail__section-label">Actions</span>
      <button class="theme-toggle" on:click={toggleTheme}>
        <span class="rail__icon">{theme === "dark" ? "L" : "D"}</span>
        <span>{theme === "dark" ? "浅色主题" : "深色主题"}</span>
      </button>
      <button class="theme-toggle" on:click={loadRemoteSources}>
        <span class="rail__icon">↻</span>
        <span>刷新源</span>
      </button>
    </div>

    <div class="rail__summary">
      <div class="mini-stat">
        <span>资产</span>
        <strong>{stats.assets}</strong>
      </div>
      <div class="mini-stat">
        <span>Override</span>
        <strong>{stats.overrides}</strong>
      </div>
      <div class="mini-stat">
        <span>发现</span>
        <strong>{discoveryMarketCount}</strong>
      </div>
      <div class="mini-stat">
        <span>复核</span>
        <strong>{reviewGroupCount}</strong>
      </div>
    </div>
  </aside>

  <main class="stage">
    <section class="hero">
      <div>
        <div class="eyebrow">Identity Operations Console</div>
        <h2>{pageLabel(page)}</h2>
        <p class="hero__copy">{pageDescription(page)}</p>
      </div>
      <div class="hero__status">
        <span>默认解析状态</span>
        <strong class={`status status--${resolution.status}`}>{statusLabel(resolution.status)}</strong>
      </div>
    </section>

    {#if page === "registry"}
      <section class="grid grid--registry">
        <article class="panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Asset Registry</div>
              <h3>资产别名表</h3>
            </div>
            <div class="group-toolbar__stats group-toolbar__stats--head">
              <strong>{assetRows.length}</strong>
              <span>当前资产</span>
            </div>
          </div>
          <div class="asset-toolbar">
            <div class="asset-filter-grid">
              <label>
                <span>检索</span>
                <input bind:value={assetQuery} placeholder="搜索 canonical / alias / class" />
              </label>
              <label>
                <span>资产类别</span>
                <select bind:value={assetClassFilter}>
                  <option value="all">全部</option>
                  {#each assetClassOptions as assetClass}
                    <option value={assetClass}>{assetClassLabel(assetClass)}</option>
                  {/each}
                </select>
              </label>
            </div>
            <div class="asset-toolbar__summary">
              <span class="asset-summary-chip">显示 {assetRows.length} 个资产</span>
              <span class="asset-summary-chip">共 {visibleAliasCount} 个别名</span>
              {#if assetClassFilter !== "all"}
                <span class="asset-summary-chip asset-summary-chip--active">{assetClassLabel(assetClassFilter)}</span>
              {/if}
            </div>
          </div>
          <div class="asset-list">
            {#if assetRows.length}
              {#each assetRows as asset}
                <div
                  class="asset-card"
                  data-asset-class={asset.asset_class}
                  data-selected={selectedAsset?.canonical === asset.canonical}
                  role="button"
                  tabindex="0"
                  aria-pressed={selectedAsset?.canonical === asset.canonical}
                  on:click={() => openAssetDetail(asset.canonical)}
                  on:keydown={(event) => handleAssetCardKeydown(event, asset.canonical)}
                >
                  <div class="asset-card__top">
                    <div class="asset-card__identity">
                      <div class="asset-card__eyebrow">Canonical Asset</div>
                      <div class="asset-card__title-row">
                        <strong>{asset.canonical}</strong>
                        <span class="asset-class-badge">{assetClassLabel(asset.asset_class)}</span>
                      </div>
                    </div>
                    <div class="asset-card__meta">
                      <span class="asset-meta-pill asset-meta-pill--action">查看详情</span>
                      <span class="asset-meta-pill">aliases {totalAssetAliasCount(asset)}</span>
                      {#if asset.unit_aliases?.length}
                        <span class="asset-meta-pill asset-meta-pill--unit">unit aliases {asset.unit_aliases.length}</span>
                      {/if}
                      <span class="asset-meta-pill asset-meta-pill--soft">registry</span>
                    </div>
                  </div>

                  <div class="asset-card__aliases">
                    {#if asset.aliases?.length}
                      {#each asset.aliases as alias}
                        <span class="asset-alias-pill">{alias}</span>
                      {/each}
                    {/if}

                    {#if asset.unit_aliases?.length}
                      {#each asset.unit_aliases as unitAlias}
                        <span class="asset-alias-pill asset-alias-pill--unit">
                          <strong>{unitAlias.alias}</strong>
                          <span class="asset-alias-pill__divider">=</span>
                          <span>{formatUnitMultiplier(unitAlias.multiplier)} {asset.canonical}</span>
                        </span>
                      {/each}
                    {/if}

                    {#if !asset.aliases?.length && !asset.unit_aliases?.length}
                      <span class="asset-alias-pill asset-alias-pill--muted">No extra alias</span>
                    {/if}
                  </div>

                  <div class="asset-card__hint">
                    <span>点击进入二级详情页</span>
                  </div>
                </div>
              {/each}
            {:else}
              <div class="empty-state">
                <strong>当前筛选下没有资产。</strong>
                <p>试试放宽关键词，或者切回“全部”资产类别。</p>
              </div>
            {/if}
          </div>
        </article>

        <div class="panel-stack">
          <article class="panel">
            <div class="panel__head">
              <div>
                <div class="eyebrow">Exchange Aliases</div>
                <h3>交易所别名</h3>
              </div>
            </div>
            <div class="alias-table">
              {#each Object.entries(registry.exchange_aliases || {}) as [source, target]}
                <div class="alias-row">
                  <div class="alias-row__side">
                    <span class="alias-label">input</span>
                    <code>{source}</code>
                  </div>
                  <div class="override-row__arrow">→</div>
                  <div class="alias-row__side alias-row__side--target">
                    <span class="alias-label">canonical</span>
                    <strong>{target}</strong>
                  </div>
                </div>
              {/each}
            </div>
          </article>
        </div>
      </section>
    {/if}

    {#if page === "asset-detail"}
      <section class="asset-detail-page">
        <article class="panel detail-page-header">
          <button class="back-button" on:click={backToRegistry}>← 返回 Registry</button>
          {#if selectedAsset}
            <div class="detail-page-title">
              <div class="eyebrow">Asset Detail</div>
              <h3>{selectedAsset.canonical} 市场身份详情</h3>
              <div class="detail-chip-row">
                <span class="asset-class-badge">{assetClassLabel(selectedAsset.asset_class)}</span>
                <span class="asset-meta-pill">plain aliases {selectedAsset.aliases?.length || 0}</span>
                <span class="asset-meta-pill asset-meta-pill--unit">unit aliases {selectedAsset.unit_aliases?.length || 0}</span>
              </div>
            </div>
            <div class="detail-page-actions">
              <button class="detail-link-button" on:click={openSelectedAssetGroups}>查看候选分组</button>
              <button class="detail-link-button detail-link-button--ghost" on:click={() => {
                overrideQuery = selectedAsset.canonical;
                navigate("rules");
              }}>查看规则</button>
            </div>
          {:else}
            <div class="detail-page-title">
              <div class="eyebrow">Asset Detail</div>
              <h3>未选择标的</h3>
            </div>
          {/if}
        </article>

        {#if selectedAsset}
          <div class="detail-page-grid">
            <article class="panel detail-page-main">
              <div class="detail-stack">
                <div class="detail-hero detail-hero--page" data-asset-class={selectedAsset.asset_class}>
                  <div class="detail-hero__identity">
                    <span class="detail-kicker">Canonical Asset</span>
                    <strong>{selectedAsset.canonical}</strong>
                  </div>
                  <p class="detail-copy">
                    当前详情会先展示静态 registry 里的 alias 和显式 override；discovery 市场分布会在后端发现源同步完成后补齐。
                  </p>
                </div>

                <section class="detail-section">
                  <div class="detail-section__head">
                    <strong>别名与换算</strong>
                    <span>同一个 underlying 的不同记法与单位</span>
                  </div>
                  <div class="detail-chip-row">
                    {#if selectedAsset.aliases?.length}
                      {#each selectedAsset.aliases as alias}
                        <span class="asset-alias-pill">{alias}</span>
                      {/each}
                    {/if}
                    {#if selectedAsset.unit_aliases?.length}
                      {#each selectedAsset.unit_aliases as unitAlias}
                        <span class="asset-alias-pill asset-alias-pill--unit">
                          <strong>{unitAlias.alias}</strong>
                          <span class="asset-alias-pill__divider">=</span>
                          <span>{formatUnitMultiplier(unitAlias.multiplier)} {selectedAsset.canonical}</span>
                        </span>
                      {/each}
                    {/if}
                    {#if !selectedAsset.aliases?.length && !selectedAsset.unit_aliases?.length}
                      <span class="asset-alias-pill asset-alias-pill--muted">Only canonical name is registered</span>
                    {/if}
                  </div>
                </section>

                <section class="detail-section">
                  <div class="detail-section__head">
                    <strong>Registry Overrides</strong>
                    <span>显式映射到这个标的的交易对规则</span>
                  </div>
                  {#if selectedAssetOverrideRows.length}
                    <div class="detail-chip-row">
                      {#each selectedAssetQuotes as quote}
                        <span class="asset-summary-chip">{quote}</span>
                      {/each}
                    </div>
                    <div class="detail-market-list detail-market-list--page">
                      {#each selectedAssetOverrideRows as item}
                        <div class="detail-market-row">
                          <div>
                            <div class="override-row__title">{item.exchange} · {item.market_type}</div>
                            <div class="detail-market-row__symbol">{item.raw_symbol}</div>
                          </div>
                          <div class="detail-market-row__target">{item.canonical_symbol}</div>
                        </div>
                      {/each}
                    </div>
                  {:else}
                    <div class="detail-empty">
                      <strong>当前没有显式 override。</strong>
                      <p>显式 override 只代表写入 registry 的静态规则；发现源里的 Binance / Bybit / Gate 等平台会在右侧 Discovery Presence 里展示。</p>
                    </div>
                  {/if}
                </section>
              </div>
            </article>

            <aside class="panel detail-page-side">
              <div class="detail-stat-grid detail-stat-grid--page">
                <div class="detail-stat">
                  <span>Registry Overrides</span>
                  <strong>{selectedAssetOverrideRows.length}</strong>
                </div>
                <div class="detail-stat">
                  <span>Override Exchanges</span>
                  <strong>{selectedAssetOverrideExchanges.length}</strong>
                </div>
                <div class="detail-stat">
                  <span>Discovery Markets</span>
                  <strong>{selectedAssetMarkets.length}</strong>
                </div>
                <div class="detail-stat">
                  <span>Discovery Exchanges</span>
                  <strong>{selectedAssetDiscoveryExchanges.length}</strong>
                </div>
              </div>

              <section class="detail-section">
                <div class="detail-section__head">
                  <strong>Discovery Presence</strong>
                  <span>当前导入市场里，这个标的出现在哪些平台</span>
                </div>
                {#if selectedAssetMarkets.length}
                  {#if selectedAssetDiscoveryLoading}
                    <div class="sync-state sync-state--loading">
                      <strong>正在补齐 discovery 数据</strong>
                      <p>{discoveryMessage || "页面首屏先展示静态 registry；后端发现源返回后，这里会合并 Binance、Hyperliquid 等市场分布。"}</p>
                    </div>
                  {/if}
                  <div class="detail-chip-row">
                    {#each selectedAssetDiscoveryExchanges as exchange}
                      <span class="asset-summary-chip">{exchange}</span>
                    {/each}
                    {#each selectedAssetDiscoveryMarketTypes as marketType}
                      <span class="asset-summary-chip asset-summary-chip--active">{marketType}</span>
                    {/each}
                  </div>
                  <div class="detail-market-list">
                    {#each selectedAssetMarkets as market}
                      <div class="detail-market-row detail-market-row--compact">
                        <div>
                          <div class="override-row__title">{market.exchange} · {market.marketType || "unknown"}</div>
                          <div class="detail-market-row__symbol">{market.rawSymbol}</div>
                        </div>
                        <div class="detail-market-row__target">{market.canonicalSymbol}</div>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <div class="detail-empty">
                    <strong>
                      {#if selectedAssetDiscoveryLoading}
                        正在读取 discovery 市场。
                      {:else}
                        当前 discovery 里还没看到这个标的。
                      {/if}
                    </strong>
                    <p>
                      {#if selectedAssetDiscoveryLoading}
                        静态 registry 会先显示 `xyz:SKHX` 这类显式规则；Binance 等平台要等 `/api/discovery/current` 或同步接口返回后才会出现在这里。
                      {:else if proxyAvailable}
                        去“候选分组”或“Symbol 生成器”同步发现市场后，这里会自动带出平台分布。
                      {:else}
                        当前还是本地 mock discovery，接上发现源后这里会更完整。
                      {/if}
                    </p>
                  </div>
                {/if}
              </section>
            </aside>
          </div>
        {:else}
          <article class="panel">
            <div class="detail-empty">
              <strong>还没有可查看的标的。</strong>
              <p>回到 Registry，点任意资产卡片进入详情。</p>
            </div>
          </article>
        {/if}
      </section>
    {/if}

    {#if page === "rules"}
      <section class="panel">
        <div class="panel__head">
          <div>
            <div class="eyebrow">Market Overrides</div>
            <h3>显式规则优先级</h3>
          </div>
          <input bind:value={overrideQuery} placeholder="搜索 exchange / symbol / canonical" />
        </div>
        <div class="override-list">
          {#each overrideRows as item}
            <div class="override-row">
              <div>
                <div class="override-row__title">{item.exchange} · {item.market_type}</div>
                <div class="override-row__symbol">{item.raw_symbol}</div>
              </div>
              <div class="override-row__arrow">→</div>
              <div class="override-row__target">{item.canonical_symbol}</div>
            </div>
          {/each}
        </div>
      </section>
    {/if}

    {#if page === "groups"}
      <section class="grid grid--groups">
        <article class="panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Discovery Groups</div>
              <h3>候选资产分组</h3>
            </div>
            <div class="group-toolbar__stats group-toolbar__stats--head">
              <strong>{candidateGroups.length}</strong>
              <span>当前结果</span>
            </div>
          </div>

          <div class="sync-stack">
            <div class="sync-proxy-banner">
              <div>
                <strong>{proxyAvailable ? "已发现可同步的发现源" : "当前使用本地示例市场"}</strong>
                <p>
                  {#if proxyAvailable}
                    候选分组页可以直接通过本地代理拉取市场发现清单。首次进入时会优先自动加载 market-kit 内建的交易所直连启动数据；后续你也可以切换到 slipstream。
                  {:else}
                    当前还没有连接发现源，所以先展示仓库内置的 mock discovery 数据。
                  {/if}
                </p>
              </div>
            </div>

            {#if proxyAvailable}
              <div class="sync-grid">
                <label>
                  <span>发现源</span>
                  <select bind:value={selectedDiscoverySourceId}>
                    {#if discoverySources.length === 0}
                      <option value="">暂无发现源</option>
                    {/if}
                    {#if discoverySources.length > 1}
                      <option value={allDiscoverySourceId}>全部发现源（合并）</option>
                    {/if}
                    {#each discoverySources as source}
                      <option value={source.id}>{source.label} ({sourceBadge(source)})</option>
                    {/each}
                  </select>
                </label>
                <label>
                  <span>当前导入源</span>
                  <input value={discoveryEnvelope?.source || "mock-discovery"} disabled />
                </label>
              </div>

              <div class="sync-actions">
                <div class="button-row">
                  <button class="sync-button" on:click={() => syncDiscoverySource()} disabled={discoveryState === "loading" || !selectedDiscoverySourceId}>
                    {discoveryState === "loading" ? "同步中…" : "同步发现市场"}
                  </button>
                  <button class="sync-button sync-button--secondary" on:click={resetDiscoveryToMock}>
                    切回本地示例
                  </button>
                </div>
                <div class={`sync-state sync-state--${discoveryState}`}>
                  <strong>
                    {#if discoveryState === "success"}
                      导入完成
                    {:else if discoveryState === "error"}
                      导入失败
                    {:else if discoveryState === "loading"}
                      正在导入
                    {:else}
                      尚未导入
                    {/if}
                  </strong>
                  <p>{discoveryMessage || "将发现源返回的市场清单拉回本地，再自动聚成候选资产分组。"} </p>
                </div>
              </div>
            {/if}
          </div>

          <div class="group-toolbar">
            <div class="group-filter-grid">
              <label>
                <span>检索</span>
                <input bind:value={groupQuery} placeholder="按资产、交易所、symbol 搜索" />
              </label>
              <label>
                <span>状态</span>
                <select bind:value={groupStatusFilter}>
                  <option value="all">全部</option>
                  <option value="review">只看待复核</option>
                  <option value="ready">只看已成组</option>
                </select>
              </label>
              <label>
                <span>资产类别</span>
                <select bind:value={groupAssetClassFilter}>
                  <option value="all">全部</option>
                  {#each groupAssetClassOptions as assetClass}
                    <option value={assetClass}>{assetClass}</option>
                  {/each}
                </select>
              </label>
              <label>
                <span>市场类型</span>
                <select bind:value={groupMarketTypeFilter}>
                  <option value="all">全部</option>
                  {#each groupMarketTypeOptions as marketType}
                    <option value={marketType}>{marketType}</option>
                  {/each}
                </select>
              </label>
              <label>
                <span>交易所</span>
                <select bind:value={groupExchangeFilter}>
                  <option value="all">全部</option>
                  {#each groupExchangeOptions as exchange}
                    <option value={exchange}>{exchange}</option>
                  {/each}
                </select>
              </label>
            </div>

            <div class="group-toolbar__footer">
              <div class="group-toolbar__stats">
                <strong>{candidateGroups.length}</strong>
                <span>当前结果</span>
                <span>待复核 {reviewGroupCount}</span>
                <span>已成组 {readyGroupCount}</span>
              </div>
              <div class="button-row">
                <button class="sync-button sync-button--ghost" on:click={resetGroupFilters}>
                  清空筛选
                </button>
                <button class="sync-button sync-button--ghost" on:click={collapseAllGroups}>
                  全部收起
                </button>
                <button class="sync-button sync-button--secondary" on:click={expandVisibleGroups}>
                  展开当前结果
                </button>
              </div>
            </div>
          </div>

          <div class="group-list">
            {#if candidateGroups.length}
              {#each candidateGroups as group}
                <div class={`group-card ${expandedGroups[group.groupKey] ? "group-card--expanded" : ""}`}>
                  <div class="group-card__summary">
                    <button class="group-card__main" on:click={() => toggleGroupExpanded(group.groupKey)}>
                      <div class="group-card__headline">
                        <div class="group-card__identity">
                          <div class="group-card__meta">{group.exchanges.join(" · ")}</div>
                          <strong>{group.groupKey}</strong>
                        </div>
                        <span class={`status status--${group.needsReview ? "ambiguous" : "resolved"}`}>
                          {group.needsReview ? "待复核" : "已成组"}
                        </span>
                      </div>

                      <div class="group-card__metrics">
                        <div>
                          <span>类别</span>
                          <strong>{group.assetClass}</strong>
                        </div>
                        <div>
                          <span>市场</span>
                          <strong>{group.markets.length}</strong>
                        </div>
                        <div>
                          <span>类型</span>
                          <strong>{group.marketTypes.join(" / ") || "未识别"}</strong>
                        </div>
                        <div>
                          <span>置信度</span>
                          <strong>{Math.round((group.primaryConfidence || 0) * 100)}%</strong>
                        </div>
                      </div>

                      <div class="group-card__preview">
                        <div class="group-chip-row">
                          {#each group.venueTypes as venueType}
                            <span class="group-chip">{venueType}</span>
                          {/each}
                        </div>
                        <div class="group-card__symbols">
                          {#each group.markets.slice(0, 3) as market}
                            <span>{market.platform}: {market.rawSymbol}</span>
                          {/each}
                          {#if group.markets.length > 3}
                            <span>+{group.markets.length - 3} more</span>
                          {/if}
                        </div>
                      </div>
                    </button>

                    <button class="group-card__toggle" on:click={() => toggleGroupExpanded(group.groupKey)}>
                      {expandedGroups[group.groupKey] ? "收起" : "展开"}
                    </button>
                  </div>

                  {#if expandedGroups[group.groupKey]}
                    <div class="group-card__details">
                      <div class="group-chip-row">
                        {#each group.evidence.slice(0, 3) as evidence}
                          <span class="group-chip group-chip--muted">{evidence}</span>
                        {/each}
                      </div>

                      <div class="group-market-table">
                        {#each group.markets as market}
                          <div class="group-market-row">
                            <div>
                              <div class="group-market-row__title">{market.platform} · {market.marketType || "unknown"}</div>
                              <div class="group-market-row__symbol">{market.rawSymbol}</div>
                            </div>
                            <div class="group-market-row__arrow">→</div>
                            <div class="group-market-row__target">{market.canonicalSymbol || "未形成 canonical"}</div>
                          </div>
                        {/each}
                      </div>
                    </div>
                  {/if}
                </div>
              {/each}
            {:else}
              <div class="empty-state">
                <strong>当前筛选下没有候选组。</strong>
                <p>试试清空筛选，或者先同步一份新的发现市场清单。</p>
              </div>
            {/if}
          </div>
        </article>

        <article class="panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Review Logic</div>
              <h3>这页在看什么</h3>
            </div>
          </div>

          <div class="sample-list">
            <div class="sync-note">
              <strong>发现层可以来自多个源</strong>
              <p>这里看的不是最终规则，而是从 slipstream 或 market-kit 自举发现源导入的市场清单，先按 base/quote family 聚成候选组。</p>
            </div>
            <div class="sync-note">
              <strong>为什么要候选组</strong>
              <p>像 <code>DRAM-USDT-SWAP</code>、<code>DRAMUSDT</code>、<code>DRAM</code> 这种不同 venue symbol，不应该直接各自进 registry，而是先进入同一个 family 做人工审核。</p>
            </div>
            <div class="sync-note">
              <strong>当前版本的聚合依据</strong>
              <p>第一版主要依据显式 base/quote、market type、交易所别名和 resolver 结果，不用价格做主键。价格以后只作为辅助证据。</p>
            </div>
          </div>
        </article>
      </section>
    {/if}

    {#if page === "symbols"}
      <section class="grid grid--symbols">
        <article class="panel symbol-panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Symbol Lab</div>
              <h3>Symbol 生成器</h3>
            </div>
            <div class="group-toolbar__stats group-toolbar__stats--head">
              <strong>{symbolRows.length}</strong>
              <span>当前结果</span>
            </div>
          </div>

          <div class="symbol-lab-brief">
            <div>
              <strong>从 registry 与 discovery 组合筛 symbol</strong>
              <p>主筛选决定输出列表，交集条件用于限定 underlying 必须也存在于另一组市场。</p>
            </div>
            <button class="sync-button sync-button--secondary" on:click={applyStockPerpOndoPreset}>
              Binance Stock Perp ∩ Ondo
            </button>
          </div>

          <div class="symbol-filter-stack">
            <label>
              <span>检索</span>
              <input bind:value={symbolQuery} placeholder="AAPL / AAPLUSDT / binance-web3" />
            </label>

            <div class="symbol-filter-section">
              <div class="symbol-filter-section__head">
                <strong>主筛选</strong>
                <button class="tiny-button" on:click={() => {
                  symbolPlatformFilter = [];
                  symbolMarketTypeFilter = [];
                  symbolAssetClassFilter = [];
                }}>全部</button>
              </div>

              <div class="symbol-filter-group">
                <span>平台</span>
                <div class="filter-chip-row">
                  {#each symbolPlatformOptions as platform}
                    <button
                      class:active={symbolPlatformFilter.includes(platform)}
                      class="filter-chip"
                      on:click={() => (symbolPlatformFilter = toggleArrayValue(symbolPlatformFilter, platform))}
                    >
                      {platformLabel(platform)}
                    </button>
                  {/each}
                </div>
              </div>

              <div class="symbol-filter-group">
                <span>市场类型</span>
                <div class="filter-chip-row">
                  {#each symbolMarketTypeOptions as marketType}
                    <button
                      class:active={symbolMarketTypeFilter.includes(marketType)}
                      class="filter-chip"
                      on:click={() => (symbolMarketTypeFilter = toggleArrayValue(symbolMarketTypeFilter, marketType))}
                    >
                      {marketType}
                    </button>
                  {/each}
                </div>
              </div>

              <div class="symbol-filter-group">
                <span>资产类别</span>
                <div class="filter-chip-row">
                  {#each symbolAssetClassOptions as assetClass}
                    <button
                      class:active={symbolAssetClassFilter.includes(assetClass)}
                      class="filter-chip"
                      on:click={() => (symbolAssetClassFilter = toggleArrayValue(symbolAssetClassFilter, assetClass))}
                    >
                      {assetClassLabel(assetClass)}
                    </button>
                  {/each}
                </div>
              </div>

              <div class="symbol-filter-group">
                <span>数据层</span>
                <div class="filter-chip-row">
                  {#each symbolLayerOptions as layer}
                    <button
                      class:active={symbolLayerFilter.includes(layer)}
                      class="filter-chip"
                      on:click={() => (symbolLayerFilter = toggleArrayValue(symbolLayerFilter, layer))}
                    >
                      {layerLabel(layer)}
                    </button>
                  {/each}
                </div>
              </div>
            </div>

            <div class="symbol-filter-section symbol-filter-section--presence">
              <div class="symbol-filter-section__head">
                <label class="inline-toggle">
                  <input type="checkbox" bind:checked={symbolPresenceEnabled} />
                  <span>启用交集条件</span>
                </label>
                <span class="asset-summary-chip">{symbolPresenceBaseCount} 个 underlying</span>
              </div>

              <div class="symbol-filter-group">
                <span>必须存在的平台</span>
                <div class="filter-chip-row">
                  {#each symbolPlatformOptions as platform}
                    <button
                      class:active={symbolPresencePlatformFilter.includes(platform)}
                      class="filter-chip"
                      on:click={() => (symbolPresencePlatformFilter = toggleArrayValue(symbolPresencePlatformFilter, platform))}
                    >
                      {platformLabel(platform)}
                    </button>
                  {/each}
                </div>
              </div>

              <div class="symbol-filter-grid">
                <label>
                  <span>存在市场类型</span>
                  <select value={symbolPresenceMarketTypeFilter[0] || ""} on:change={(event) => (symbolPresenceMarketTypeFilter = event.currentTarget.value ? [event.currentTarget.value] : [])}>
                    <option value="">全部</option>
                    {#each symbolMarketTypeOptions as marketType}
                      <option value={marketType}>{marketType}</option>
                    {/each}
                  </select>
                </label>
                <label>
                  <span>存在资产类别</span>
                  <select value={symbolPresenceAssetClassFilter[0] || ""} on:change={(event) => (symbolPresenceAssetClassFilter = event.currentTarget.value ? [event.currentTarget.value] : [])}>
                    <option value="">全部</option>
                    {#each symbolAssetClassOptions as assetClass}
                      <option value={assetClass}>{assetClassLabel(assetClass)}</option>
                    {/each}
                  </select>
                </label>
              </div>
            </div>

            <div class="symbol-filter-grid">
              <label>
                <span>输出字段</span>
                <select bind:value={symbolOutputField}>
                  <option value="rawSymbol">Raw Symbol</option>
                  <option value="canonicalSymbol">Canonical Symbol</option>
                  <option value="baseAsset">Base Asset</option>
                </select>
              </label>
              <label>
                <span>输出格式</span>
                <select bind:value={symbolOutputFormat}>
                  <option value="quotedCsv">"A","B"</option>
                  <option value="csv">A,B</option>
                  <option value="lines">逐行</option>
                  <option value="json">JSON Array</option>
                </select>
              </label>
            </div>

            <div class="button-row">
              <button class="sync-button sync-button--ghost" on:click={resetSymbolFilters}>清空筛选</button>
              {#if proxyAvailable}
                <button class="sync-button sync-button--secondary" on:click={() => syncDiscoverySource()} disabled={discoveryState === "loading" || !selectedDiscoverySourceId}>
                  {discoveryState === "loading" ? "同步中…" : "同步发现市场"}
                </button>
              {/if}
            </div>
          </div>
        </article>

        <article class="panel symbol-results-panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Generated Set</div>
              <h3>{symbolBaseCount} 个 underlying</h3>
            </div>
            <div class="group-toolbar__stats group-toolbar__stats--head">
              <strong>{symbolSelectedCount}</strong>
              <span>已选</span>
            </div>
          </div>

          <div class="symbol-summary-strip">
            <span class="asset-summary-chip asset-summary-chip--active">Rows {symbolRows.length}</span>
            <span class="asset-summary-chip">Presence {symbolPresenceEnabled ? "on" : "off"}</span>
            <span class="asset-summary-chip">{discoveryEnvelope?.source || "mock-discovery"}</span>
          </div>

          <div class="button-row symbol-selection-actions">
            <button class="sync-button sync-button--secondary" on:click={selectAllSymbolRows}>全选当前结果</button>
            <button class="sync-button sync-button--ghost" on:click={invertSymbolSelection}>反选</button>
            <button class="sync-button sync-button--ghost" on:click={deselectAllSymbolRows}>取消选择</button>
          </div>

          <div class="symbol-list">
            {#if symbolRows.length}
              {#each symbolRows as item}
                <button
                  class="symbol-row"
                  class:selected={selectedSymbolIds.has(item.id)}
                  on:click={() => toggleSymbolRow(item.id)}
                >
                  <div class="symbol-row__main">
                    <div class="symbol-row__title">
                      <strong>{item.rawSymbol}</strong>
                      <span>{item.baseAsset}</span>
                    </div>
                    <div class="symbol-row__meta">
                      <span>{item.platformLabel}</span>
                      <span>{item.marketType}</span>
                      <span>{assetClassLabel(item.assetClass)}</span>
                      <span>{layerLabel(item.layer)}</span>
                    </div>
                  </div>
                  <div class="symbol-row__target">{item.canonicalSymbol}</div>
                </button>
              {/each}
            {:else}
              <div class="empty-state">
                <strong>没有匹配的 symbol。</strong>
                <p>如果你在等 Binance Web3/Ondo 列表，先同步最新 discovery，或放宽交集条件看主筛选是否有结果。</p>
              </div>
            {/if}
          </div>

          <div class="generated-box">
            <div class="generated-box__head">
              <strong>生成结果</strong>
              <div class="button-row">
                <button class="sync-button sync-button--secondary" on:click={copyGeneratedSymbolText}>复制</button>
              </div>
            </div>
            <pre>{generatedSymbolText || "选择 symbol 后生成文本"}</pre>
            {#if symbolCopyMessage}
              <p>{symbolCopyMessage}</p>
            {/if}
          </div>
        </article>
      </section>
    {/if}

    {#if page === "samples"}
      <section class="grid grid--samples">
        <article class="panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Remote Sync</div>
              <h3>远端待补规则样本</h3>
            </div>
          </div>

          <div class="sync-stack">
            <div class="sync-proxy-banner">
              <div>
                <strong>{proxyAvailable ? "已发现本地同步代理" : "当前是纯静态模式"}</strong>
                <p>
                  {#if proxyAvailable}
                    你现在可以直接选择预设源，一键同步 ECS 上导出的样本，不需要手填 URL 或处理浏览器跨域。
                  {:else}
                    只有在 <code>pnpm dev</code> 本地开发模式下，才会启用自动同步代理。现在仍可使用下方手动 URL 兜底。
                  {/if}
                </p>
              </div>
            </div>

            {#if proxyAvailable}
              <div class="sync-grid">
                <label>
                  <span>预设同步源</span>
                  <select bind:value={selectedSourceId}>
                    {#if sampleSources.length === 0}
                      <option value="">暂无可用源</option>
                    {/if}
                    {#each sampleSources as source}
                      <option value={source.id}>{source.label} ({sourceBadge(source)})</option>
                    {/each}
                  </select>
                </label>
                <label>
                  <span>远端地址</span>
                  <input value={sampleSources.find((item) => item.id === selectedSourceId)?.url || "未配置"} disabled />
                </label>
              </div>

              <div class="sync-actions">
                <div class="button-row">
                  <button class="sync-button" on:click={() => syncPresetSource()} disabled={syncState === "loading" || !selectedSourceId}>
                    {syncState === "loading" ? "同步中…" : "同步当前源"}
                  </button>
                  <button class="sync-button sync-button--secondary" on:click={syncAllPresetSources} disabled={syncState === "loading" || sampleSources.length === 0}>
                    同步全部预设源
                  </button>
                  <button class="sync-button sync-button--ghost" on:click={loadRemoteSources}>
                    刷新预设
                  </button>
                </div>
                <div class={`sync-state sync-state--${syncState}`}>
                  <strong>
                    {#if syncState === "success"}
                      同步完成
                    {:else if syncState === "error"}
                      同步失败
                    {:else if syncState === "loading"}
                      正在同步
                    {:else}
                      尚未同步
                    {/if}
                  </strong>
                  <p>{syncMessage || "从本地配置的预设源拉取 unresolved / ambiguous 样本。"} </p>
                </div>
              </div>
            {/if}

            <div class="sync-note">
              <strong>本地配置文件</strong>
              <p>
                将你的远端源写入 <code>frontend/sync-sources.local.json</code>。仓库里已提供
                <code>frontend/sync-sources.example.json</code> 作为模板，而且 <code>.local</code> 文件已加入忽略，不会污染公共仓库。
              </p>
            </div>

            <div class="sync-manual">
              <div class="eyebrow">Manual Fallback</div>
              <h4>手动 URL 兜底</h4>
            </div>

            <div class="sync-grid">
              <label>
                <span>来源标签</span>
                <input bind:value={syncConfig.source} placeholder="veridex" />
              </label>
              <label class="field-span-2">
                <span>导出地址</span>
                <input bind:value={syncConfig.url} placeholder="https://api.example.com/veridex-api/api/v1/identity-cases" />
              </label>
              <label>
                <span>鉴权 Header</span>
                <input bind:value={syncConfig.authHeader} placeholder="X-Veridex-Admin-Code" />
              </label>
              <label>
                <span>鉴权值</span>
                <input bind:value={syncConfig.authValue} placeholder="可留空" />
              </label>
            </div>

            <div class="sync-actions">
              <button class="sync-button" on:click={syncRemoteCases} disabled={syncState === "loading"}>
                {syncState === "loading" ? "同步中…" : "同步远端样本"}
              </button>
              <div class={`sync-state sync-state--${syncState}`}>
                <strong>
                  {#if syncState === "success"}
                    同步完成
                  {:else if syncState === "error"}
                    同步失败
                  {:else if syncState === "loading"}
                    正在同步
                  {:else}
                    尚未同步
                  {/if}
                </strong>
                <p>{syncMessage || "将 ECS 上导出的 unresolved / ambiguous 样本拉回本地审核。"} </p>
              </div>
            </div>

            <div class="sync-note">
              <strong>兼容说明</strong>
              <p>支持直接返回数组，也支持 <code>{`{ cases: [...] }`}</code>、<code>{`{ items: [...] }`}</code>、<code>{`{ data: [...] }`}</code> 这类包裹格式。只有在手动 URL 模式下，才需要额外确认远端接口是否允许浏览器跨域访问。</p>
            </div>
          </div>
        </article>

        <article class="panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Imported Cases</div>
              <h3>已同步样本</h3>
            </div>
            <input bind:value={syncQuery} placeholder="搜索 source / exchange / symbol / canonical" />
          </div>

          <div class="sample-list">
            {#if syncRows.length}
              {#each syncRows as item}
                <div class="sample-card">
                  <div class="sample-card__head">
                    <div>
                      <div class="sample-card__meta">{item.source} · {item.request.exchange || "unknown"}</div>
                      <strong>{item.request.symbol || "未提供 raw symbol"}</strong>
                    </div>
                    <span class={`status status--${item.status}`}>{syncStatusLabel(item.status)}</span>
                  </div>

                  <div class="sample-card__grid">
                    <div>
                      <span>市场提示</span>
                      <strong>{item.request.marketTypeHint || "未提供"}</strong>
                    </div>
                    <div>
                      <span>Canonical 提示</span>
                      <strong>{item.request.canonicalSymbolHint || "未提供"}</strong>
                    </div>
                    <div>
                      <span>解析结果</span>
                      <strong>{item.resolution?.market?.canonicalSymbol || item.resolution?.reason || "未确定"}</strong>
                    </div>
                    <div>
                      <span>出现次数</span>
                      <strong>{item.count}</strong>
                    </div>
                  </div>

                  <div class="sample-card__timeline">
                    <span>首次出现：{formatTime(item.firstSeenAt)}</span>
                    <span>最近出现：{formatTime(item.lastSeenAt)}</span>
                  </div>

                  {#if item.reason}
                    <p class="sample-card__reason">{item.reason}</p>
                  {/if}
                </div>
              {/each}
            {:else}
              <div class="empty-state">
                <strong>还没有同步到样本。</strong>
                <p>先填入 `veridex / tradfi-monitor` 的导出地址，再点一次“同步远端样本”。</p>
              </div>
            {/if}
          </div>
        </article>
      </section>
    {/if}

    {#if page === "playground"}
      <section class="grid grid--playground">
        <article class="panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Resolver Input</div>
              <h3>解析试验台</h3>
            </div>
          </div>
          <div class="form-grid">
            <label>
              <span>交易所</span>
              <input bind:value={request.exchange} placeholder="okx" />
            </label>
            <label>
              <span>Raw Symbol</span>
              <input bind:value={request.symbol} placeholder="DRAM-USDT-SWAP" />
            </label>
            <label>
              <span>市场类型提示</span>
              <input bind:value={request.marketTypeHint} placeholder="perpetual / spot" />
            </label>
            <label>
              <span>Canonical 提示</span>
              <input bind:value={request.canonicalSymbolHint} placeholder="DRAM/USDT" />
            </label>
            <label>
              <span>InstType</span>
              <input bind:value={request.instType} placeholder="SWAP / SPOT" />
            </label>
            <label>
              <span>ProductType</span>
              <input bind:value={request.productType} placeholder="USDT-FUTURES" />
            </label>
          </div>
        </article>

        <article class="panel result-panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Resolve Result</div>
              <h3>解析输出</h3>
            </div>
            <span class={`status status--${resolution.status}`}>{statusLabel(resolution.status)}</span>
          </div>

          <div class="result-meta">
            <div>
              <span>Confidence</span>
              <strong>{Math.round((resolution.confidence || 0) * 100)}%</strong>
            </div>
            <div>
              <span>Reason</span>
              <strong>{resolution.reason}</strong>
            </div>
          </div>

          {#if resolution.market}
            <div class="identity-card">
              <div class="identity-line"><span>exchange</span><strong>{resolution.market.exchange}</strong></div>
              <div class="identity-line"><span>marketType</span><strong>{resolution.market.marketType}</strong></div>
              <div class="identity-line"><span>rawSymbol</span><strong>{resolution.market.rawSymbol}</strong></div>
              <div class="identity-line"><span>venueSymbol</span><strong>{resolution.market.venueSymbol}</strong></div>
              <div class="identity-line"><span>canonical</span><strong>{resolution.market.canonicalSymbol}</strong></div>
              <div class="identity-line"><span>assetClass</span><strong>{resolution.market.assetClass}</strong></div>
            </div>
          {:else if resolution.candidates?.length}
            <div class="candidate-list">
              {#each resolution.candidates as candidate}
                <div class="identity-card">
                  <div class="identity-line"><span>exchange</span><strong>{candidate.exchange}</strong></div>
                  <div class="identity-line"><span>marketType</span><strong>{candidate.marketType}</strong></div>
                  <div class="identity-line"><span>canonical</span><strong>{candidate.canonicalSymbol}</strong></div>
                </div>
              {/each}
            </div>
          {:else}
            <div class="empty-state">
              <strong>当前输入尚不能确定唯一市场身份。</strong>
              <p>这正是 `market-kit` 需要显式返回 `unresolved / ambiguous` 的原因，而不是继续硬猜。</p>
            </div>
          {/if}
        </article>
      </section>
    {/if}
  </main>
</div>
