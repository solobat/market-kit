# Market identity audit — 2026-10-30 (Beijing)

Covered **19,782 keys /20 platforms**,22,081 evidence rows. Current discovery+registry union supplemented by official Extended416 perpetuals,all493 gTrade pairs,GRVT184,GMX126 pools,Gate1065 contracts and40 Bybit dated futures. All80 resolve batches succeeded(max250,at most2 concurrent). Every key has a disposition and preserved source/chain evidence. Initial states:{'resolved': 18571, 'unresolved': 1088, 'ambiguous': 123}. Structural resolution is not independent execution-comparability certification.

All14 main fetches succeeded;inventory cached2026-10-30T00:56:12.606750307Z,no cache error. Supplementary official Bitget/Bybit/Hyperliquid all11 catalogs and QFEX refdata saved with timestamps.12 added,426 absent union keys. Large absence set is returning/disappearing Gate leveraged-token coverage;no delisting or multiplier inferred,no deletion performed. Online remains4e897dd/schema8;prior a1bbe11/schema9 QFEX fix not deployed. Prior870 rules,including39 blank-settlement guards and233 QFEX fixes,passed initial readback.

## Five exact online rules

1. Bitget EWZSUSDT,FTNTUSDT,PFEUSDT: official support listing identifies stock perpetuals launched2026-10-29;exact API verifies isRwa YES,perpetual,USDT margin. Existing generic stock class retained,venue product/underlying/comparison IDs scoped and external comparison ambiguous. EWZS not aliased to EWZ. No ETF/issuer/unit equivalence inferred. Indexed official support page and article excerpt are class evidence;not claiming a full direct article retrieval.
2. QFEX BRENT-USD: new market had USD settlement despite official margin_asset USDC. Corrected USDC settlement,retained USD reference quote,scoped commodity:qfex:brent product and ambiguous comparison. No commodity future roll or barrel conversion inferred. This recurrence is expected until yesterday's schema9 collector/generator fix deploys;do not rewrite already-correct233 prior rules.
3. Hyperliquid xyz:ICE: official meta proves exact market exists but supplies no issuer/class identity. Online generic crypto:ticker:ice eligible join was unsupported. Exact market now unknown:hyperliquid:xyz:ice,unknown class,ambiguous comparison,USDC settlement. This is a conservative identity quarantine,not an assertion that ICE is a stock or a particular crypto. Official XYZ docs inventory fetch403;leave issuer/unit proof pending. No global ICE class edit or alias. Asset canonical endpoint may still show a global crypto ICE;exact scoped market resolve takes precedence for this product.

Writes in batches3+2,compact=1,strict prewrite equality,immediate readback;before/after saved in first-online-write.json and extra-online-write.json. Combined online-write.json is the expected-rule manifest at latest write time. All **875 audited rules** passed after a successful auto-sync started after latest write. Asset readbacks saved separately. No overwrite loops or sync pauses.

After repairs,235 QFEX current-reference settlement checks and745 Extended official-category checks passed. Former faults are preserved as initial flags,with final exact results in dispositions.

## Official lifecycle changes and remaining evidence

Hyperliquid mkts:SMALL2000,US500,USBOND,USTECH gain isDelisted true;deployed schema8 already generates scoped inactive IDs and prohibited comparison for all4. No manual rewrite required. Official inactive count203→207. No reactivation changes;four prior active-flag-removal leads remain under manual prohibition. New xyz:ICE handled above.

Bitget RCONL fullName shortened;ISIN unchanged. No split,leverage reset,conversion or underlying COIN merge inferred;leveraged product boundary retained. Bybit40 dated futures and four previously scheduled-delivery perpetual metadata unchanged. Actual expiry/past-delivery operations are not assumed from symbol names alone. New Binance6 stock perpetuals carry existing stock mapping;structural check does not certify cross-venue unit equivalence. USDC-AED retains native pair. Placeholder consumer-feedback endpoints,registry-only Variational AMC/BAC identity proof,historical chain/wrapper collisions and retained r-prefixed alias cleanup remain open.

## Validation and publication

identity/curation/server regression suites passed,server build passed,diff check passed. Only reviewed data/evidence and this report changed today;no source-code/hot-path change or cadence/drop/freshness change. No production CPU improvement claimed. Prior schema9 code remains pending deployment. This audit did not push,deploy,restart or alter sync settings. Local logs/state remain untracked;preexisting logs preserved.

Sources: [Bitget official listing](https://www.bitget.com/support/articles/12560603899143),[Bitget official futures index](https://www.bitget.com/support/sections/12508313405000),[QFEX official refdata](https://api.qfex.com/refdata),[Hyperliquid official metadata](https://api.hyperliquid.xyz/info),[Extended official catalog](https://api.starknet.extended.exchange/api/v1/info/markets). Fetch URL/time manifests and per-key dispositions retained locally,without credentials.
