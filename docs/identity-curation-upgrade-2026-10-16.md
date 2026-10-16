# Identity curation upgrade — 2026-10-16

This release prevents automatic discovery from overwriting human corrections and fixes the classification/name/unit errors identified by the full market audit. Registry generation version is **6**.

## What changes

Registry priority, highest first:

1. Human edits in `<MARKET_KIT_RUNTIME_REGISTRY_PATH>.manual.json`.
2. The 1,636 reviewed exact-market rules embedded from `identity/reviewed_registry.json`.
3. Current discovery-generated rules.
4. Other embedded fallback rules.

The manual sidecar contains human edits only. It is independent of generated versions and is written with a temporary file, sync and atomic rename. API writes and sync commits serialize through one update mutex. Resolver construction happens before the brief reader-lock swap; requests keep using the previous complete snapshot during construction. Discovery aggregation and resolver constructors own their normalization copies.

`POST /api/v1/registry/overrides/batch` accepts `{"items": [...]}` with 1–500 existing override request objects and the same admin authorization as the single endpoint. Every item is validated before persisting anything; successful batches normalize, persist and rebuild once. Responses contain `count` and normalized `overrides`, without a full registry. Identical batches do not rewrite files or rebuild the resolver. Existing single-market and asset-category endpoints remain available. Global category changes are rejected for known ticker collisions; use exact-market overrides instead. Explicit category corrections also get matching default asset/underlying IDs, rather than inheriting another class's IDs.

Discovery selects the strongest metadata for each exact venue/symbol/type before generation. Explicit venue stock/commodity evidence beats generic downstream `crypto`; equally supported conflicting identities stay ambiguous. An unclassified spot market cannot inherit a stock ID just because its ticker collides with a stock. Strong coin metadata beats stock/commodity ticker guesses, except known metal-backed tokens PAXG/XAUT/XAGT.

Extended and gTrade now have direct official metadata collectors. Extended preserves ADR and `_24_5` product names, category and activity state. gTrade includes stock, index, commodity and forex groups omitted by older crypto-only feeds; catalog membership alone is not presented as proof that a legacy contract is currently tradable. Forex is not guessed to be crypto.

Remote discovery responses have an explicit **32 MiB** limit, detected by reading at most limit+1. Previously the 4 MiB reader silently truncated valid inventories and then reported invalid JSON. Oversize responses now produce an explicit size error; the previous valid snapshot is retained.

Scaled crypto normalization updates the canonical symbol, inferred ticker IDs, comparison key and unit conversion together. Explicit split/rebase multipliers and custom product IDs retain their meaning. Numeric index names such as US100 are not token denominations. HIP-3 keeps the source quote asset instead of replacing it with a CEX quote; USDH is not relabeled USDT. Automatic abbreviation/prefix guessing no longer creates global aliases.

Important exact identities:

- XYZ `xyz:SKHX` and Lighter `SKHYNIXUSD`: verified one Korean ordinary share, quoted in USD, normalized to SKHYNIX.
- `SKHY`: ADR; remains distinct.
- Bitget `RSKHXUSDT`: Leverage Shares 2x Long SK Hynix ETP, ISIN US88340C2632; separate product identity and `leveraged_etp` kind.
- Unverified USOIL is not globally aliased to the USO ETF.

The reviewed file contains 1,468 Bitget, 99 Extended, 36 Lighter, 12 gTrade, 7 Binance, 4 Bybit, 4 Variational and 6 Hyperliquid overrides. `registry/reviewed_registry.evidence.json` records exact keys and public official sources. These rules do not assert current tradability, make same-name products interchangeable, or constitute execution authorization. Other audit flags still need evidence; ambiguous identities remain closed to automatic comparison.

## ECS deployment

The owner performs deployment. Preserve `.env` and back up the configured runtime data directory before updating. Do not replace the runtime file with the reviewed file: the reviewed rules are already embedded in the binary.

```bash
git pull --ff-only
./pm2-restart.sh
```

The existing script builds the Go backend and records the Git commit in `/api/v1/version`. Deploy the frontend separately through its existing pipeline so its numeric-index guard is updated too; the backend fixes do not depend on rebuilding the frontend.

