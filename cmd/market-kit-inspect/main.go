package main

import (
	"encoding/json"
	"flag"
	"log"

	"github.com/solobat/market-kit/identity"
)

func main() {
	registryPath := flag.String("registry", "registry/default.json", "path to registry json")
	exchange := flag.String("exchange", "", "exchange name")
	symbol := flag.String("symbol", "", "raw market symbol")
	marketType := flag.String("market-type", "", "market type hint")
	instType := flag.String("inst-type", "", "inst type hint")
	productType := flag.String("product-type", "", "product type hint")
	canonicalHint := flag.String("canonical-hint", "", "canonical symbol hint")
	flag.Parse()

	registry, err := identity.LoadRegistryFile(*registryPath)
	if err != nil {
		log.Fatalf("load registry: %v", err)
	}

	resolver := identity.NewResolver(registry)
	result := resolver.Resolve(identity.ResolveRequest{
		Exchange:            *exchange,
		Symbol:              *symbol,
		CanonicalSymbolHint: *canonicalHint,
		MarketTypeHint:      *marketType,
		InstType:            *instType,
		ProductType:         *productType,
	})

	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("marshal result: %v", err)
	}
	log.Print(string(payload))
}
