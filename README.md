# market-kit

`market-kit` is a small shared Go module for market identity normalization across independent repos.

It is designed for systems like:

- discovery systems such as `slipstream`
- signal producers such as `tradfi-monitor`
- verification services such as `veridex`
- downstream derived-data projects

The goal is to centralize:

- exchange alias normalization
- market type inference
- venue symbol normalization
- canonical symbol / asset identity mapping
- explicit `resolved / ambiguous / unresolved` outcomes

## Recommended integration boundary

`market-kit` should primarily integrate with `slipstream`, not directly with every downstream business project.

Recommended roles:

- `slipstream`
  - discovery layer
  - continuously finds markets across platforms
  - exports raw market inventory such as `platform / venueType / marketType / symbol / base / quote`
- `market-kit`
  - aggregation + curation layer
  - groups raw discovered markets into candidate identity families
  - lets operators review ambiguous cases
  - publishes audited registry rules
- `veridex` / `tradfi-monitor`
  - consumers of published identity rules
  - may emit unresolved / ambiguous feedback, but should not each become their own market discovery source of truth

This keeps responsibilities clean:

- `slipstream` discovers
- `market-kit` aggregates and curates
- other projects consume

## Aggregation model

`market-kit` should aggregate in two stages after importing data from `slipstream`.

1. market grouping
   - normalize exchange ids
   - normalize market type
   - cluster raw venue symbols, base/quote, and exchange metadata into candidate market identities

2. asset grouping
   - merge candidate markets into canonical asset families
   - produce audited aliases and explicit market overrides

Price data may be used as a fuzzy supporting signal, but not as the primary identity key.

Preferred order of evidence:

- explicit override rules
- exchange metadata
- symbol structure and base/quote extraction
- known asset aliases
- optional price similarity as supporting evidence
- human review for final confirmation

## Slipstream import contract

The preferred upstream discovery source for `market-kit` is `slipstream`.

`slipstream` does not need to publish final canonical identities. It only needs to export normalized market inventory rows.

Recommended payload shape:

```json
{
  "source": "slipstream",
  "generatedAt": "2026-05-15T00:00:00Z",
  "items": [
    {
      "sourceId": "slipstream",
      "platformId": "okx",
      "platform": "OKX",
      "venueType": "cex",
      "marketType": "perp",
      "symbol": "DRAM-USDT-SWAP",
      "baseAsset": "DRAM",
      "quoteAsset": "USDT",
      "chain": "",
      "status": "live",
      "externalUrl": "https://www.okx.com/..."
    }
  ]
}
```

These rows map directly to the `discovery.ImportEnvelope` and `discovery.ImportedMarket` types.

## Discovery package

This repo now includes a first-pass discovery aggregation package:

```text
discovery/
  types.go
  aggregator.go
```

The discovery layer is intentionally separate from the runtime resolver.

It handles:

- importing raw market inventory from `slipstream`
- normalizing discovery rows into candidate markets
- grouping cross-venue markets into asset families such as `DRAM/USDT`

It does not:

- publish final rules automatically
- mutate the registry by itself
- treat fuzzy price similarity as a primary identity key

## Candidate groups

The first version groups imported markets by canonical asset family:

- canonical base asset
- quote asset

This is enough to produce review-ready buckets such as:

- `DRAM/USDT`
- `AAPL/USDT`
- `BTC/USDT`

Each group keeps:

- all imported venue markets
- exchanges seen
- market types seen
- chains seen
- evidence notes
- a `needsReview` flag

That means the review workflow becomes:

1. `slipstream` exports markets
2. `market-kit` imports them into candidate groups
3. operator reviews ambiguous families
4. registry rules are updated intentionally
5. a new version is tagged and consumed downstream

## Frontend console

This repo also includes a static Svelte console in `frontend/` for:

- browsing asset aliases
- inspecting market override rules
- testing resolver inputs visually
- syncing unresolved / ambiguous samples exported from remote services such as `veridex`
- reviewing candidate asset groups imported from `slipstream` discovery markets

Run locally:

```bash
cd frontend
pnpm install
pnpm dev
```

For more automated remote sample review, copy:

```bash
cp frontend/sync-sources.example.json frontend/sync-sources.local.json
```

