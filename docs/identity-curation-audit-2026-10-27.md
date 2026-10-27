# Market identity audit — 2026-10-27 (Beijing)

Covered 20,145 unique market keys /20 platforms, 21,878 evidence rows; all81 resolve batches succeeded (250 per batch, maximum2 concurrent). Full current discovery +registry union supplemented by official Extended404 perpetuals, all493 gTrade pairs, GRVT184, GMX126 pools, Gate1061 futures and40 Bybit dated futures. Every key has a disposition and preserved source/chain evidence. Initial states: {'resolved': 18931, 'unresolved': 1091, 'ambiguous': 123}. Resolved/unflagged means structural checks passed, not independently certified trading equivalence.

Fresh online inventory cached2026-10-27T00:57:52.086143454Z;14 primary requests succeeded. Official supplementary catalogs include Bybit919 instruments,Bitget860 perpetuals,Hyperliquid11 catalogs,and QFEX direct refdata. URL/time evidence saved locally. 43 added keys,20 absent keys. Absence is not a delisting instruction;expired23Oct Bybit legacy keys and temporary snapshot omissions were not recreated.

Online deployment aff2e8b now uses generated schema8 and2253 reviewed rules. Existing618 audited manual rules passed initial readback,including39 absent-settlement CFD guards. Schema8 dated futures and inactive guards are now deployed. Hyperliquid206 inactive flags unchanged;Bybit40 dated contracts unchanged;Gate official contract inventory/type fields unchanged. Bitget spot has no changed identity fields and one absent RCRPJYUSDT;no corporate-action ratio or deletion inferred. New official Bitget perpetuals EIDO/EQT/STM,Bybit ABBV/AGPU/TTMI and Hyperliquid xyz:ANT captured. Binance official EQUITY tags checked for ABBV/EIDO/EQT/STM. Existing deployed pre-IPO discovery correctly scopes xyz:ANT to Anthropic with ambiguous product comparison;no name-only join to io:ANTH.

## Executed online fixes

1. gTrade DAR/USD: deployment removed yesterday's typed-write blocker. Fresh official pair/group still identifies altcoins. Replaced temporary unknown/prohibited quarantine with crypto:gtrade:dar,crypto_cfd,multi_collateral_cfd,ambiguous comparison and absent settlement. Same-name equity remains separate. No global DAR alias/class change.
2. QFEX YUBICO-SEK: direct official refdata identifies EQUITY,baseYUBICO,quoteSEK,marginUSDC. Fixed unknown classification and incorrect SEK settlement: rwa_stock,scoped equity:product:qfex:yubico-sek,underlying equity:ticker:yubico,comparison ambiguous,settlementUSDC,native SEK quote unchanged. No ISIN or external stock equivalence inferred. Generic curation excludes nonstable quote equities;added reviewed exact rule without broadening quote eligibility.

Writes use compact=1 and exact keys;before/after payloads retained,without credentials. DAR was restored first;the subsequent combined batch retained that same correct DAR rule while adding YUBICO. No repeated overwrite loop. Immediate readbacks passed. Final619 audited rules have no identity mismatches. Fresh successful sync verification is recorded in final-online-verification.json;report finalized only after its start exceeded latest write.

## Local maintenance and validation

Reviewed YUBICO rule +official evidence committed with regression protecting SEK quote versus USDC settlement,scoped comparison key and ambiguous status. Relevant identity/curation/server tests passed;git diff --check passed. An initial test command included registry,which is a JSON-only directory and not a Go package;corrected package list passed. No hot path algorithm,cadence,buffering or message policy changed;no CPU/freshness improvement claimed. No deployment/restart/push performed by this audit.

## Coverage limitations and pending evidence

New Binance-web3 wrapped on-tickers retain chain evidence and unresolved state rather than being merged by symbol. QFEX stock mapping is venue scoped;no official issuer/ISIN mapping asserted. Arcus KPEPE remains native product;no inferred1000 multiplier. Remaining historical wrapping collisions,retained r-prefixed aliases,Variational/OKX product proof and consumer feedback placeholder URLs remain open. New eligible stock rows were structurally checked against available official categories,not independently certified for cross-venue execution. Full per-market results and flags retained locally;no unverified flags automatically converted into repairs.

Sources: [QFEX official refdata](https://api.qfex.com/refdata),[gTrade official variables](https://backend-arbitrum.gains.trade/trading-variables),[Binance official futures](https://fapi.binance.com/fapi/v1/exchangeInfo),[Bitget instruments](https://api.bitget.com/api/v3/market/instruments?category=SPOT),[Hyperliquid official metadata](https://api.hyperliquid.xyz/info).
