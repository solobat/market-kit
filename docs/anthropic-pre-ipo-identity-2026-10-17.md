# Anthropic pre-IPO identity correction — 2026-10-17

The reported Binance `ANTHROPIC` / Hyperliquid `ANTH` positions refer to the same issuer if the Hyperliquid market is Entropy `io:ANTH`. The screenshot omits the deployer; confirmation was requested. Paragon `para:ANTH` is not included without independent current unit/issuer evidence.

## Evidence

- Binance official launch notice: https://www.binance.com/en-AU/support/announcement/detail/2b9ff1e1acf8433793bc40c59a0cb852 — Anthropic PBC, estimated 1 billion shares, USDT settlement; share count and contract size may change on IPO.
- Entropy issuer directory: https://docs.entropy.io/asset-directory/pre-ipo-assets — ANTH references Anthropic PBC, USDC quote, no-IPO resolution August 18, 2028.
- Entropy denomination/lifecycle: https://docs.entropy.io/market-types/pre-ipo-perpetuals — each $1 price means $1B issuer valuation; continues as market-cap contract after IPO.
- Read official HTML main content on 2026-10-17; Markdown fetches were blocked. Do not substitute third-party summaries for these documents.

## Changes

Four exact reviewed rules: Binance native `ANTHROPICUSDT` and display `ANTHROPIC/USDT`; Hyperliquid native `io:ANTH` and Portfolio-Lens display `io:ANTH/USD`. All use canonical base `ANTHROPIC`, class `rwa_stock`, underlying `equity:private:anthropic`, unit 1. Product IDs and comparison keys remain distinct; comparison status is ambiguous because lifecycle/oracle/conversion rules differ. No global ANTH alias or leveraged-product merge.

Portfolio-Lens position pairing now recognizes Entropy using its original `io:ANTH` symbol, even when the existing market-kit cache returns `ANTH`. Unknown deployers, Paragon, bare ANTH, and other venues remain independent. Existing quantity matching remains in use because both current pre-IPO units correspond to $1B valuation. Revisit after any Binance share-count/contract-size change.

The online write added both Binance symbol spellings before the new regression test completed. This exposed an existing resolver defect: equivalent separator spellings are counted as conflicting candidates. Old online resolve therefore currently reports ambiguous for Binance; Entropy resolves correctly. Online has no override-delete API. The code correction deduplicates complete equal identities (excluding raw spelling only); any differences in units, IDs, comparison status, settlement, expiry, or normalized venue symbol remain ambiguous. Deployment is required to restore Binance resolved status. Do not repeatedly rewrite these entries.

## Verification and deployment

- Go race suite and vet passed; Linux amd64 server build passed.
- Full 19,505-row replay across two successful syncs passed with 1,702 reviewed rules. Sample sync wall 0.742s / CPU 0.877s; reader p99 2.46µs. No ingest cadence, freshness or drop-policy changes.
- Portfolio-Lens TypeScript check and position pairing/exposure/hedge tests passed, including equal-quantity $10.3K long vs $10.2K short (no naked exposure alerts), real residual quantity, and excluded products/deployers.
- Exact online write at 2026-10-17T01:55:30.397837Z. Successful sync started 01:56:35.737624632Z and finished 01:56:37.9619025Z; Entropy corrections persisted. Binance still ambiguous on old code as described above.
- Online evidence: `.automation-state/market-kit-audit/anthropic-2026-10-17/` (before/write/readback and primary-document extracts). No credentials saved.
- Deploy the new market-kit commit plus previous unpushed a92453c; reload/rebuild Portfolio-Lens. After deployment, resolve all four exact rules and verify shared underlying, distinct product IDs, and successful sync persistence. Previous XYZ100 post-deploy correction also remains pending; see the daily audit.
- Portfolio-Lens had extensive pre-existing user changes; only a small additive pairing fix, one call-site argument, and a dedicated regression test/import were changed for this report. No unrelated work was staged.