Then edit `frontend/sync-sources.local.json` with your real ECS exporter URLs and headers.

Each source can optionally declare a `kind`:

- `discovery`
  - market inventory feeds such as `slipstream`
- `sample`
  - unresolved / ambiguous identity cases from downstream services

If omitted, `market-kit` infers `slipstream`-like projects as discovery sources and everything else as sample sources.

For `slipstream`, the recommended source is:

```json
{
  "id": "slipstream-prod",
  "label": "Slipstream 市场发现",
  "project": "slipstream",
  "kind": "discovery",
  "url": "https://api.example.com/slipstream/api/discovery/markets?limit=5000",
  "headers": {
    "X-Slipstream-Admin-Code": "replace-me"
  }
}
```

In production, the server can also be configured directly through environment variables:

```bash
MARKET_KIT_SLIPSTREAM_DISCOVERY_URL=https://api.example.com/slipstream/api/discovery/markets?limit=5000
MARKET_KIT_SLIPSTREAM_ADMIN_CODE=replace-me
MARKET_KIT_AUTOSYNC_ENABLED=true
MARKET_KIT_AUTOSYNC_INTERVAL=1m
```

With auto-sync enabled, `market-kit` periodically pulls the configured slipstream discovery export, generates a dynamic registry layer, hot-swaps the runtime resolver, and writes that dynamic layer to `MARKET_KIT_RUNTIME_REGISTRY_PATH`.

Even without any remote config, the server now exposes a built-in discovery source:

- id: `market-kit-bootstrap`
- kind: `discovery`
- behavior: fetches bootstrap market inventory directly from exchange public REST endpoints such as Binance, Bybit, OKX, Bitget, Gate, and Hyperliquid

When the console is running with `pnpm dev`, it exposes a local sync proxy so you can pull remote unresolved / ambiguous samples with one click, without retyping URLs every time and without depending on browser CORS against the remote exporter.

## Deployable server

This repo can also run as a directly deployable web application.

The Go server in:

```text
cmd/market-kit-server/
server/
```

provides:

- `GET /api/healthz`
- `GET /api/v1/version`
- `GET /api/v1/registry`
- `GET /api/v1/assets/{asset}`
- `GET /api/v1/resolve?exchange=<exchange>&symbol=<symbol>[&marketType=<type>]`
- `POST /api/v1/resolve`
- `POST /api/v1/resolve/batch`
- `GET /api/registry`
- `GET /api/discovery/sources`
- `GET /api/discovery/sync?source=<id|all>`
- `GET /api/discovery/lookup?symbol=<symbol>[&source=<id|all>]`
- static hosting for the built `frontend/dist` app

That means online deployment no longer depends on the Vite dev proxy.

### Production identity API

Downstream projects should use the `/api/v1/*` endpoints as the stable service boundary. These endpoints use the runtime merged registry already embedded in the server binary and do not fetch remote discovery sources during request handling.

Single resolve:

```bash
curl 'http://127.0.0.1:18120/api/v1/resolve?exchange=gate&symbol=SPCX_USDT&marketType=spot'
```

Response shape:

```json
{
  "status": "resolved",
  "confidence": 1,
  "reason": "matched explicit market override",
  "market": {
    "exchange": "gate",
    "marketType": "spot",
    "rawSymbol": "SPCX_USDT",
    "venueSymbol": "SPCX_USDT",
    "canonicalSymbol": "SPCX/USDT",
    "baseAsset": "SPCX",
    "quoteAsset": "USDT",
    "assetClass": "rwa_stock"
  }
}
```

Batch resolve:

```bash
curl -X POST 'http://127.0.0.1:18120/api/v1/resolve/batch' \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"exchange":"gate","symbol":"SPCX_USDT","marketType":"spot"}]}'
```

Registry and metadata:

```bash
curl 'http://127.0.0.1:18120/api/v1/version'
curl 'http://127.0.0.1:18120/api/v1/registry'
curl 'http://127.0.0.1:18120/api/v1/assets/SPCX'
```

The older `/api/discovery/*` endpoints are operational review tools. They are useful for discovery sync, candidate grouping, and audit workflows, but production consumers should not depend on them for hot-path identity resolution.

