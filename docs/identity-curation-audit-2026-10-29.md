# Market identity audit — 2026-10-29 (Beijing)

Covered **20,196 unique market keys /20 platforms**, 21,920 evidence rows. Current discovery+registry union,supplemented with official Extended416 perpetuals,all493 gTrade pairs,GRVT184,GMX126 pools,Gate1065 contracts and40 Bybit dated futures. All81 resolve batches succeeded(max250,at most2 concurrent). Every key has a disposition and source/chain evidence. Initial states:{'resolved': 18985, 'unresolved': 1088, 'ambiguous': 123}. Structural resolution is not independent certification for cross-venue execution.

All14 primary fetches succeeded;inventory cached2026-10-29T00:58:11.844679096Z,no cache error. Official Bitget spot/futures,Bybit linear,Hyperliquid core+allHIP3 catalogs and QFEX refdata saved with source/time evidence.24 added union keys,4 absent keys;no delisting inferred from absence. Online4e897dd/schema8,2254 reviewed rules: yesterday's c1f29dd code/data still pending deployment. Prior630 exact rules and39 absent-settlement guards passed initial readback.

## Systemic QFEX settlement error —233 exact markets

Direct official QFEX refdata has margin_asset USDC independently of native reference quote.233 resolved markets had incorrect settlementAsset(generally USD,also new NIFTY INR). Corrected only settlement for existing markets,retaining product identity,underlying,comparison key/status,raw symbol,native quote and multipliers. NIFTY-INR additionally gets scoped index:qfex:nifty-inr product/comparison,unknown enum class(no index enum),index_perpetual,ambiguous comparison and USDC settlement. It is an INR-reference index,not an INR collateral product or crypto token. No global currency/asset alias edits. YUBICO's previously correct USDC rule remained unchanged.

Collector now preserves official margin_asset in InstrumentMetadata.SettlementAsset. Generated QFEX overrides use this explicit field only when metadata source=qfex,after other product guards;missing/untrusted metadata retains existing fallback. NativeQuoteCurrency stays separate. Generated schema bumped8→9 to invalidate stale generated settlements on next deployment. Nonstable reference quotes still require reviewed exact rules;did not broaden generic quote eligibility or infer index constituent/FX ratios. Reviewed233 exact rules make today's online repair persistent and independently deployable. Historical/manual rule precedence remains intact.

## Other7 exact repairs

Bitget CLS,DASHSTOCK,ITUB,NU,PBR perpetuals: official indexed support announcement identifies stock contracts and exact API confirms isRwa YES,perpetual,USDT margin. DASHSTOCK and ITUB were unknown;other three had generic eligible stock identities. Scoped all5 venue products with ambiguous cross-venue units;no global DASH/NU crypto reclassification. isRwa alone was not used. The support page direct fetch returned403;official search-index excerpt supplied stock-class evidence only. No unsupported full-article or issuer/ADR claims.

Extended BR-USD and BR/USD: official description Bedrock,categoryCrypto. Display slash variant was unknown while native symbol was crypto. Both now share a scoped crypto:extended:bedrock identity,ambiguous comparison,without inferring token address or cross-venue join.

One batch240 exact writes used compact=1 and strict prewrite equality. Immediate readback passed. Final **870 audited rules** passed after successful auto-sync started later than latest write2026-10-29T01:02:42.784686Z;no sync overwrites. Separate asset and resolve readbacks retained;asset-level canonical metadata alone is not cross-market identity proof.

## Status changes and pending evidence

New Hyperliquid xyz:NU already carries official isDelisted true;deployed schema8 correctly produces scoped inactive product and prohibited comparison. Official inactive count202→203;the four prior reactivation leads remain under manual comparison prohibition. No new reactivation changes. Bybit EPIC/MITO/PTB/TMX remain LinearPerpetual but now have nonzero deliveryTime1793350800000:record scheduled lifecycle metadata,do not reclassify as dated futures merely from deliveryTime.40 actual LinearFutures unchanged. Gate and Bitget VELODROME contracts absent from current official catalogs;no historical deletion solely on absence.

Bitget RAURE spot returns with no changed ISIN evidence. Variational AMC/BAC appear only in registry,without current discovery rows or independent product metadata;record identity proof gap,do not assert stock/crypto equivalence or trade eligibility certification. Binance new B-suffixed spot wrappers remain unknown/ambiguous pending issuer/chain/ratio evidence. Retained r-prefixed aliases,historical wrapping collisions,Variational/OKX product proof and placeholder consumer-feedback endpoints remain open. No fake live feedback coverage.

## Validation and runtime implications

Full Go suite,go vet,Python audit tests,final collector/curation regression,server build and diff check passed. New tests preserve EUR reference versus USDC margin and reject unrelated-source margin metadata. Full21920-row local replay passed:sync wall865.6ms,CPU888.6ms,reader p50/p99625/2250ns,heap152.9MB. Single local run,not production profiling or a claimed performance improvement. Changes run at discovery/sync cadence;no ingest frequency,queue,drop or freshness policy changed. No production restart/push/deploy performed.

Sources: [QFEX official refdata](https://api.qfex.com/refdata),[Bitget official stock listing](https://www.bitgetapp.com/support/articles/12560603899056),[Bitget official contracts](https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES),[Extended official markets](https://api.starknet.extended.exchange/api/v1/info/markets),[Bybit official linear catalog](https://api.bybit.com/v5/market/instruments-info?category=linear&limit=1000),[Hyperliquid official metadata](https://api.hyperliquid.xyz/info). Local manifests retain timestamps,fetch outcomes and per-key dispositions without credentials.
