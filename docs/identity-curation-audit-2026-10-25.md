# Market identity audit — 2026-10-25 (Beijing)

Covered **19,686 unique market keys /20 platforms**,19,618 evidence rows. Full current discovery plus registry override union,with official Extended404 perpetuals,all493 gTrade pairs,184 GRVT instruments,126 GMX pools,1,061 Gate futures and40 Bybit dated futures. All79 resolve batches succeeded(maximum250,two workers). Every key has a disposition,result and preserved raw source/chain evidence in local dispositions.jsonl. Initial statuses:18,496 resolved,1,067 unresolved,123 ambiguous. Structural success is not independent identity/comparability certification.

One added key:io:SKHX.295 absent keys,all Gate;many are returning/disappearing leveraged spot entries and no deletion or delisting was inferred from absence. Snapshot cached2026-10-25T01:02:12.586821753Z,no cache error. All14 main fetches succeeded with URL/time manifest. All official3,594 Bitget spot and857 USDT perpetual keys covered.3,019 official stock spot records have correct stock class. Official Gate/Bitget symbol inventory and identity fields unchanged;Bitget security drift empty. Bybit official916-instrument catalog and40 futures have no contract-type,expiry,settlement or identity-field drift. Prior411 rules and38 multi-collateral unknown-settlement guards passed readback.

## New critical finding: inactive markets labeled live and eligible

The collector discarded Hyperliquid isDelisted and marked every core/HIP-3 row live. New io:SKHX exposed the issue: official metadata isDelisted true,while discovery said live and resolve allowed a generic equity:ticker:skhx comparison. Entropy's official equity directory contains no SKHX issuer specification. Its name alone does not prove a Korean common share,ADR,leveraged ETF or another reference. It is now an unknown scoped inactive product with comparison prohibited;no global SKHX/SKHY reclassification or leverage/unit assumption.

Fetched official metadata for the core market and every HIP-3 deployment:11 catalog responses,zero errors. **206 native products carry isDelisted true**,including core tokens and inactive HIP-3/placeholder symbols.104 were comparison eligible and102 ambiguous. The flag may include disabled/not-yet-enabled placeholders;it does not prove prior trading activity,issuer identity or historical event date.

**206 exact online overrides** now prohibit comparison for these products while preserving historical lookup records,raw symbols and current multiplier/quote/settlement boundaries. Existing classes/underlying identifiers retained except unverified io:SKHX,which uses an unknown product identity. Scoped delisted-product IDs separate different deployments. No stock/crypto ticker-wide reclassification,price-ratio mapping or inferred reactivation. isDelisted controls operational eligibility,not underlying asset identity. No credentials saved.

Primary evidence: [Hyperliquid official info API](https://api.hyperliquid.xyz/info),POST perpDexs plus meta for core and each dex;all request bodies,response rows and timestamps retained in official-hyperliquid-all-metadata.json. [Entropy official equity directory](https://docs.entropy.io/asset-directory/equity-assets.md) also preserved. Official spot token metadata confirms collateral token0 is USDC for the new io market;other repaired markets preserve previously resolved quote/settlement rather than introduce new collateral assumptions.

## Systemic prevention

Collector now reads isDelisted for both core and HIP-3 catalogs,emits status delisted and preserves sourced instrumentMetadata.IsDelisted. No extra production network calls or polling cadence changes. Generator gives the explicit inactive flag priority over generic live/classification hints and emits a product-scoped prohibited comparison. An inactive entry remains queryable for history;it cannot be grouped with an active comparable market. Regression updates cash:WTI to remain separate from live OKX CL,while the remaining active alias and quote-boundary checks remain.

Inactive metadata is not allowed to promote a global asset/issuer alias. Generated schema bumped7→8;this includes yesterday's dated-future schema7 protection. Online remains9534aed/schema6,reviewed1942,so collector inventory may still show stale live labels until deployment. Online manual rules already block comparison. Deploy the local commits to prevent recurrence for newly inactive/new dated contracts.

If a product is re-enabled,it needs an official status/identity/unit review and an explicit override update before comparison can resume. This run did not silently restore eligibility. Existing inactive issuer/unit questions remain unresolved even after operational quarantine.

## Validation,performance and persistence

Go race suite,go vet,Linux amd64 build and full19,618-row replay across two successful local syncs pass. Regressions cover core/HIP-3 status serialization,stale generic live evidence in both orders,no global alias promotion,active market behavior and inactive aggregation boundaries. Independent generation without reviewed overrides verified **all206 official inactive products** produce prohibited comparison contracts.

Local baseline uses the original snapshot;post-change replay enriches that same row set with fresh official status metadata. Baseline CPU770ms/wall710ms,reader p50/p99 833/4,000ns;enriched replay CPU708ms/wall603ms,reader708/2,208ns. Heap187→124MB is a single-run GC-sensitive observation,not a production memory-saving claim. Generator asset rows3996→3910 reflect suppressing unsupported global aliases from inactive evidence. No production CPU/event-loop metrics available. No per-tick rebuild,queue,drop or frequency change;only operational comparison eligibility becomes conservative. Extra metadata objects are allocated at discovery cadence.

Immediate readback passed. After a successful sync started after the write,all**617 prior/new rules** matched expected identity,type,expiry,settlement,unit and comparison fields,with asset readbacks retained. No overwrite,repeated writes or sync pause.206 newly exact rules add to411 prior rules. Full before/after payloads,official evidence and verification retained.

## Remaining coverage limits

Configured tradfi/veridex sample endpoints still api.example.com,so live consumer feedback unavailable. Identity/unit questions remain for inactive io:SKHX,io:MOONSHOT,xyz:TTWO/MUU and other deployments;quarantine does not certify ordinary-share/ADR/leveraged reference identity. Variational CVNA/ZCSH/MDB,OKX X-prefixed spot,chain/wrapping/collision leads and retained r-prefixed crypto aliases remain pending. Existing411 rules stayed correct. No speculative merge,global cleanup,remote push,deployment or restart. Preexisting untracked .automation-state/logs preserved and excluded from commit.

Latest write 2026-10-25T01:07:45.865728+00:00;successful post-write sync 2026-10-25T01:09:07.525303268Z → 2026-10-25T01:09:09.313079488Z. Full verification:final-online-verification.json.
