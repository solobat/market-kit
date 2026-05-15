<script>
  import { onMount } from "svelte";
  import { buildCandidateGroups, loadDiscoveryEnvelope, normalizeDiscoveryEnvelope } from "./lib/discovery.js";
  import { loadRegistry, normalizeImportedCases, registryStats, resolveIdentity } from "./lib/identity.js";

  const registry = loadRegistry();
  const stats = registryStats();
  const discoveryStorageKey = "market-kit.discovery-envelope";
  const defaultDiscoveryEnvelope = loadDiscoveryEnvelope();
  const syncConfigKey = "market-kit.sync-config";
  const syncCasesKey = "market-kit.sync-cases";

  let theme = "dark";
  let assetQuery = "";
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
  let proxyAvailable = false;
  let remoteSources = [];
  let selectedSourceId = "";
  let discoveryEnvelope = defaultDiscoveryEnvelope;
  let discoveryState = "idle";
  let discoveryMessage = "";
  let selectedDiscoverySourceId = "";

  $: assetRows = (registry.asset_aliases || []).filter((item) => {
    const query = assetQuery.trim().toUpperCase();
    if (!query) return true;
    return (
      String(item.canonical || "").toUpperCase().includes(query) ||
      String(item.asset_class || "").toUpperCase().includes(query) ||
      (item.aliases || []).some((alias) => String(alias || "").toUpperCase().includes(query))
    );
  });

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
  $: candidateGroups = buildCandidateGroups(discoveryEnvelope).filter((group) => {
    const query = groupQuery.trim().toUpperCase();
    if (!query) return true;
    return (
      String(group.groupKey || "").toUpperCase().includes(query) ||
      String(group.assetClass || "").toUpperCase().includes(query) ||
      (group.exchanges || []).some((value) => String(value || "").toUpperCase().includes(query)) ||
      (group.markets || []).some((market) =>
        [market.rawSymbol, market.platform, market.marketType, market.venueType]
          .some((value) => String(value || "").toUpperCase().includes(query))
      )
    );
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

  function pageLabel(value) {
    if (value === "groups") return "候选分组";
    if (value === "samples") return "待补样本";
    if (value === "playground") return "解析试验台";
    if (value === "rules") return "规则检视";
    return "Registry";
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

  function persistSyncState() {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(syncConfigKey, JSON.stringify(syncConfig));
    window.localStorage.setItem(syncCasesKey, JSON.stringify(syncedCases));
  }

  function persistDiscoveryState() {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(discoveryStorageKey, JSON.stringify(discoveryEnvelope));
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

  async function loadRemoteSources() {
    try {
      const payload = await fetchFromEndpoints(["/api/discovery/sources", "/__market-kit/sources"]);
      remoteSources = Array.isArray(payload.sources) ? payload.sources : [];
      proxyAvailable = true;
      if (!selectedSourceId && remoteSources.length) {
        selectedSourceId = remoteSources[0].id;
      }
      const discoverySources = remoteSources.filter((item) =>
        String(item.project || item.id || "").toLowerCase().includes("slipstream")
      );
      if (!selectedDiscoverySourceId && discoverySources.length) {
        selectedDiscoverySourceId = discoverySources[0].id;
      }
    } catch {
      proxyAvailable = false;
      remoteSources = [];
    }
  }

  async function syncDiscoverySource(sourceId = selectedDiscoverySourceId) {
    if (!sourceId) {
      discoveryState = "error";
      discoveryMessage = "请先选择一个 slipstream 导出源。";
      return;
    }

    discoveryState = "loading";
    discoveryMessage = "正在同步 slipstream 市场清单…";

    try {
      const payload = await fetchFromEndpoints([
        `/api/discovery/sync?source=${encodeURIComponent(sourceId)}`,
        `/__market-kit/sync?source=${encodeURIComponent(sourceId)}`
      ]);
      const project = payload.source?.project || payload.source?.id || "slipstream";
      discoveryEnvelope = normalizeDiscoveryEnvelope(payload.payload, project);
      discoveryState = "success";
      discoveryMessage = `已从 ${payload.source?.label || sourceId} 导入 ${discoveryEnvelope.items.length} 个市场。`;
      persistDiscoveryState();
    } catch (error) {
      discoveryState = "error";
      discoveryMessage = error instanceof Error ? `同步失败：${error.message}` : "同步 slipstream 导出失败。";
    }
  }

  function resetDiscoveryToMock() {
    discoveryEnvelope = defaultDiscoveryEnvelope;
    discoveryState = "success";
    discoveryMessage = "已切回仓库内置的本地示例市场清单。";
    persistDiscoveryState();
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
    if (!remoteSources.length) {
      syncState = "error";
      syncMessage = "当前没有可用的远端同步源。";
      return;
    }

    syncState = "loading";
    syncMessage = `正在顺序同步 ${remoteSources.length} 个远端源…`;

    let total = 0;
    try {
      for (const source of remoteSources) {
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
      syncMessage = `已同步 ${remoteSources.length} 个远端源，共 ${total} 条样本。`;
      persistSyncState();
    } catch (error) {
      syncState = "error";
      syncMessage = error instanceof Error ? `批量同步失败：${error.message}` : "批量同步失败。";
    }
  }

  onMount(() => {
    const savedConfig = window.localStorage.getItem(syncConfigKey);
    const savedCases = window.localStorage.getItem(syncCasesKey);
    const savedDiscovery = window.localStorage.getItem(discoveryStorageKey);
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
    if (savedDiscovery) {
      try {
        discoveryEnvelope = JSON.parse(savedDiscovery);
      } catch {}
    }
    loadRemoteSources();
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
  <aside class="rail">
    <div class="rail__brand">
      <div class="rail__mark">MK</div>
      <div>
        <div class="eyebrow">Shared Identity Layer</div>
        <h1>market-kit</h1>
      </div>
    </div>

    <nav class="rail__nav">
      <button class:active={page === "registry"} on:click={() => (page = "registry")}>Registry</button>
      <button class:active={page === "rules"} on:click={() => (page = "rules")}>规则检视</button>
      <button class:active={page === "groups"} on:click={() => (page = "groups")}>候选分组</button>
      <button class:active={page === "samples"} on:click={() => (page = "samples")}>待补样本</button>
      <button class:active={page === "playground"} on:click={() => (page = "playground")}>解析试验台</button>
    </nav>

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
        <span>交易所</span>
        <strong>{stats.exchanges}</strong>
      </div>
      <div class="mini-stat">
        <span>别名</span>
        <strong>{stats.exchangeAliases}</strong>
      </div>
      <div class="mini-stat">
        <span>未解析</span>
        <strong>{unresolvedCount}</strong>
      </div>
      <div class="mini-stat">
        <span>歧义</span>
        <strong>{ambiguousCount}</strong>
      </div>
      <div class="mini-stat">
        <span>发现市场</span>
        <strong>{discoveryMarketCount}</strong>
      </div>
      <div class="mini-stat">
        <span>待复核组</span>
        <strong>{reviewGroupCount}</strong>
      </div>
    </div>

    <div class="rail__note">
      <div class="eyebrow">Design Note</div>
      <p>
        这不是行情面板，而是统一的身份层控制台。它负责收口 <code>symbol / marketType / alias</code>，
        让多个独立仓库共享同一套判断。
      </p>
    </div>

    <button class="theme-toggle" on:click={toggleTheme}>{theme === "dark" ? "切到浅色" : "切到深色"}</button>
  </aside>

  <main class="stage">
    <section class="hero">
      <div>
        <div class="eyebrow">Identity Operations Console</div>
        <h2>{pageLabel(page)}</h2>
        <p class="hero__copy">
          参考 `tradfi-monitor / slipstream` 的控制台感，但重点展示 shared registry、override 规则和解析结果，而不是某个单一业务项目。
        </p>
      </div>
      <div class="hero__status">
        <span>默认解析状态</span>
        <strong class={`status status--${resolution.status}`}>{statusLabel(resolution.status)}</strong>
      </div>
    </section>

    {#if page === "registry"}
      <section class="grid">
        <article class="panel">
          <div class="panel__head">
            <div>
              <div class="eyebrow">Asset Registry</div>
              <h3>资产别名表</h3>
            </div>
            <input bind:value={assetQuery} placeholder="搜索 canonical / alias / class" />
          </div>
          <div class="asset-list">
            {#each assetRows as asset}
              <div class="asset-card">
                <div class="asset-card__head">
                  <strong>{asset.canonical}</strong>
                  <span>{asset.asset_class}</span>
                </div>
                <div class="asset-card__aliases">
                  {#if asset.aliases?.length}
                    {#each asset.aliases as alias}
                      <span>{alias}</span>
                    {/each}
                  {:else}
                    <span class="muted">无额外 alias</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </article>

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
                <code>{source}</code>
                <span>→</span>
                <strong>{target}</strong>
              </div>
            {/each}
          </div>
        </article>
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
            <input bind:value={groupQuery} placeholder="搜索 group / exchange / symbol / class" />
          </div>

          <div class="sync-stack">
            <div class="sync-proxy-banner">
              <div>
                <strong>{proxyAvailable ? "已发现 slipstream 同步源" : "当前使用本地示例市场"}</strong>
                <p>
                  {#if proxyAvailable}
                    候选分组页可以直接通过本地代理拉取 slipstream 的市场清单，再自动聚成待审核的 asset family。
                  {:else}
                    当前还没有连接 slipstream 导出，所以先展示仓库内置的 mock discovery 数据。
                  {/if}
                </p>
              </div>
            </div>

            {#if proxyAvailable}
              <div class="sync-grid">
                <label>
                  <span>Slipstream 导出源</span>
                  <select bind:value={selectedDiscoverySourceId}>
                    {#if remoteSources.filter((item) => String(item.project || item.id || "").toLowerCase().includes("slipstream")).length === 0}
                      <option value="">暂无 slipstream 源</option>
                    {/if}
                    {#each remoteSources.filter((item) => String(item.project || item.id || "").toLowerCase().includes("slipstream")) as source}
                      <option value={source.id}>{source.label} ({source.project || source.id})</option>
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
                    {discoveryState === "loading" ? "同步中…" : "同步 slipstream 市场"}
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
                  <p>{discoveryMessage || "将 slipstream 的市场发现结果拉回本地，再自动聚成候选资产分组。"} </p>
                </div>
              </div>
            {/if}
          </div>

          <div class="group-list">
            {#each candidateGroups as group}
              <div class="group-card">
                <div class="group-card__head">
                  <div>
                    <div class="group-card__meta">{group.exchanges.join(" · ")}</div>
                    <strong>{group.groupKey}</strong>
                  </div>
                  <span class={`status status--${group.needsReview ? "ambiguous" : "resolved"}`}>
                    {group.needsReview ? "待复核" : "已成组"}
                  </span>
                </div>

                <div class="group-card__grid">
                  <div>
                    <span>资产类别</span>
                    <strong>{group.assetClass}</strong>
                  </div>
                  <div>
                    <span>市场数</span>
                    <strong>{group.markets.length}</strong>
                  </div>
                  <div>
                    <span>市场类型</span>
                    <strong>{group.marketTypes.join(" / ") || "未识别"}</strong>
                  </div>
                  <div>
                    <span>最低置信度</span>
                    <strong>{Math.round((group.primaryConfidence || 0) * 100)}%</strong>
                  </div>
                </div>

                <div class="group-chip-row">
                  {#each group.venueTypes as venueType}
                    <span class="group-chip">{venueType}</span>
                  {/each}
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
            {/each}
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
              <strong>发现层来自 slipstream</strong>
              <p>这里看的不是最终规则，而是从 slipstream 导入的市场清单，先按 base/quote family 聚成候选组。</p>
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
                    {#if remoteSources.length === 0}
                      <option value="">暂无可用源</option>
                    {/if}
                    {#each remoteSources as source}
                      <option value={source.id}>{source.label} ({source.project || source.id})</option>
                    {/each}
                  </select>
                </label>
                <label>
                  <span>远端地址</span>
                  <input value={remoteSources.find((item) => item.id === selectedSourceId)?.url || "未配置"} disabled />
                </label>
              </div>

              <div class="sync-actions">
                <div class="button-row">
                  <button class="sync-button" on:click={() => syncPresetSource()} disabled={syncState === "loading" || !selectedSourceId}>
                    {syncState === "loading" ? "同步中…" : "同步当前源"}
                  </button>
                  <button class="sync-button sync-button--secondary" on:click={syncAllPresetSources} disabled={syncState === "loading" || remoteSources.length === 0}>
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
