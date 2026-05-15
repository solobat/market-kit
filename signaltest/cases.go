package signaltest

type Payload struct {
	Source              string
	SourceSchemaVersion string
	Kind                string
	CanonicalSymbol     string
	Symbol              string
	SymbolA             string
	SymbolB             string
	PrimarySymbol       string
	SecondarySymbol     string
	ExchangeA           string
	ExchangeB           string
	PrimaryExchange     string
	SecondaryExchange   string
	MarketTypeA         string
	MarketTypeB         string
	PrimaryMarketType   string
	SecondaryMarketType string
	ObservedValueBps    float64
	Direction           string
	Metadata            map[string]string
}

type ReplayExpectation struct {
	FinalStatus          string
	PrimaryStatus        string
	SecondaryStatus      string
	PrimaryVenueSymbol   string
	SecondaryVenueSymbol string
	ErrorContains        string
}

type Case struct {
	Name        string
	Description string
	Payload     Payload
	Replay      ReplayExpectation
}

const (
	IdentityStatusResolved   = "resolved"
	IdentityStatusUnresolved = "unresolved"

	ProcessingStatusConfirmed = "confirmed"
	ProcessingStatusFailed    = "failed"

	HyperliquidAmbiguousCLMessage = "ambiguous hyperliquid venue for CL/USDT"
	ExplicitHIP3RequiredMessage   = "explicit HIP-3 symbol required"
)

func TradfiMonitorKMBMNRSpread() Case {
	return Case{
		Name:        "tradfi-monitor-km-bmnr-spread",
		Description: "tradfi-monitor should preserve the Hyperliquid HIP-3 venue symbol for BMNR spreads sent to veridex.",
		Payload: Payload{
			Source:              "tradi-monitor",
			SourceSchemaVersion: "tradi-monitor.v1",
			Kind:                "spread_arbitrage",
			CanonicalSymbol:     "BMNR/USDT",
			Symbol:              "km:BMNR",
			SymbolA:             "km:BMNR",
			SymbolB:             "BMNRUSDT",
			PrimarySymbol:       "km:BMNR",
			SecondarySymbol:     "BMNRUSDT",
			ExchangeA:           "hyperliquid",
			ExchangeB:           "binance",
			PrimaryExchange:     "hyperliquid",
			SecondaryExchange:   "binance",
			MarketTypeA:         "perpetual",
			MarketTypeB:         "perpetual",
			PrimaryMarketType:   "perpetual",
			SecondaryMarketType: "perpetual",
			ObservedValueBps:    125,
			Direction:           "short-binance-long-hyperliquid",
			Metadata: map[string]string{
				"monitorId":   "fixture:bmnr",
				"asset":       "BMNR",
				"veridexMode": "test_notify",
			},
		},
		Replay: ReplayExpectation{
			FinalStatus:          ProcessingStatusConfirmed,
			PrimaryStatus:        IdentityStatusResolved,
			SecondaryStatus:      IdentityStatusResolved,
			PrimaryVenueSymbol:   "KM:BMNR",
			SecondaryVenueSymbol: "BMNRUSDT",
		},
	}
}

func MarketWatchUSOBasis() Case {
	return Case{
		Name:        "market-watch-uso-basis",
		Description: "market-watch should upgrade canonical USO/USDT into the unique Hyperliquid HIP-3 venue symbol when syncing to veridex.",
		Payload: Payload{
			Source:              "market-platform",
			SourceSchemaVersion: "market-watch.v1",
			Kind:                "basis",
			CanonicalSymbol:     "USO/USDT",
			Symbol:              "km:USOIL",
			SymbolA:             "km:USOIL",
			PrimarySymbol:       "km:USOIL",
			ExchangeA:           "hyperliquid",
			PrimaryExchange:     "hyperliquid",
			MarketTypeA:         "perpetual",
			PrimaryMarketType:   "perpetual",
			ObservedValueBps:    275,
			Direction:           "hyperliquid-basis",
			Metadata: map[string]string{
				"veridexMode": "test_notify",
			},
		},
		Replay: ReplayExpectation{
			FinalStatus:        ProcessingStatusConfirmed,
			PrimaryStatus:      IdentityStatusResolved,
			PrimaryVenueSymbol: "KM:USOIL",
		},
	}
}

func VeridexAmbiguousCLSpread() Case {
	return Case{
		Name:        "veridex-ambiguous-cl-spread",
		Description: "veridex should fail fast when it receives an ambiguous Hyperliquid CL canonical spread without an explicit HIP-3 symbol.",
		Payload: Payload{
			Source:              "tradi-monitor",
			SourceSchemaVersion: "tradi-monitor.v1",
			Kind:                "spread_arbitrage",
			CanonicalSymbol:     "CL/USDT",
			Symbol:              "CL/USDT",
			SymbolA:             "CL/USDT",
			SymbolB:             "CLUSDT",
			PrimarySymbol:       "CL/USDT",
			SecondarySymbol:     "CLUSDT",
			ExchangeA:           "hyperliquid",
			ExchangeB:           "binance",
			PrimaryExchange:     "hyperliquid",
			SecondaryExchange:   "binance",
			MarketTypeA:         "perpetual",
			MarketTypeB:         "perpetual",
			PrimaryMarketType:   "perpetual",
			SecondaryMarketType: "perpetual",
			ObservedValueBps:    110,
			Direction:           "short-binance-long-hyperliquid",
			Metadata: map[string]string{
				"veridexMode": "test_notify",
			},
		},
		Replay: ReplayExpectation{
			FinalStatus:   ProcessingStatusFailed,
			PrimaryStatus: IdentityStatusUnresolved,
			ErrorContains: HyperliquidAmbiguousCLMessage,
		},
	}
}

func IngestionReplayCases() []Case {
	return []Case{
		TradfiMonitorKMBMNRSpread(),
		MarketWatchUSOBasis(),
	}
}