`GET /api/discovery/sync?source=market-kit-bootstrap` will trigger a fresh bootstrap pull from the built-in exchange REST collectors. Use `source=all` to merge every configured discovery source, including `market-kit-bootstrap` and `slipstream-prod`.

`GET /api/discovery/lookup` adds a lightweight presence query layer on top of discovery imports, so you can ask questions such as:

```bash
curl 'http://127.0.0.1:18120/api/discovery/lookup?symbol=TSM'
curl 'http://127.0.0.1:18120/api/discovery/lookup?symbol=DRAM&source=slipstream-prod'
curl 'http://127.0.0.1:18120/api/discovery/lookup?symbol=MRVL&source=all'
```

The response includes:

- matched canonical asset groups
- exchanges where the asset appears
- market types seen on those exchanges
- raw venue symbols for each matched market

### Production build flow

1. build the frontend
2. build the Go server
3. deploy both together from the same repo checkout

Example:

```bash
cd frontend
pnpm install
pnpm build

cd ..
go build ./cmd/market-kit-server
./market-kit-server
```

### PM2 deployment

For a simple ECS host deployment, copy `.env.production.example` to `.env`, adjust the
port or source config path if needed, then start the service with PM2:

```bash
cp .env.production.example .env
chmod +x ./pm2-*.sh ./deploy/deploy_ubuntu_pm2.sh
./pm2-start.sh
./pm2-status.sh
./pm2-logs.sh lines 100
```

On Ubuntu ECS hosts, the helper script installs the lightweight runtime dependencies,
builds the Go API server, starts or restarts PM2, and checks `/api/healthz`. It does
not build the Svelte frontend; deploy the frontend separately, for example on Vercel:

```bash
./deploy/deploy_ubuntu_pm2.sh --backend-port 18120
```

The PM2 process is named `market-kit` and reads `ecosystem.config.cjs`. Run
`./pm2-save.sh` after the service is healthy if you want PM2 to persist the process list
for reboot recovery.

### Nginx API proxy

If the frontend is deployed separately on Vercel, expose the ECS backend through Nginx
instead of opening the Go server directly. The included snippet follows the same shared
API-domain pattern as `tradfi-monitor` and proxies `/market-kit-api/*` to the local
backend on `127.0.0.1:18120`:

```bash
bash scripts/setup-nginx-market-kit-api.sh
```

Then include the installed snippet inside the target HTTPS `server { ... }` block:

```nginx
include /etc/nginx/snippets/market-kit-api.conf;
```

Verify locally and through Nginx:

```bash
curl http://127.0.0.1:18120/api/healthz
curl -i https://api.immortal.app/market-kit-api/api/healthz
curl -i https://api.immortal.app/market-kit-api/api/v1/version
```

Local Vite dev and Vercel deployments are configured to keep using same-origin `/api/*`
requests and automatically forward them to `https://api.immortal.app/market-kit-api`.
Override the local dev target with `VITE_MARKET_KIT_API_BASE` if needed.

### Server environment variables

- `MARKET_KIT_HTTP_ADDR`
  - default: `:18120`
- `MARKET_KIT_SYNC_SOURCES_PATH`
  - optional path to your real sync source config file
- `MARKET_KIT_SLIPSTREAM_DISCOVERY_URL`
  - preferred production shortcut for the slipstream discovery export
- `MARKET_KIT_SLIPSTREAM_ADMIN_CODE`
  - optional admin code sent as `X-Slipstream-Admin-Code`
- `MARKET_KIT_AUTOSYNC_ENABLED`
  - default: `true`
- `MARKET_KIT_AUTOSYNC_INTERVAL`
  - default: `1m`
- `MARKET_KIT_AUTOSYNC_SOURCE`
  - optional explicit discovery source id; by default the first non-bootstrap discovery source is used
- `MARKET_KIT_RUNTIME_REGISTRY_PATH`
  - default: `data/runtime_generated_registry.json`
- `MARKET_KIT_REQUEST_TIMEOUT`
  - default: `12s`
- `MARKET_KIT_FRONTEND_DIST`
  - default: `frontend/dist`

If `MARKET_KIT_SYNC_SOURCES_PATH` is not set, the server falls back to:

