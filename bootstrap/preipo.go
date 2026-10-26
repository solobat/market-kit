package bootstrap

import (
	"context"
	"github.com/solobat/market-kit/discovery"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var preIPORows = regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
var preIPOCells = regexp.MustCompile(`(?is)<td\b[^>]*>(.*?)</td>`)
var preIPOTags = regexp.MustCompile(`<[^>]+>`)
var shareIssuer = regexp.MustCompile(`(?i)\bshare(?:s)? of ([A-Za-z][A-Za-z .&-]*?)(?:,| class | common stock| stock)`)
var legalSuffix = regexp.MustCompile(`(?i)\s+(?:PBC|Inc\.?|Corporation|Corp\.?|Group|Ltd\.?|Limited)$`)
var issuerChars = regexp.MustCompile(`[^A-Z0-9]`)

func xyzPreIPOIssuers(data []byte) map[string]string {
	out := map[string]string{}
	clean := func(b []byte) string {
		return strings.TrimSpace(html.UnescapeString(preIPOTags.ReplaceAllString(string(b), "")))
	}
	for _, row := range preIPORows.FindAllSubmatch(data, -1) {
		cells := preIPOCells.FindAllSubmatch(row[1], -1)
		if len(cells) < 3 || clean(cells[1][1]) != "Live" {
			continue
		}
		matches := shareIssuer.FindAllStringSubmatch(clean(cells[2][1]), -1)
		if len(matches) != 1 {
			continue
		}
		name := strings.TrimSpace(matches[0][1])
		for {
			next := legalSuffix.ReplaceAllString(name, "")
			if next == name {
				break
			}
			name = next
		}
		issuer := issuerChars.ReplaceAllString(strings.ToUpper(name), "")
		if issuer != "" {
			out["xyz:"+clean(cells[0][1])] = issuer
		}
	}
	return out
}
func enrichXYZPreIPO(ctx context.Context, client *http.Client, items []discovery.ImportedMarket) {
	hasXYZ := false
	for _, item := range items {
		if strings.HasPrefix(item.Symbol, "xyz:") {
			hasXYZ = true
			break
		}
	}
	if !hasXYZ {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", discovery.XYZPreIPOURL, nil)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return
	}
	issuers := xyzPreIPOIssuers(data)
	for i := range items {
		if issuer := issuers[items[i].Symbol]; issuer != "" && items[i].InstrumentMetadata != nil {
			items[i].InstrumentMetadata.PreIPOIssuer = issuer
			items[i].InstrumentMetadata.PreIPOEvidenceURL = discovery.XYZPreIPOURL
			items[i].BaseAsset = issuer
			items[i].AssetClass = "rwa_stock"
		}
	}
}
