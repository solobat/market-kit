# Market identity audit — 2026-10-22 (Beijing)

Covered **19,561 unique markets /20 platforms**, 20,201 evidence rows. Full discovery inventory plus registry override union; all79 resolve batches succeeded (maximum250, two workers). Each market has a disposition and retained raw source/chain/instrument metadata in local dispositions.jsonl. Initial statuses:18,370 resolved,1,068 unresolved,123 ambiguous. Structural success does not independently certify identity or economic comparability.

Fresh inventory cached at2026-10-22T01:02:00.158171246Z with no cache error. All14 main fetches succeeded; URL/time/size in fetch-manifest.json. Supplemented official Extended404 perpetuals, all493 gTrade pairs,184 GRVT instruments,126 GMX pools,1,055 Gate futures. All3,601 official Bitget spot and852 USDT perpetual keys were covered. Bitget futures separately fetched with timestamp. Configured consumer sample endpoints still api.example.com; real consumer feedback remains unavailable.

## Incremental results and repairs

30 added coverage keys, no absent keys. Prior307 repairs read back without errors, including38 unknown-settlement multi-collateral guards. Online commit9534aed and reviewed count1942 confirm prior data/helper changes and newer sourced-name enrichment are deployed. Working tree initially contained only preexisting untracked .automation-state and logs; these remain excluded from commit.

**10 exact stock market rules** written with compact batch API and immediately verified:

- Gate FASTRETAILING_USDT, TAIYOYUDEN_USDT, IBKR_USDT, VLO_USDT: official contract_type stocks. Retain native raw names, venue product IDs and ambiguous comparisons; no automatic Japanese ordinary-share/ADR equivalence or FX/ratio inference. Previous classification was already stock, so these are boundary refinements, not four class errors.
- Bitget ABBVUSDT, EBAYUSDT, ELICEUSDT, HLTUSDT, IBKRUSDT, MSCIUSDT: six official stock perpetuals. **ELICE was unknown** and is now stock; other five stock classes were already present, and receive explicit venue product identity. Official listing announcements confirm stock class and USDT settlement. isRwa YES alone was not used as stock proof. Order increments/contract lot sizes are not price multipliers. All comparisons remain ambiguous.

Sources: [Gate official contracts](https://api.gateio.ws/api/v4/futures/usdt/contracts), [Bitget five stock listings](https://www.bitget.com/support/articles/12560603898430), [Bitget ELICE listing](https://www.bitget.com/support/articles/12560603898332), [Bitget official contracts](https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES). Exact before/after fields, source evidence and current metadata retained in repair-plan.json, online-write.json and reviewed_registry.evidence.json. No global ticker/class alias edit.

## Prior leads and limits

Bitget's [Axis Robotics spot listing](https://www.bitget.com/support/articles/12560603898107) identifies AXIS as a crypto project on Base with token contract0x5bE75e3b0aFf5D1A042A5363Ae36AcB1d6d755ea; current Bitget metadata and Bybit sourced fullName agree. Current AXIS rules already classify crypto; no redundant writes. This closes the Bitget issuer/chain lead, not every venue's independent token equivalence. No new Bitget ISIN or issuer-name changes; only two trading-status changes, not corporate-action evidence.

New xyz:MUU and Lighter TUR30 still show crypto without enough product identity/unit proof. Lighter official orderBookDetails confirms TUR30 market4647 is an active external-deployer perpetual with multiplier1, but supplies no issuer/index definition; supplementary snapshot and fetch timestamp are retained. Retain as important review leads; do not infer leverage or index identity from ticker. Existing xyz:TTWO, io:MOONSHOT, Variational MDB and OKX XNFLX still need deployer/issuer/contract evidence. TradeXYZ's official US stock session page was checked but lists no individual TTWO/MUU instrument proof. New Binance B-suffix products and THB spot keys remain pending independent identity/quote-support evidence, not inferred wrappers or delistings. No price/name-only cross-platform joins.

## Validation and delivery

Go race suite, go vet, Linux amd64 build, and two-sync complete inventory replay pass. Added regression: Gate/Bitget IBKR keep separate comparison identities despite equal display symbol; ELICE is stock/ambiguous; FASTRETAILING retains native base/unit1. No runtime hot-path code or scheduling/drop/freshness policy changed. Offline replay sync wall612.5ms/CPU703.3ms; reader p50 1,333ns/p99 4,375ns; heap173MB. These are local test figures, not production CPU/event-loop measurements.

After a successful new automatic sync, all**317 prior/new rules** read back correctly, with asset readbacks retained. No overwrite, repeated write or sync pause. Local reviewed data/evidence and regression are committed; no push, deployment or restart performed.

Latest write 2026-10-22T01:05:46.601140+00:00; successful post-write sync 2026-10-22T01:05:54.66373521Z → 2026-10-22T01:06:03.809843311Z. Full readback: final-online-verification.json.
