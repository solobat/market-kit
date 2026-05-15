package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/solobat/market-kit/bootstrap"
)

func main() {
	outputPath := flag.String("output", "", "Path to write discovery JSON. Writes to stdout when empty.")
	sourcesFlag := flag.String("sources", "", "Comma-separated exchange ids to fetch, e.g. binance,bybit,okx")
	timeoutFlag := flag.Duration("timeout", 20*time.Second, "HTTP timeout for upstream exchange requests")
	flag.Parse()

	sourceIDs := splitCSV(*sourcesFlag)
	client := &http.Client{Timeout: *timeoutFlag}

	envelope, err := bootstrap.Fetch(context.Background(), client, sourceIDs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap discovery failed: %v\n", err)
		os.Exit(1)
	}

	payload, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal discovery payload failed: %v\n", err)
		os.Exit(1)
	}
	payload = append(payload, '\n')

	if strings.TrimSpace(*outputPath) == "" {
		_, _ = os.Stdout.Write(payload)
		return
	}

	if err := os.WriteFile(*outputPath, payload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s failed: %v\n", *outputPath, err)
		os.Exit(1)
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
