# Market identity audit — 2026-10-26 (Beijing)

Covered **20,122 unique market keys /20 platforms**,21,792 evidence rows. Full online discovery and registry override union,plus official Extended404 perpetuals,all493 gTrade pairs,184 GRVT instruments,126 GMX pools,1,061 Gate futures and40 Bybit dated futures. All81 resolve batches succeeded(maximum250,two workers). Every market has a disposition,result and retained raw source/chain evidence in local dispositions.jsonl. Initial states:18,932 resolved,1,067 unresolved,123 ambiguous. Resolved/unflagged is not independent product comparability certification.

Fresh discovery cached2026-10-26T01:04:11.083221543Z,no cache error. All14 main fetches succeeded with URL/time manifests.436 added keys,no absent keys:426 Gate,9 Hyperliquid spot identifiers,one Binance PREDICT perpetual. Many Gate entries are native leveraged tokens and remain separate unresolved/ambiguous products;no underlying merge or ratio inference. Official Bitget spot/futures and Gate futures inventory/identity fields unchanged. Bybit contract type,expiry,settlement and symbol inventory unchanged. Hyperliquid core plus all HIP-3 metadata fetched successfully;all206 inactive flags unchanged,no reactivation. Bitget security drift only RBUUUUSDT status online→limit_open,not identity or corporate-action evidence.

Prior617 exact rules and38 unknown-settlement multi-collateral guards passed initial readback. No previously repaired rule was overwritten. Online still9534aed/schema6;local schema8 collector/dated/inactive guards remain pending deployment. Known placeholders and inactive entries are not silently re-enabled.

## New conflict: crypto DAR CFD mistaken for equity DAR

Official gTrade pair250 is DAR/USD,groupIndex10,whose group name is **altcoins**. Both fresh official catalog and bootstrap underlyingCategory coin identify the crypto class. Initial online result was rwa_stock,equity:ticker:dar,eligible,equity_perpetual,with settlement USD. This is a market-specific crypto/equity collision,not permission to reclassify every DAR ticker.

Desired reviewed rule: crypto:gtrade:dar product and underlying,ambiguous comparison,crypto_cfd,contract multi_collateral_cfd,reference DAR/USD,unknown settlement. No token address or issuer-rename equivalence established;no global DAR→D alias or matching to the stock. Bitget RDARUSDT official metadata identifies Darling Ingredients Inc.,ISIN US2372661015. Its stock classification remains unchanged in independent online readback.

Sources: [gTrade official trading variables](https://backend-arbitrum.gains.trade/trading-variables),[Bitget official stock/spot catalog](https://api.bitget.com/api/v3/market/instruments?category=SPOT). Exact official pair/group and reviewed intent retained in repair-plan.json and reviewed_registry.evidence.json.

## Deployed write-API blocker and safe online result

A strict pre-write check initially found a changed response and prevented that write;recheck showed the same stock misclassification. Typed crypto repair was then written once,with immediate readback. The deployed API silently filled a missing settlementAsset from quote USD for known asset classes. Resolver's existing multi-collateral blank-settlement guard cannot undo a nonempty incorrect value already inserted by the request normalizer. This was an API normalization error,not a sync overwrite.

A single corrective write safely quarantined the exact pair: **unknown class,unknown:gtrade:dar product,comparison prohibited,multi_collateral_cfd,settlement absent**. It no longer appears as the stock and cannot participate in automated comparison. Original symbol/reference quote and unit1 retained. This temporary unknown label is a technical quarantine,not uncertainty that the official category is crypto. Same-name stock still resolves rwa_stock/eligible independently. No global class edit.

The desired local reviewed crypto identity intentionally differs from the temporary online quarantine. Deploy the API fix,then reapply the desired DAR rule to replace the higher-priority manual quarantine. Do not repeatedly attempt the typed empty-settlement write against9534aed;it cannot represent this combination correctly. Both the initial typed write/mismatch and corrective before/after payloads are preserved in initial-typed-online-write.json and online-write.json. The failed pre-write response was not captured on the initial attempt,so its changed fields are unknown;the guard made no mutation at that point.

## Code fix and validation

Known-class override normalization and resolver now share DefaultSettlementAsset. Multi-collateral CFDs and supported dated-future contract types preserve missing settlement;explicit collateral remains authoritative. Ordinary supported perpetual default behavior remains. Local generation already has an explicit-coin-before-ticker fallback and collision regression including DAR;this code is not yet deployed online. No additional speculative classification rule was needed.

Added server regression covers crypto requests for multi-collateral/dated contracts,case/whitespace normalization,explicit USDC and ordinary perpetual compatibility. Added identity regression uses an independent homonymous stock fixture and checks scoped crypto CFD identity,ambiguous comparison and unknown settlement. Full Go race suite,go vet,Linux amd64 build and two-sync full21,792-row replay pass. Local replay CPU969ms/wall1.249s,reader p50/p99 1,000/9,833ns,heap156MB. These are single-run local runtime observations,not production CPU/event-loop claims. No polling,queue,drop or freshness cadence changes;normalization remains constant-size field processing.

After a successful automatic sync started after the corrective write,**all618 prior/new online rules** matched expected fields,including the deliberate quarantine,and asset readbacks were retained. No sync pause or repeated rewrite loop. One new exact market rule was changed,with one corrective write to remove the API-inserted false settlement. Runtime live protection is complete as quarantine;restoration of full crypto classification is blocked pending deployment and reviewed-rule reapply.

## Remaining coverage gaps and delivery

Live tradfi/veridex feedback still unavailable because configured sample URLs use api.example.com. Inactive product issuer/unit questions,retained r-prefixed alias source review,Variational CVNA/ZCSH/MDB,OKX X-prefixed spot and chain/wrapping/collision leads remain. Structural coverage and a resolved status do not prove eligibility. Prior206 inactive products and40 dated contracts remain protected. No deployment,restart or push. Local code/reviewed rule,evidence and tests committed;preexisting .automation-state/logs preserved and excluded. No credentials saved.

Corrective write 2026-10-26T01:14:17.854144+00:00;successful post-write sync 2026-10-26T01:15:07.524449124Z → 2026-10-26T01:15:09.641969051Z. Complete checks:final-online-verification.json.
