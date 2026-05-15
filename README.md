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
- `GET /api/registry`
- `GET /api/discovery/sources`
- `GET /api/discovery/sync?source=<id>`
- `GET /api/discovery/lookup?symbol=<symbol>[&source=<id>]`
- static hosting for the built `frontend/dist` app

That means online deployment no longer depends on the Vite dev proxy.

`GET /api/discovery/sync?source=market-kit-bootstrap` will trigger a fresh bootstrap pull from the built-in exchange REST collectors.

`GET /api/discovery/lookup` adds a lightweight presence query layer on top of discovery imports, so you can ask questions such as:

```bash
curl 'http://127.0.0.1:18120/api/discovery/lookup?symbol=TSM'
curl 'http://127.0.0.1:18120/api/discovery/lookup?symbol=DRAM&source=slipstream-prod'
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

### Server environment variables

- `MARKET_KIT_HTTP_ADDR`
  - default: `:18120`
- `MARKET_KIT_SYNC_SOURCES_PATH`
  - optional path to your real sync source config file
- `MARKET_KIT_REQUEST_TIMEOUT`
  - default: `12s`
- `MARKET_KIT_FRONTEND_DIST`
  - default: `frontend/dist`

If `MARKET_KIT_SYNC_SOURCES_PATH` is not set, the server falls back to:

1. `frontend/sync-sources.local.json`
2. `frontend/sync-sources.example.json`

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

- `--sources binance,bybit,okx`
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

This command currently:

- keeps only CEX markets
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
