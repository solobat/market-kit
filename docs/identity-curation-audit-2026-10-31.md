# Market identity audit — 2026-10-31 (Beijing)

Covered **20,225 snapshot keys /20 platforms**, 21,782 evidence rows,plus8 wrong-contract-type queries added during verification. Current discovery+registry union supplemented by official Extended416 perpetuals,all493 gTrade pairs,GRVT184,GMX126 pools,Gate1070 contracts and40 Bybit dated futures. All81 resolve batches succeeded(max250,at most2 concurrent). Every snapshot key and additional repaired query has a disposition with raw source/chain evidence. Initial states:{'resolved': 19029, 'unresolved': 1073, 'ambiguous': 123}. Structural resolution is not independent execution-comparability certification.

All14 main fetches succeeded;inventory cached2026-10-31T00:56:10.987467081Z,no cache error. Official supplementary Bybit/Bitget/Hyperliquid all11 catalogs and QFEX refdata saved with timestamps.459 added,16 absent snapshot keys;added includes436 Gate entries(many returning leveraged tokens),14 Bybit,6 OKX,2 Hyperliquid,1 QFEX. Absence12 Binance,3 Bybit,1 OKX is coverage delta,not automatic delisting proof. No unit inference or underlying merge for returning leveraged products.

Online remains4e897dd/schema8/2254 reviewed rules;prior875 exact rules and39 absent-settlement guards passed. Pending QFEX schema9 code is included in today's schema10 deployment,not yet active online.

## Twenty exact online repairs

- Gate BIDU,EWZS,FTNT,ITUB,SMR USDT perpetuals: official contract_type stocks. Bybit BKNG,EWZS,OXY,SMR,XLI,XP USDT perpetuals: official symbolType stock/ETF,contractType LinearPerpetual,settleCoin USDT.11 rules now retain stock class but use venue-specific product/underlying/comparison IDs and ambiguous comparison until issuer/units verified. EWZS not aliased to EWZ;XP not globally reclassified;lot size not a price multiplier.
- QFEX ICE-USD: official EQUITY,USD reference,USDC margin. Corrected settlement and scoped ambiguous product. QFEX's stock class does not identify Hyperliquid xyz:ICE;its unknown/ambiguous quarantine remains verified.
-8 new Bybit20NOV26 dated products had correct actual future identities(expiry1795161600000,USDT settlement,ambiguous). Negative tests requesting the same symbols as perpetuals exposed a fallback bug:date20NOV26 became quote and settlement,unknown/ambiguous. No eligible join was claimed,but the currency fields were wrong. Exact perpetual-request overrides now retain official future identity/expiry/settlement and prohibit comparison. Actual future keys remain unchanged. This is type-mismatch quarantine,not eight newly listed perpetuals.

Two batches12+8,compact=1,strict prewrite equality,immediate readback. first-online-write.json and dated-online-write.json retain before/after;combined online-write.json is expected-rule manifest at latest write time.875 prior+20 new=**895 audited rules** passed after a new successful sync started later than latest write2026-10-31T01:00:44.994459Z.16 dated positive/negative queries passed separate final readback;initial failed probes retained transparently. Asset endpoints read separately. No sync overwrite or repeated rewrite loop.

## Systemic fixes /schema10

Generated native Gate stock and Bybit stock/ETF LinearPerpetual products now default to scoped venue identities and ambiguous comparison. Generic downstream stock hints do not authorize this guard;Gate native stocks category or Bybit source/type metadata required. Actual dated futures excluded. Reviewed exact identities continue to take priority. This prevents future native stock listings from silently joining by generic ticker while awaiting product-unit review.

Resolver now refuses heuristic parsing of a Bybit dated-suffix symbol after no explicit rule matches requested market type. Registered actual futures and reviewed wrong-type prohibitions retain priority. Unknown dated requests return unresolved with no fabricated quote/settlement. A fixed8-character ASCII suffix check runs only for Bybit fallback;no regex allocation,expiry inference,global scan,serialization or deep copy. Regression tests cover registered future,wrong perpetual,unknown lower-case dated suffix,normal perpetual,untrusted stock metadata and future-vs-perpetual boundaries. README documents behavior.

## Official lifecycle and remaining uncertainty

Bybit eight30OCT26 contracts absent and eight20NOV26 contracts added;40 active dated futures retained. EPIC/MITO/PTB/TMX now absent after previously scheduled lifecycle timestamp;no invented conversion. Hyperliquid new io:AAPL official isDelisted true:deployed schema8 correctly prohibits comparison,inactive count207→208. No manual rewrite needed. Prior four reactivation leads remain prohibited pending units/issuer review. Bitget HEI/MOODENG/ZENT spot and10000NEX perpetual absent from official catalogs;no historical deletion solely on absence. No Bitget security-field drift.

After repairs,236 QFEX current settlement checks and745 Extended official-category checks passed. New OKX WUXIBIO stays unknown/ambiguous pending product proof. Historical chain wrappers,retained r-prefixed aliases,Variational registry-only AMC/BAC proof,XYZ ICE issuer proof and placeholder consumer-feedback endpoints remain open. No live feedback or independent all-market execution certification claimed.

## Validation /runtime cost

Full Go suite,go vet,server build,diff check passed. Full21782-row local service replay passed:sync wall808.95ms,CPU879.05ms,reader p50/p99709/3291ns,heap185.2MB. Single local sample,not production profiling or evidence of improved CPU;input/heap differs from previous runs. Go service has no JS event-loop metric. Generation guard runs at sync cadence;fallback guard constant-size and only after failed exact lookup. No polling frequency,queue,coalescing,drop or freshness policy change,therefore no intentional new staleness/drop/burst tradeoff. Correctness change intentionally rejects unregistered dated queries. No push/deploy/restart performed by audit;schema10 remains pending manual deployment.

Sources: [Gate official contracts](https://api.gateio.ws/api/v4/futures/usdt/contracts),[Bybit official linear catalog](https://api.bybit.com/v5/market/instruments-info?category=linear&limit=1000),[QFEX official refdata](https://api.qfex.com/refdata),[Hyperliquid official metadata](https://api.hyperliquid.xyz/info). Fetch source/time manifests and per-key dispositions retained locally without credentials.
