# Market identity audit — 2026-10-23 (Beijing)

Covered **19,586 unique markets /20 platforms**,20,746 evidence rows. Full discovery inventory plus registry override union; all79 resolve batches succeeded (maximum250,two workers). Every key has a disposition, result and raw source/chain evidence in local dispositions.jsonl. Initial states:18,395 resolved,1,068 unresolved,123 ambiguous. Resolved/unflagged is not independent certification of product identity or comparability.

Fresh discovery cached at2026-10-23T01:04:12.313380447Z,no cache error. All14 main fetches succeeded, with URL/time/size manifest. Official supplements: Extended404 perpetuals, all493 gTrade pairs,184 GRVT instruments,126 GMX pools and1,056 Gate futures. All3,602 official Bitget spot keys and854 USDT perpetual keys were present. Separate official Bybit requests independently confirmed LMTUSDT/MXLUSDT/PGUSDT stock type, LinearPerpetual and USDT settlement. Bootstrap sourced full names agree. Consumer sample endpoints still api.example.com; real feedback unavailable.

## Incremental findings and fixes

25 added keys,no absent keys. Official Gate adds KIMCHI_USDT; Bitget spot adds RT1USDT and futures add DHRUSDT/TSTOCKUSDT. No Bitget ISIN/name/security-type change; one trading-status change, not a corporate-action event. Previous317 manual rules and38 multi-collateral unknown-settlement guards passed readback. Online9534aed still reports reviewed1942; yesterday's local76cc7f7 rules are protected by online manual overrides pending deployment.

**Six exact rules** written using compact batch API and immediately verified:

- Bitget TSTOCKUSDT: unknown→rwa_stock. Official announcement identifies AT&T, not crypto T. Retain native TSTOCK display/raw symbol, explicit venue stock product, USDT settlement and ambiguous comparison. No global T alias/classification change.
- Bitget DHRUSDT: already stock; add explicit venue stock product boundary and ambiguous comparison.
- Bybit LMTUSDT,MXLUSDT,PGUSDT: already stock; replace generic ticker-only join identities with scoped venue products and ambiguous comparison. Official stock metadata and contract settlement verified independently.
- Bitget RT1USDT spot: unknown→crypto. Current official symbolType crypto,isReality no,ISIN absent,fullName Real Token. Retain RT1 spelling and venue-specific identity, not an inferred rToken wrapper or global chain-equivalence alias. Comparison remains ambiguous pending token-contract identity across venues. Article title confirms Real Token, but body retrieval was unavailable; category proof uses the fresh official API, not unsupported article-body claims.

Sources: [Bitget DHR/TSTOCK stock announcement](https://www.bitget.org/support/articles/12560603898581), [Bitget official futures](https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES), [Bitget official spot catalog](https://api.bitget.com/api/v3/market/instruments?category=SPOT), [Bybit official LMT instrument](https://api.bybit.com/v5/market/instruments-info?category=linear&symbol=LMTUSDT), [Bybit MXL](https://api.bybit.com/v5/market/instruments-info?category=linear&symbol=MXLUSDT), [Bybit PG](https://api.bybit.com/v5/market/instruments-info?category=linear&symbol=PGUSDT). Evidence and before/after payloads retained in reviewed_registry.evidence.json,repair-plan.json,online-write.json and official-bybit-new-stocks.json. No credentials saved. Order/lot size is not a price multiplier.

## Remaining leads and limits

New Variational CVNA/ZCSH still have generic crypto identities without independent issuer/contract proof; keep review leads, no ticker/price-only merge. Gate KIMCHI and new OKX X-prefixed spot products remain unknown/ambiguous pending primary token/issuer metadata. Existing xyz:MUU,xyz:TTWO,Lighter TUR30,io:MOONSHOT,Variational MDB and Binance wrapped-looking/THB keys remain pending. Entropy pre-IPO directory and TradeXYZ documentation index rechecked without individual MOONSHOT/MUU/TTWO proof. Historical wrapping/chain/collision flags retain raw evidence; daily structural coverage is not independent identity verification of every row.

## Validation and delivery

Go race suite,go vet,Linux amd64 build and full20,746-row replay across two successful local syncs pass. Added regression guards native TSTOCK/RT1 names, different stock/crypto classification, ambiguous comparisons and unit1; no global stock/crypto reclassification. Local replay wall934ms/CPU911ms,reader p50 750ns/p99 2,542ns,heap142.4MB. These are local measurements,not production CPU/event-loop results. No runtime hot-path/cadence/drop/freshness policy changes.

After a successful sync started after the final write,all**323 prior/new rules** read back without errors; asset readbacks retained. No overwrite/repeated writes or sync pause. Local reviewed data,evidence and regression committed. No push/deployment/restart performed. Preexisting untracked .automation-state/logs preserved and excluded.

Latest write 2026-10-23T01:07:37.040302+00:00; successful post-write sync 2026-10-23T01:09:07.525184865Z → 2026-10-23T01:09:09.283039833Z. Full verification: final-online-verification.json.
