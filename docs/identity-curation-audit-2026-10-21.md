# Market identity audit — 2026-10-21 (Beijing)

Covered **19,531 unique markets / 20 platforms**, 20,734 evidence rows. Full online discovery inventory plus registry overrides, supplemented with Extended (404 perpetuals), all gTrade groups (493 pairs), GRVT (184), GMX (126 pools), and Gate futures. All 79 batches succeeded (maximum250, two workers). All current official Bitget spot and 845 USDT perpetual keys were present. Initial states: 18,342 resolved, 1,066 unresolved, 123 ambiguous. Each key has a disposition and source/chain evidence in local dispositions.jsonl. Resolved and unflagged are not independent identity/comparability certification.

Discovery snapshot 2026-10-21T00:59:28.596485024Z; no cache error. All14 fetches succeeded with URL/time in fetch-manifest.json. Existing250 manual rules read back correctly; 38 multi-collateral settlement guards remain intact. Online code **790cbb9** confirms the previously pending Gate collector fix is deployed. Yesterday's 06c0a76 remains local; new metadata helper has not been deployed.

## Repairs

**57 exact overrides** written and read back, then **307 prior/new rules** rechecked after a new successful automatic sync. No errors or overwrite. No repeated rewrite or service restart.

- Extended28 markets: official native `-USD` identity and historical `/USD` override had different classifications. Correct both variants with venue-scoped identity and ambiguous comparison: four stocks/ETFs (NOW, LYTE, BOT, EWZ), 24 crypto references. This corrects23 unknown crypto variants, crypto→stock on four variants, and stock→crypto on BILL/USD. There are56 rules, not56 independent assets. Official directory explicitly identifies ServiceNow, RoboStrategy, Roundhill Photonics & Optics ETF, iShares MSCI Brazil ETF, Billions Network and the crypto projects. No global ticker reclassification.
- Gate MPLX_USDT: stock heuristic was wrong for this Metaplex perpetual, supported by Gate's explicit Metaplex contract announcement and current official contract metadata. Keep venue product crypto identity, ambiguous comparison. Contract lot10 is not a tenfold price rebase; unitMultiplier remains1.

Sources: [Extended official directory](https://api.starknet.extended.exchange/api/v1/info/markets), [Gate Metaplex contract announcement](https://www.gate.com/fr/announcements/article/35422/gate.io-has-added-metaplex-mplx-perpetual-contract), [Gate current contracts](https://api.gateio.ws/api/v4/futures/usdt/contracts). Official metadata and before/after payloads retained locally; no credentials stored.

## Prevention and validation

Read-only audit now accepts `--official-extended`; checks both native and display variants against official category. Captures28 pre-repair official category conflicts, including rows with no discovery evidence that existed only as registry overrides. Unknown official categories are not guessed; conflicting official categories fail closed. This exposes stale explicit rules that ordinary structural scans missed. Reviewed rules protect current cases across sync; no automatic future class rewrite or broad global alias was introduced.

Four Python tests pass. Go race suite, go vet, Linux amd64 build and two-sync full-inventory replay pass; an additional identity regression covers NOW/LYTE/BILL/SUI separator consistency and the MPLX stock/crypto collision. Offline replay sync wall796ms, CPU953ms; reader p50 666ns/p99 1,250ns; heap146.7MB. These are local test measurements, not production CPU/event-loop measurements. No realtime runtime, cadence, drop or freshness policy changed.

## Incremental evidence and limits

16 added coverage keys,241 absent keys (all Gate); absence is not a confirmed delisting. Official Gate adds MPLX_USDT; Bitget spot adds AXISUSDT and lacks RATKRUSDT; official Bitget futures lacks EPICUSDT. No ISIN change. Bitget473 trading-status changes and RBOTZUSDT description shortening to BOTZ ETF do not establish corporate action or ratio changes.

AXIS identity/chain and OKX wrapped XNFLX remain pending independent proof; xyz:TTWO now appears with a crypto classification and requires deployer-issued stock identity/unit evidence. io:MOONSHOT and Variational MDB remain pending from yesterday. No price/name-only merges. Official contract catalogs establish listing/category, not cross-venue economic equivalence or present tradability. Source empty base/quote evidence can inflate structural conflict flags; retained raw evidence distinguishes missing fields from actual contradictory identity.

Configured tradfi/veridex sample sources still point at api.example.com; real consumer feedback unavailable. Existing unresolved/ambiguous and chain/wrapped/collision leads remain. No deployment, restart or push performed. Local .automation-state and logs preserved and excluded from commit.

Successful post-write sync: 2026-10-21T01:05:23.477517778Z → 2026-10-21T01:05:32.689836936Z. Latest write: 2026-10-21T01:04:47.697771+00:00. Assets read back and retained in final-online-verification.json.