On the first startup, a pre-v6 generated snapshot is rebuilt rather than frozen as human policy. Explicit legacy fractional split/rebase settings, expiry/contract settings and prohibited-comparison overrides are migrated into the sidecar. Other historical manual edits cannot be reliably distinguished from generated data in the old mixed file: keep the backup and re-submit any additional known human corrections through the existing write API. Do not import the entire old generated file into the sidecar.

After restart:

```bash
curl -fsS http://127.0.0.1:18120/api/v1/version
curl -fsS http://127.0.0.1:18120/api/v1/auto-sync
curl -fsS 'http://127.0.0.1:18120/api/v1/resolve?exchange=bitget&symbol=RAATUSDT&marketType=spot'
curl -fsS 'http://127.0.0.1:18120/api/v1/resolve?exchange=lighter&symbol=US100&marketType=perpetual'
curl -fsS 'http://127.0.0.1:18120/api/v1/resolve?exchange=lighter&symbol=1000PEPE&marketType=perpetual'
curl -fsS 'http://127.0.0.1:18120/api/v1/resolve?exchange=hyperliquid&symbol=xyz:SKHX&marketType=perpetual'
curl -fsS 'http://127.0.0.1:18120/api/v1/resolve?exchange=bitget&symbol=RSKHXUSDT&marketType=spot'
```

Expect generation version 6, `reviewedOverrideCount=1636` and a `manualPath` in auto-sync status. `manualOverrideCount` starts at zero unless older explicit product settings were migrated or new edits were made. `suspiciousCryptoCount` is now the full count; `suspiciousCryptoAssets` remains a maximum-25 preview.

Verify after a successful sync and after restarting again. RAAT should be equity AAT; US100 must stay US100; Lighter 1000PEPE should be PEPE with a 1,000 quantity multiplier and 0.001 price multiplier. XYZ SKHX and Bitget RSKHX must have different underlying IDs. Preserve the sidecar in every future backup. Rolling back to the old binary does not provide manual-priority guarantees; keep the pre-deployment runtime backup if rollback is necessary.

## Validation and performance

- Go suite, race detector, frontend tests/build and Linux build.
- Every embedded reviewed rule resolves to its expected exact identity.
- Manual edits survive generated-version changes, two sync cycles and restart; concurrent edits and discovery readers preserve immutable snapshots.
- A valid >4 MiB discovery payload succeeds; >32 MiB explicitly fails.
- Real official collectors returned 493 gTrade and 403 Extended market records during this run.
- Local replay: 18,411 input records, all 1,636 reviewed identities retained after two syncs. On an Apple M4, one sync used about 0.79–0.81 CPU-seconds / 0.69 seconds wall time. A 500-edit batch took about 146–153 ms; identical no-op about 2.4–2.6 ms; direct reader P99 during an administrative update about 1–2.3 microseconds. These are local fixture measurements, not production service measurements.
- Indexed resolver microbenchmark before: 365–396 ns/op; after: 371–454 ns/op; both 400 B/op and 4 allocations. Short samples do not establish a meaningful throughput improvement or regression. This release targets correctness and repeated administrative work, not a claim of lower realtime CPU.

No ticker cadence, buffering, drop policy or synchronization interval was changed. Allowing previously truncated inventories and adding two metadata endpoints increases the amount of valid data processed during a sync and can increase a sync burst. Request/size limits remain bounded; reader access stays indexed. The operator should confirm live CPU, memory, sync duration, source coverage and freshness after deployment. There is no claim of measured production CPU improvement.

To repeat the complete online read-only audit, save `/api/discovery/current` and `/api/v1/registry` responses and run:

```bash
python3 scripts/audit-online-markets.py --inventory /tmp/inventory.json --registry /tmp/registry.json --output-dir /tmp/market-kit-audit
```

The optional full-inventory local replay uses a saved discovery/current wrapper and never accesses exchange networks:

```bash
MARKET_KIT_TEST_INVENTORY=/tmp/inventory.json go test ./server -run '^TestFullInventoryReplay$' -v
```
