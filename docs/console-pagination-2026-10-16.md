# Console server-side pagination

The console now reads `GET /api/v1/console` instead of downloading the full
runtime registry and discovery inventory on mount. Existing full-export APIs
remain available for integrations and audits.

## Request contract

- `view`: `registry`, `rules`, `groups`, `asset-detail`, `symbols`, `samples`, or `playground`.
- `page`: one-based, default 1; `limit`: default 50, range 1–100. Invalid values return 400.
- `q`: case-insensitive substring search, applied before pagination. Registry searches include canonical, aliases, unit aliases and class; exact canonical/alias matches come first.
- Registry: `class`; groups: `class`, `platform`, `type`, `status=review|ready`.
- Asset detail: `asset`, matched exactly against canonical. Contracts are paged as a unit, retaining both override and discovery provenance for the same venue/type/raw symbol.
- Symbols: comma-separated `platform`, `type`, `class`, `layer`. Optional `presence=true` and `presence_platform`, `presence_type`, `presence_class` compute the underlying intersection against the entire server inventory before paging.
- Explicit symbol download: `export=quotedCsv|csv|lines|json`, `field=rawSymbol|canonicalSymbol|baseAsset`. Returns a deduplicated, sorted attachment containing only that field, covering all matches regardless of page. The browser does not load full market objects to generate this download.

Response includes the current page, full matched `total`, global facets and
counts, and discovery snapshot timestamp/error. Group pages contain at most
three preview markets per group, with full `groupCounts`; the asset detail link
opens the paged market list. Samples and playground return only metadata;
playground and imported samples use the server resolver and bounded batch API.
`/api/discovery/sync?compact=1` returns a market-count summary instead of the
full discovery payload; the console then refreshes its active query.

## Performance and freshness

The server builds search text, stable ordering, symbol rows and facets once per
registry resolver / discovery-cache timestamp revision. It reuses discovery
groups built during sync when they match the current resolver. After a manual
classification or identity edit, groups are recomputed against the new registry
once, preventing stale classification. The console index has its own mutex;
index work never holds the realtime resolver reader lock. It is built lazily,
so the first browse after a revision pays the index-building cost. Existing
sync/ingest frequency and delivery semantics are unchanged; there is no ticker
buffer, message drop or additional source polling on each browser search.

The browser keeps only the active page. Search is debounced by 250 ms;
superseded requests are aborted and guarded by a revision check. Filter and
page-size changes reset to page 1. Selection/copy is explicitly per page;
whole-result downloads preserve bulk output. Initial metadata/source requests
run independently, and no full discovery fetch is started on mount.

Local replay of an 18,411-market inventory and 8k+ asset registry, using
uncompressed JSON sizes (not production network latency):

| Request | Response bytes | Warm handler wall time |
| --- | ---: | ---: |
| Old full runtime registry | 6,033,948 | 9.4 ms |
| Old full discovery | 8,551,352 | 16.1 ms |
| Registry page, 50 assets | 7,459 | 0.28 ms |
| Rules page, 50 rules | 17,940 | 0.17 ms |
| Groups page, 50 previews | 99,506 | 1.11 ms |
| OPENAI detail | 18,357 | 1.38 ms |
| Symbol intersection page | 19,246 | 2.04 ms |

The initial data response drops from approximately 14.6 MB to 7.5 KB; static JS
and CSS are additional. Full cold index construction measured 456 ms wall /
563 ms process CPU in separate samples. Reusing current discovery groups
measured 44 ms wall / 77 ms CPU. In a 20-load local process-CPU comparison,
legacy full registry + discovery serialization took 589 ms versus 5.3 ms for
warm registry pages. These are development-machine measurements, not an ECS
CPU or user-visible network latency guarantee. Low-frequency index rebuilds
still create a short CPU burst after data changes; browser render work is
bounded by the selected page size.

Validation includes `go test -race ./...`, `go vet ./...`, frontend tests/build,
and browser checks for page navigation, search resetting page 1, deep asset
links and Symbol filtering. Optional inventory replay:

```sh
MARKET_KIT_CONSOLE_INVENTORY=/path/to/discovery-current.json \
MARKET_KIT_CONSOLE_REGISTRY=/path/to/runtime-registry.json \
go test ./server -run '^TestConsoleFullSnapshotPayloadAndLatency$' -v
```

## Deployment

Deploy the Go backend first, then build/deploy the frontend (`npm --prefix
frontend run build`). A frontend deployed before the matching backend will show
a query error; it deliberately does not fall back to megabyte-scale downloads.
The existing `/api/:path*` proxy rewrite already covers `/api/v1/console`.
No database migration or environment-variable change is required. Keep the
normal automatic discovery sync enabled, or explicitly sync discovery after
startup; the console reads the server cache and shows when it is not ready.

At verification time on 2026-10-16, production `/api/v1/version` and remote
`main` still reported `5c0fa2f`; health reported required registry version 4.
The earlier local identity-curation commit `4365223` requires deployment as
well. The pagination commit is a descendant and includes that work. Deploy
from the new commit, verify `/api/v1/version`, then verify
`/api/v1/console?view=registry&limit=50` returns JSON. In browser network tools,
loading `/registry` should request console pagination and source metadata,
without requesting `/api/v1/registry` or `/api/discovery/current`.
