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

For `slipstream`, the recommended source is:

```json
{
  "id": "slipstream-prod",
  "label": "Slipstream 市场发现",
  "project": "slipstream",
  "url": "https://api.example.com/slipstream/api/discovery/markets?limit=5000",
  "headers": {
    "X-Slipstream-Admin-Code": "replace-me"
  }
}
```

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
- static hosting for the built `frontend/dist` app

That means online deployment no longer depends on the Vite dev proxy.

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
7. upgrade downstream repos to the new tag

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

## Principles

`market-kit` should be the shared identity layer, not a business-logic layer.

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
