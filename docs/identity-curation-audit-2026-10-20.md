# Market identity audit — 2026-10-20 (Beijing)

Full online inventory and registry override union: **19,756 keys / 20 platforms**, 22,109 evidence rows; all 80 resolve batches succeeded (250 maximum, two workers). Official Extended, all gTrade groups, GRVT, GMX, Gate futures, Bitget spot/futures and Binance/Hyperliquid catalogs supplemented discovery. All 846 official Bitget USDT futures are covered. Snapshot inventory timestamp: 2026-10-20T00:57:31Z. Initial resolve states: 18,567 resolved, 1,066 unresolved, 123 ambiguous. These are structural checks, not independent certification of price comparability.

Compared with yesterday: 138 added coverage keys, 56 absent Gate spot keys. Absence and 871 Bitget status changes are not automatic delisting or security-conversion evidence. Bitget spot adds 107 stock records; missing ISIN/ADR conversion evidence is retained rather than guessed.

## Verified repairs

19 exact market rules written using compact batch API; immediate readback passed:

- Six Gate stock contracts (GRAL, NU, PBR, VICR, VIST, YPF): explicit venue product boundaries. Existing heuristic stock classification was not counted as a classification error.
- Twelve Bitget stock perpetuals: TWTSM, TWDELTA, TWASX, TWUMC, NANYA, WINBOND, YAGEO, WALSIN, CSTOCK, MDT, BP, VLO. Ten previously unclassified; MDT/VLO identity boundaries refined. Taiwan references remain distinct from US ADRs and wrapped tokens. All comparison statuses remain ambiguous pending contract-unit evidence.
- Bitget RNHPUSDT: official ISIN US42226B5012 → US42226B6002. Issuer confirms Class A conversion into common stock **1:1**, effective 2026-10-19 09:30 ET. Keep wrapped token product identity distinct; do not apply the historical reverse split or globally rewrite NHP.

Evidence:

- [Gate official futures catalog](https://api.gateio.ws/api/v4/futures/usdt/contracts)
- [Bitget Taiwan stock listing](https://www.bitget.com/support/articles/12560603897858)
- [Bitget four stock listing](https://www.bitget.com/support/articles/12560603898053)
- [Bitget official futures](https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES)
- [Bitget official spot securities](https://api.bitget.com/api/v3/market/instruments?category=SPOT)
- [NHP issuer conversion announcement](https://investors.nhpreit.com/Investors/investor-news/news-details/2026/National-Healthcare-Properties-Announces-Conversion-of-Class-A-Common-Stock-into-Common-Stock/default.aspx)

## Prevention and validation

Read-only audit accepts `--official-bitget` and `--previous-official-bitget`, writes security-metadata-drift.json and per-market identifier-change flags. Distinguishes enrichment, missing identifiers, identifier replacement and trading status changes; never infers split ratios or cross-platform equivalence. Official fetch snapshots and repair evidence retain timestamps. Three Python regression tests, Go race suite, go vet, Linux amd64 build, and full inventory replay across two syncs passed. Offline replay: sync wall 859 ms / CPU 1.032 s, reader p50 667 ns / p99 4,292 ns, heap 217.4 MB. These are local measurements, not production CPU/event-loop results. No realtime runtime/cadence or message dropping changes.

## Coverage limits / follow-up

Each market has a disposition with source/chain evidence in local dispositions.jsonl; initial flags remain review leads even after readback. New io:MOONSHOT and Variational MDB lack sufficient issuer/product-unit proof in this run; retain pending review, no name-only merge. Entropy pre-IPO and pre-listing official directories were checked but did not establish MOONSHOT identity. Existing unresolved/ambiguous and historical collision/wrapping leads remain. Configured sample feedback URLs still use api.example.com; actual consumer feedback coverage is unavailable. No credentials stored.

Online code still 46b2cc0. Yesterday's 790cbb9 is already pushed but Gate collector prevention still needs manual ECS deployment. Today's reviewed data and audit helper are committed locally; no automatic push/deploy. Online manual rules do not depend on this deployment.

Post-write persistence: **True**, 250 prior/new exact rules verified, 0 failures. Successful sync: 2026-10-20T01:07:17.870135506Z → 2026-10-20T01:07:33.855238024Z. Asset readbacks retained.
