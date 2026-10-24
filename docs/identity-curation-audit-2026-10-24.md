# Market identity audit — 2026-10-24 (Beijing)

Covered **19,980 unique market keys /20 platforms**,22,132 evidence rows. Full discovery inventory plus registry overrides, supplemented with official Extended404 perpetuals,all493 gTrade pairs,184 GRVT instruments,126 GMX pools,1,061 Gate futures, and Bybit40 dated futures under their actual future type. All80 final resolve batches succeeded (maximum250,two workers). Daily delta395 added coverage keys,one absent Binance WALBNB spot key;40 additions represent correct Bybit future-type coverage. Absence is not confirmed delisting. Initial states:18,790 resolved,1,067 unresolved,123 ambiguous. Every key has raw source/chain evidence and disposition in local dispositions.jsonl; structural coverage does not independently certify economic equivalence.

Fresh discovery cached2026-10-24T00:59:58.135592312Z,no cache error. All14 main fetches succeeded with URL/time manifests. Registry/health/inventory were fetched at different instants,so counts differ during sync. All3,594 official Bitget spot keys and857 USDT perpetual keys covered. All3,019 official stock spot records still resolve as stock; generic metadata did not override those classes. No official Bitget identity/security-ID drift;three trading-status changes. Official catalog lacks8 previous crypto spot symbols; no automatic deletion. Consumer sample sources remain api.example.com,so live consumer feedback is unavailable.

## Critical repair: dated futures treated as perpetuals

Bybit official category-linear catalog contains916 instruments,including**40 LinearFutures**. The collector previously marked every linear instrument perp and discarded contractType,deliveryTime and settleCoin. All40 old perpetual keys were eligible with token-level comparison keys,which incorrectly allowed a dated future to share comparison identity with a perpetual or another expiry.

**80 exact online rules** now cover those40 contracts:

- Wrong legacy perpetual keys: comparison prohibited,dated-future product identity and official expiry/settlement retained. A resolved API status still requires the consumer to check comparisonStatus.
- Correct future keys: separate contract-specific identity/key,official expiry and USDT settlement,comparison ambiguous pending independent cross-venue contract equivalence. No expiry or multiplier inferred from price/name.

Tokens include BTC,ETH,DOGE,HYPE,MNT,SOL,XAUT and XRP across official Nov/Dec2026 and2027 maturities. XAUT remains commodity. Underlying token identity alone does not permit joining different expiry/instrument products.

Primary source: [Bybit official instrument catalog](https://api.bybit.com/v5/market/instruments-info?category=linear&limit=1000). Complete official pages and timestamps retained in official-bybit-linear.json. Every native symbol,type,base/quote,settlement and expiry used in the repairs is retained in reviewed_registry.evidence.json and repair-plan.json.

## New stock products

Eight more scoped rules: Gate DHR_USDT,HLT_USDT,MDT_USDT,MSCI_USDT,MXL_USDT; Bitget CSXUSDT,PGUSDT,SPOTUSDT. All already classified stock,so these are product-boundary refinements,not eight class errors. Official Gate stock type and Bitget listing announcement checked; comparison remains ambiguous with venue-scoped products. SPOT is Spotify's stock contract,not a spot market-type hint. Contract/order size is not price multiplier; unitMultiplier1 retained. No global ticker reclassification.

Sources: [Gate official futures](https://api.gateio.ws/api/v4/futures/usdt/contracts),[Bitget SPOT/PG/CSX stock announcement](https://www.bitget.com/support/articles/12560603898699),[Bitget official futures](https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES).

## Systemic prevention

- Bybit collector preserves official contract type,expiry and settlement in additive instrumentMetadata fields,and emits future for dated contracts.
- Generator accepts Bybit future rows,gives authoritative future metadata priority over generic classifications,and uses separate dated-product comparison identities. Missing or conflicting expiry/settlement fails closed. Conflicting official snapshots cannot choose a valid contract by source order.
- A date-shaped Bybit name can only quarantine an unverified derivative; it never supplies an inferred expiry,settlement or comparison eligibility. Legacy wrong-type keys are prohibited.
- Non-pruning sync also quarantines retained stale dated overrides absent from the current catalog. Generic token aliases cannot fill future identity/settlement fields. Resolver no longer guesses settlement from quote for dated-product contract types.
- Generated schema bumped6→7. Existing manual/reviewed rules remain higher-priority. Code is local pending user deployment; online88 scoped rules provide current protection.

## Validation and performance

Go race suite,go vet,Linux amd64 build and full22,132-row replay across two successful local syncs pass. Regression tests cover collector serialization and future/perpetual distinction;separate expiries;unknown settlement;official-vs-generic evidence ordering;conflicting expiry;and retained stale overrides. An independent replay of **all40 official futures through the generated registry without reviewed overrides** verifies exact expiry,settlement and noneligible cross-venue status.

Changes run at discovery/sync cadence,not ticker ingest. Pattern compiled once;no per-message clone/sort/serialization,queue or frequency change. Resolver adds only constant-time contract-type checks. Same-snapshot local runtime measurements: baseline sync CPU873ms/wall932ms,reader p50/p99 708/3,666ns;after CPU857ms/wall775ms,reader917/3,791ns. Single samples are noisy and do not establish a production speedup. Heap157→174MB includes88 additional reviewed rules. Production CPU/event-loop metrics were not available; no production CPU claim. Freshness cadence unchanged,no queue/drop semantics changed;unverified/wrong-type futures deliberately lose comparison eligibility.

## Persistence and remaining leads

**88 new online rules** (80 dated-product keys +8 stock keys),immediate readback passed. After a successful new sync,all**411 prior/new rules** matched expected identity,type,expiry,settlement and comparison fields. Assets read back and retained. No overwrite,repeated write or sync pause. Online remains9534aed with schema6 and reviewed1942; deploy current local code to prevent recurrence as contracts roll.

Asset directory expanded5,021→7,801;2,782 new aliases include many r-prefixed crypto names.2,759 are not used as current override base symbols. Active Bitget official stock market classes checked correct;source/history alias cleanup remains pending and no global aliases were deleted on count/name alone. Many355 initial added market keys are returning Gate leveraged spot products;these remain separately unresolved/ambiguous,not collapsed into their underlyings. Broader identity/chain/unit leads remain,including external-deployer products,pre-IPO names,Variational CVNA/ZCSH/MDB,OKX X-prefixed spot and quote-source conflicts. Daily structural resolution is not independent verification of all those products.

No deployment,restart or push performed. Local tracked changes committed;preexisting .automation-state/logs preserved and excluded. No credentials saved.

Latest write 2026-10-24T01:04:36.014860+00:00; successful post-write sync 2026-10-24T01:06:07.525331042Z → 2026-10-24T01:06:15.851382561Z. Complete checks: final-online-verification.json.