1. `frontend/sync-sources.local.json`
2. `frontend/sync-sources.example.json`

Auto-sync intentionally ignores the built-in `market-kit-bootstrap` source unless you explicitly set `MARKET_KIT_AUTOSYNC_SOURCE=market-kit-bootstrap`. This keeps slipstream as the single recurring market collector in production.

## Repo boundaries

This repository intentionally keeps both the shared backend module and the local review console in one place.

Treat them differently:

- release-critical:
  - `identity/`
  - `registry/`
  - `go.mod`
- local review tooling:
  - `frontend/`

The `frontend/` app exists to review registry rules locally before release. It does not need to be deployed publicly.

When using the remote sample sync panel, the browser reads JSON directly from your remote exporter URL. In practice that means the remote endpoint should allow cross-origin access from your local browser session.

When running in local dev mode with `sync-sources.local.json` configured, the console can instead fetch through the local Vite proxy endpoints:

- `GET /__market-kit/sources`
- `GET /__market-kit/sync?source=<id>`

That path is more automated and avoids most manual CORS friction during review.

## Recommended workflow

1. edit registry or resolver logic
2. review the result locally in `frontend/`
3. run verification
4. commit
5. tag
6. push
7. downstream upgrade PRs are opened automatically for configured Go consumers

Verification:

```bash
go test ./...
cd frontend && pnpm build
```

## Commit discipline

To keep history readable, use narrow commits:

- `registry: ...`
- `identity: ...`
- `frontend: ...`

Examples:

- `registry: add dram okx spot and perp overrides`
- `identity: improve okx market type inference`
- `frontend: refine resolver playground layout`

## Tag discipline

Before creating a tag, confirm whether the release actually changes shared runtime behavior.

Usually tag when one of these changed:

- registry rules in `identity/default_registry.json`
- resolver behavior in `identity/`
- public Go module surface

Usually do not create a new release tag for frontend-only polishing unless you explicitly want to version the local review console changes too.

## Automatic downstream bumps

This repo can automatically open downstream upgrade PRs whenever a new `v*` tag is pushed.

Current configured downstream Go consumers live in:

```text
.github/downstream-go-consumers.json
```

Right now that automation targets:

- `solobat/tradfi-monito`
- `solobat/veridex` in `backend/`

The workflow lives in:

```text
.github/workflows/bump-downstreams-on-tag.yml
```

It uses:

```text
scripts/bump-downstream-go-module.sh
```

Required GitHub secret:

- `DOWNSTREAM_REPO_TOKEN`
  - should have enough access to clone the downstream repos, push branches, and open pull requests
  - for a fine-grained token, grant `contents: read/write` and `pull requests: read/write` on each downstream repo

Behavior on each new tag:

1. clone each configured downstream repo
2. run `go get github.com/solobat/market-kit@<tag>`
3. run `go mod tidy`
4. push branch `codex/market-kit-<tag>`
5. open or update a PR in the downstream repo

If the automation is added after a tag is already published, you can also run the workflow manually with a tag input such as `v0.2.3`.

This only handles downstream repos that consume `market-kit` as a Go module. HTTP consumers such as local tools or browser extensions should continue to update through their own deployment flow instead.

## Principles

`market-kit` should be the shared identity layer, not a business-logic layer.

## Generated registry layer

`market-kit` now supports two registry layers:

- `identity/default_registry.json`
  - hand-curated seed rules
  - explicit aliases for sensitive assets such as RWA / stock-like / commodity-like instruments
- `identity/generated_registry.json`
  - auto-curated high-confidence rules derived from `slipstream` market discovery export
  - intended to collapse the long tail of venue-specific stable-quote CEX symbols into explicit `market_overrides`

At runtime the Go resolver and local frontend audit console merge these two layers, with the hand-curated default registry taking precedence when keys overlap.

## Bootstrap discovery export

If you want `market-kit` to generate its own initial discovery payload without waiting for `slipstream`, you can pull directly from the built-in exchange REST collectors:

```bash
cd /Users/tomasyang/github/market-kit
go run ./cmd/market-kit-bootstrap-discovery \
  --output /tmp/market-kit-bootstrap-discovery.json
```

Optional flags:

- `--sources binance,binance-web3,bybit,okx`
  - fetch only selected exchanges
