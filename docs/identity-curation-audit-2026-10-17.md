# Daily identity curation — 2026-10-17

The complete discovery/registry union plus current official Extended, GRVT and
GMX catalog supplements covered 19,696 unique exchange/raw-symbol/type keys
across 20 platforms. Every key has an audit disposition in
`.automation-state/market-kit-audit/full-2026-10-17/`; successful resolution is
not independent identity verification or permission to compare/trade products.

Online exact-market writes corrected:

- 53 GRVT categories: 40 EQUITY contracts classified as crypto, 12 CRYPTO
  contracts classified unknown, and COPPER classified unknown. Official
  `all_instruments` supplies `asset_class`. Corrected identities are venue scoped
  and comparison remains ambiguous until economic units are independently known.
- Seven 1000CAT markets whose crypto category had inherited Caterpillar's equity
  IDs through a global scaled alias. Preserve the existing 1000 unit conversion,
  use crypto IDs, and withhold unverified cross-venue comparisons.
- Bitget `RBZUSDT`: official BOSS ZHIPIN stock, ISIN `US48553T1060`, previously
  assigned Brent commodity identity. It now has an independent equity/ISIN ID;
  other BZ markets are unchanged. Official Bitget categories were checked for
  2,913 existing stock markets, with this single category mismatch corrected.

`xyz:XYZ100` was additionally quarantined with an independent equity-index ID,
multiplier 1 and ambiguous comparison. The official trade.xyz specification
identifies an equity index, not 100 shares/units of XYZ. The deployed pre-fix
normalizer still folds its canonical name into XYZ. This release protects
XYZ100 in both Go and frontend normalization and embeds the correct native
canonical. **The online manual safety override must be updated after deploying
this guard**, since manual overrides take precedence over reviewed data.

The root cause fixes add official GRVT metadata at the existing low-frequency
bootstrap cadence and prevent a scaled crypto market from borrowing inferred
IDs from a same-name stock. Explicit custom product IDs, rebases, raw instrument
names, quotes and other product boundaries remain preserved. Reviewed exact
rules now total 1,698. No global stock/crypto class reassignment was made online.

Go race tests, vet, frontend tests/build and Linux amd64 compilation passed.
A replay of 19,505 input rows (including official evidence supplements) retained
reviewed rules across two automatic syncs. The collector adds one metadata HTTP
request per bootstrap cycle; source latency/timeouts may lengthen the refresh.
No realtime ticker processing, configured cadence or message-drop policy changed.
Production code has not been deployed by this maintenance run.

## Remaining coverage / deployment work

- gTrade's official `trading-variables` endpoint failed TLS with Python and curl.
  There were no gTrade records in today's discovery snapshot. All 459 existing
  registry keys were resolved, but current stock/index/commodity/FX catalog
  coverage cannot be claimed. Bootstrap currently hides an individual collector
  failure when another collector succeeds; improve per-source health separately.
- Historical overrides, wrapped assets, chain-dependent identities and remaining
  class/ID leads require evidence. Many diagnostic flags are deliberately
  ambiguous unknown IDs rather than confirmed errors. Do not bulk-enable them.
- Deploy backend and frontend normally. Then submit the bounded camelCase JSON
  request at `.automation-state/market-kit-audit/full-2026-10-17/post-deploy-index-request.json`
  to `POST /api/v1/registry/overrides?compact=1` using the existing authorized
  access. Verify `xyz:XYZ100` keeps canonical `XYZ100/USDC`, multiplier 1 and
  ambiguous comparison across a successful sync. Do not repeat this write against
  old commit `8f526f5`, which still folds the name.

Evidence: [GRVT catalog](https://market-data.grvt.io/full/v1/all_instruments),
[Bitget catalog](https://api.bitget.com/api/v3/market/instruments?category=SPOT),
[Binance contracts](https://fapi.binance.com/fapi/v1/exchangeInfo), and
[trade.xyz specifications](https://docs.trade.xyz/perpetuals/specifications-and-schedules/specification-index.md).
Raw snapshots, timestamps, precise requests and sync readbacks are retained in
the local audit directory; credentials are not stored there.
