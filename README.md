# market-kit

`market-kit` is a small shared Go module for market identity normalization across independent repos.

It is designed for systems like:

- signal producers such as `tradfi-monitor`
- verification services such as `veridex`
- downstream derived-data projects

The goal is to centralize:

- exchange alias normalization
- market type inference
- venue symbol normalization
- canonical symbol / asset identity mapping
- explicit `resolved / ambiguous / unresolved` outcomes

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
replace github.com/tomasyang/market-kit => ../market-kit
```

Then once stable:

1. push `market-kit`
2. tag a version
3. remove `replace`
4. upgrade dependency normally
