//go:build darwin || linux

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/solobat/market-kit/discovery"
	"github.com/solobat/market-kit/identity"
)

// Optional real-inventory replay: no exchange network access and no writes to
// production. Pass a saved discovery/current response via this environment key.
func TestFullInventoryReplay(t *testing.T) {
	path := os.Getenv("MARKET_KIT_TEST_INVENTORY")
	if path == "" {
		t.Skip("set MARKET_KIT_TEST_INVENTORY to a saved full discovery response")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wrapped struct {
		Payload discovery.ImportEnvelope `json:"payload"`
	}
	if err = json.Unmarshal(raw, &wrapped); err != nil {
		t.Fatal(err)
	}
	if len(wrapped.Payload.Items) == 0 {
		t.Fatal("expected a non-empty saved discovery/current response")
	}
	payload, err := json.Marshal(wrapped.Payload)
	if err != nil {
		t.Fatal(err)
	}
	app := manualTestApp(t)
	app.baseRegistry, err = identity.LoadDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	app.reviewedRegistry, err = identity.LoadReviewedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	app.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(payload))), Header: make(http.Header)}, nil
	})}
	var before, after syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &before)
	started := time.Now()
	if err = app.runAutoSyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	syncTime := time.Since(started)
	syscall.Getrusage(syscall.RUSAGE_SELF, &after)
	reviewed := app.reviewedRegistry.MarketOverrides
	verify := func() {
		t.Helper()
		resolver := app.runtimeResolver()
		for _, v := range reviewed {
			r := resolver.Resolve(identity.ResolveRequest{Exchange: v.Exchange, Symbol: v.RawSymbol, MarketTypeHint: v.MarketType})
			if r.Market == nil || r.Market.AssetID != v.AssetID || r.Market.AssetClass != v.AssetClass || r.Market.UnitMultiplier != v.UnitMultiplier {
				t.Fatalf("reviewed identity lost after full sync: %s %s %+v", v.Exchange, v.RawSymbol, r)
			}
		}
	}
	verify()
	requests := make([]registryOverrideRequest, 0, 500)
	for _, v := range reviewed[:500] {
		requests = append(requests, registryOverrideRequest{Exchange: v.Exchange, RawSymbol: v.RawSymbol, MarketType: v.MarketType, CanonicalSymbol: v.CanonicalSymbol, AssetClass: v.AssetClass, AssetID: v.AssetID, UnderlyingID: v.UnderlyingID, ComparisonKey: v.ComparisonKey, ComparisonStatus: string(v.ComparisonStatus), InstrumentKind: v.InstrumentKind, SettlementAsset: v.SettlementAsset, UnitMultiplier: v.UnitMultiplier})
	}
	started = time.Now()
	if _, err = app.commitManualOverrides(requests); err != nil {
		t.Fatal(err)
	}
	batchTime := time.Since(started)
	started = time.Now()
	if _, err = app.commitManualOverrides(requests); err != nil {
		t.Fatal(err)
	}
	noOpTime := time.Since(started)
	if err = app.runAutoSyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	verify()
	// Reader latency while an administrative registry build is in flight.
	done := make(chan error, 1)
	go func() {
		q := registryOverrideRequest{Exchange: "test", RawSymbol: "REPLAYUSDT", MarketType: "spot", CanonicalSymbol: "REPLAY/USDT", AssetClass: "crypto"}
		_, e := app.commitManualOverrides([]registryOverrideRequest{q})
		done <- e
	}()
	latencies := make([]int64, 20000)
	for i := range latencies {
		q := wrapped.Payload.Items[i%len(wrapped.Payload.Items)]
		start := time.Now()
		app.runtimeResolver().Resolve(identity.ResolveRequest{Exchange: q.PlatformID, Symbol: q.Symbol, MarketTypeHint: q.MarketType})
		latencies[i] = time.Since(start).Nanoseconds()
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	cpuMicros := (after.Utime.Sec-before.Utime.Sec)*1000000 + int64(after.Utime.Usec-before.Utime.Usec) + (after.Stime.Sec-before.Stime.Sec)*1000000 + int64(after.Stime.Usec-before.Stime.Usec)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	t.Log(fmt.Sprintf("rows=%d reviewed=%d sync_wall=%s sync_cpu_us=%d batch_500=%s batch_500_noop=%s reader_p50_ns=%d reader_p99_ns=%d heap_mb=%.1f", len(wrapped.Payload.Items), len(reviewed), syncTime, cpuMicros, batchTime, noOpTime, latencies[len(latencies)/2], latencies[len(latencies)*99/100], float64(mem.HeapAlloc)/1024/1024))
}
