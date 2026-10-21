package discovery

import (
	"github.com/solobat/market-kit/identity"
	"testing"
	"time"
)

func TestSecurityDescriptionKey(t *testing.T) {
	cases := []struct{ region, raw, want string }{
		{"HK", "HK0992", "HK:00992"}, {"HK", "992HK", "HK:00992"}, {"HK", "09992", "HK:09992"},
		{"CN", "688825", "CN:688825"}, {"KR", "5930", "KR:005930"}, {"KR", "KR005930", "KR:005930"},
		{"US", "BRK.B", "US:BRK.B"}, {"US", "BRK-B", "US:BRK-B"},
		{"", "992", ""}, {"HK", "00000", ""}, {"HK", "992HKX", ""}, {"CN", "UNITREE", ""},
	}
	for _, tc := range cases {
		if got := SecurityDescriptionKey(tc.region, tc.raw); got != tc.want {
			t.Errorf("%s %s = %s, want %s", tc.region, tc.raw, got, tc.want)
		}
	}
}
func TestInstrumentDescriptionMatchesSecurityWithoutMergingIdentity(t *testing.T) {
	now := time.Now().UTC()
	target := CandidateMarket{Exchange: "binance", BaseAsset: "HK0992", AssetClass: "rwa_stock", UnderlyingID: "equity:ticker:hk0992", ComparisonKey: "original/USDT", ComparisonStatus: identity.ComparisonEligible, InstrumentMetadata: &InstrumentMetadata{Source: "binance", MarketRegion: "HK", UnderlyingTicker: "HK0992", NativeQuoteCurrency: "HKD"}}
	donor := CandidateMarket{Exchange: "bybit", BaseAsset: "LENOVO", AssetClass: "rwa_stock", UnderlyingID: "equity:ticker:lenovo", ComparisonStatus: identity.ComparisonEligible, InstrumentMetadata: &InstrumentMetadata{FullName: "Lenovo Group Limited", MarketRegion: "HK", UnderlyingTicker: "992HK", SymbolType: "stock", Source: "bybit", SourceURL: "https://api.bybit.com/v5/market/instruments-info", ObservedAt: now}}
	rows := []CandidateMarket{target, donor}
	enrichInstrumentDescriptions(rows)
	m := rows[0].InstrumentMetadata
	if m.FullName != "Lenovo Group Limited" || m.DescriptionSource != "bybit" || m.DescriptionMatchMethod != "market_and_security_code" || m.Source != "binance" || m.NativeQuoteCurrency != "HKD" {
		t.Fatalf("metadata: %+v", m)
	}
	if rows[0].UnderlyingID != target.UnderlyingID || rows[0].ComparisonKey != target.ComparisonKey || rows[0].BaseAsset != "HK0992" {
		t.Fatal("description enrichment changed identity")
	}
	if target.InstrumentMetadata.FullName != "" {
		t.Fatal("input metadata mutated")
	}
	for _, tc := range []struct {
		name   string
		change func(*CandidateMarket)
	}{
		{"different HK code", func(r *CandidateMarket) { r.InstrumentMetadata.UnderlyingTicker = "09992" }},
		{"different market", func(r *CandidateMarket) { r.InstrumentMetadata.MarketRegion = "US" }},
		{"crypto", func(r *CandidateMarket) { r.AssetClass = "crypto" }},
		{"ambiguous", func(r *CandidateMarket) { r.ComparisonStatus = identity.ComparisonAmbiguous }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := donor
			meta := *donor.InstrumentMetadata
			d.InstrumentMetadata = &meta
			tc.change(&d)
			r := []CandidateMarket{target, d}
			enrichInstrumentDescriptions(r)
			if r[0].InstrumentMetadata.FullName != "" {
				t.Fatalf("wrong enrichment: %+v", r[0])
			}
		})
	}
	conflict := donor
	meta := *donor.InstrumentMetadata
	meta.FullName = "Conflict"
	conflict.InstrumentMetadata = &meta
	rows = []CandidateMarket{target, donor, conflict}
	enrichInstrumentDescriptions(rows)
	if rows[0].InstrumentMetadata.FullName != "" {
		t.Fatal("conflicting name selected")
	}
}

func TestDescriptionsUseReviewedIdentityAcrossVenueCodes(t *testing.T) {
	for _, region := range []string{"CN", "KR", "US"} {
		donor := CandidateMarket{Exchange: "bybit", AssetClass: "rwa_stock", UnderlyingID: "reviewed:issuer", ComparisonStatus: identity.ComparisonEligible, InstrumentMetadata: &InstrumentMetadata{FullName: "Issuer", MarketRegion: region, UnderlyingTicker: "123", SymbolType: "stock", Source: "bybit"}}
		target := CandidateMarket{Exchange: "qfex", AssetClass: "rwa_stock", UnderlyingID: "reviewed:issuer", ComparisonStatus: identity.ComparisonEligible, InstrumentMetadata: &InstrumentMetadata{NativeQuoteCurrency: "CNY", Source: "qfex"}}
		rows := []CandidateMarket{target, donor}
		enrichInstrumentDescriptions(rows)
		if rows[0].InstrumentMetadata.FullName != "Issuer" || rows[0].InstrumentMetadata.NativeQuoteCurrency != "CNY" {
			t.Fatalf("%s metadata not preserved", region)
		}
	}
}
