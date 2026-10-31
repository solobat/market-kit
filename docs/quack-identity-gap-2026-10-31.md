# Q identity gap and discovery evidence retention — 2026-10-31

## Observed cause

Online registry schema 8 had a crypto/rwa_stock collision for Q. Binance QUSDT
had explicit COIN evidence, whereas Bybit QUSDT had no symbolType/class hint and
Variational Q had no native classification in the discovery export. Their exact
market overrides were unknown/ambiguous and shadowed the global crypto alias.
The opportunity engine correctly rejected them before history sampling. This is
not evidence that Q had previously been correctly classified and was then lost.

Official Variational metadata names ticker Q "Quack AI"; Bybit's exact linear
instrument names QUSDT "quack-ai", base Q, USDT quote/settlement. Bitget RQUSDT
is a separate reviewed equity product. Do not reclassify all markets named Q.

## Changes

Generated market overrides now retain explicit coin/RWA classification evidence,
separately from ticker fallback. At non-pruning discovery sync, historical ticker
collisions and explicit per-market evidence are considered before alias backfill.
An exact previously verified crypto market can retain its identity when a later
source omits the classification. Legacy/global/generic labels are not promoted
to verified evidence. New explicit contrary metadata, changed canonical product,
dated products, prohibited products and equal-strength source conflicts do not
inherit the old classification. Conflicted markets can no longer be made eligible
again by the generic alias backfill stage.

Evidence fields are additive; older registries without provenance remain
conservative. This cannot automatically certify an entirely new unverified
same-ticker asset. Such a market still requires native evidence or exact review.
Pruning intentionally rebuilds from current evidence rather than retaining history.

Two exact reviewed Quack AI rules are embedded for Variational Q perpetual and
Bybit QUSDT perpetual, with official source evidence recorded. They override weak
generated records at startup and after every sync, while preserving the equity
and unverified-venue boundaries.

## Validation

Full Go tests, vet, server build and race tests for curation/identity/server passed.
Regression tests cover persisted evidence across consecutive syncs, newly appearing
and historical collisions, legacy unverified labels, contrary stock evidence,
equal-strength conflicts, pruning, changed products and Q's reviewed-overlay path.

A 19,639-row current discovery replay against the same online registry changed
zero existing market identities and recorded classification provenance on 6,067
rules. One local baseline/new sync measured CPU 245/245 ms, wall 248/189 ms and
allocated 203/233 MiB; this is not production profiling or a CPU improvement claim.
All additional work runs at discovery cadence. Resolve/ticker hot paths, polling
cadence, buffering, message drops and market-data freshness are unchanged; sync
allocation cost increases. There is no JavaScript event-loop change.

Online exact Q rules were written at approximately 05:25:12 UTC. A successful
auto-sync started 05:25:13 UTC and finished 05:25:14 UTC; subsequent resolve readback
confirmed both crypto legs remain eligible and the stock keeps equity identity.
The opportunity engine caches identities for ten minutes; no historical Q samples
are fabricated and current opportunity qualification is still required.

Code changes are local and require deployment to activate the general mechanism.
The online data repair is already active. No service restart or remote deployment
was performed.

Sources:
- https://omni-client-api.prod.ap-northeast-1.variational.io/metadata/stats
- https://api.bybit.com/v5/market/instruments-info?category=linear&symbol=QUSDT