- `--timeout 30s`
  - override upstream request timeout

The output shape matches `discovery.ImportEnvelope`, so it can be fed back into the frontend review flow or any downstream importer that already accepts `slipstream`-style discovery exports.

## Curating from slipstream

After pulling the latest `slipstream` discovery export locally, regenerate the auto-curated layer with:

```bash
cd /Users/tomasyang/github/market-kit
go run ./cmd/market-kit-curate-slipstream \
  --input /path/to/slipstream-discovery.json \
  --output identity/generated_registry.json
```

For the production automation path, let `market-kit` fetch platform market inventories directly and write a compact review report:

```bash
go run ./cmd/market-kit-curate-slipstream \
  --bootstrap \
  --source-name market-kit-bootstrap \
  --output identity/generated_registry.json \
  --review-output identity/generated_registry.review.md
```

You can limit direct collectors while testing:

```bash
go run ./cmd/market-kit-curate-slipstream \
  --bootstrap \
  --sources binance,binance-web3,bybit,okx,bitget,gate,hyperliquid \
  --output identity/generated_registry.json
```

If you still want to curate from an external discovery export such as `slipstream`, the remote URL mode remains available:

```bash
go run ./cmd/market-kit-curate-slipstream \
  --url "https://api.example.com/slipstream/api/discovery/markets?limit=5000" \
  --header "X-Slipstream-Admin-Code: $SLIPSTREAM_ADMIN_CODE" \
  --output identity/generated_registry.json \
  --review-output identity/generated_registry.review.md
```

The GitHub workflow in `.github/workflows/auto-curate-registry.yml` is manual-only. It can run the direct bootstrap path on demand, but it no longer runs on a schedule.

When the generated registry changes, the workflow opens or updates a PR. Humans only need to review `identity/generated_registry.review.md`, with special attention to:

- new RWA / commodity overrides
- removed overrides

Unknown-class crypto-like overrides are promoted only as exchange-explicit base/quote mappings; they do not create asset aliases or asset classes. Anything that would affect official asset classification must be backed by explicit metadata or a precise allowlist entry.

By default the command is append-safe for automation: it merges new generated rules into the existing `identity/generated_registry.json` and does not delete rules that are missing from the latest discovery response. Use `--prune` only for an intentional full rebuild from a trusted complete export.

This command currently:

- keeps stable-quoted CEX markets as canonical anchors
- includes stable-quoted DEX markets such as Hyperliquid HIP-3 perps
- infers high-confidence Hyperliquid HIP-3 aliases when a namespaced venue ticker uniquely maps to a known RWA CEX base
- focuses on stable quotes such as `USDT / USDC / USD / FDUSD`
- generates explicit `market_overrides`
- generates `asset_aliases` with inferred classes:
  - `fiat_stable`
  - `rwa_stock`
  - `rwa_commodity`
  - default `crypto`

Recommended local verification after regeneration:

```bash
go test ./...
cd frontend
pnpm build
```

It should answer:

- What exchange is this?
- Is this spot / perpetual / future?
- What is the raw venue symbol?
- What is the canonical symbol?
- What is the underlying asset alias / asset class?

It should not answer:

- Whether a signal is good
- Whether an alert should notify
- Whether an opportunity should rank high

## Package layout

```text
identity/
  registry.go
  resolver.go
  types.go
registry/
  default.json
cmd/market-kit-inspect/
  main.go
```

## Basic usage

```go
registry, err := identity.LoadRegistryFile("registry/default.json")
if err != nil {
    panic(err)
}

resolver := identity.NewResolver(registry)
result := resolver.Resolve(identity.ResolveRequest{
    Exchange:       "okx",
    Symbol:         "DRAM-USDT-SWAP",
    MarketTypeHint: "perpetual",
})
```

## Resolve statuses

- `resolved`: one identity is confidently selected
- `ambiguous`: multiple reasonable candidates exist; caller should not guess
- `unresolved`: not enough information or no matching rule

## Local development from another repo

During local development in another project:

```go
replace github.com/solobat/market-kit => ../market-kit
```

Then once stable:

1. push `market-kit`
2. tag a version
3. remove `replace`
4. upgrade dependency normally
