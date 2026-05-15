package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/solobat/market-kit/bootstrap"
)

type SyncSource struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Project string            `json:"project"`
	Kind    string            `json:"kind"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type syncSourceFile struct {
	Sources []SyncSource `json:"sources"`
}

func loadSyncSources(config Config) ([]SyncSource, error) {
	loaded := []SyncSource{}
	candidates := make([]string, 0, 3)
	if config.SyncSourcesPath != "" {
		candidates = append(candidates, config.SyncSourcesPath)
	}
	candidates = append(candidates,
		filepath.Join("frontend", "sync-sources.local.json"),
		filepath.Join("frontend", "sync-sources.example.json"),
	)

	for _, candidate := range candidates {
		payload, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		var parsed syncSourceFile
		if err := json.Unmarshal(payload, &parsed); err != nil {
			return nil, err
		}

		out := make([]SyncSource, 0, len(parsed.Sources))
		for _, source := range parsed.Sources {
			source.ID = strings.TrimSpace(source.ID)
			source.Label = strings.TrimSpace(source.Label)
			source.Project = strings.TrimSpace(source.Project)
			source.Kind = normalizeSyncSourceKind(source)
			source.URL = strings.TrimSpace(source.URL)
			if source.ID == "" || source.URL == "" {
				continue
			}
			if source.Label == "" {
				source.Label = source.ID
			}
			if source.Headers == nil {
				source.Headers = map[string]string{}
			}
			out = append(out, source)
		}
		loaded = out
		break
	}

	loaded = appendBuiltInSources(loaded)
	return loaded, nil
}

func appendBuiltInSources(sources []SyncSource) []SyncSource {
	seen := map[string]bool{}
	for _, source := range sources {
		seen[source.ID] = true
	}

	builtins := []SyncSource{
		{
			ID:      bootstrap.BuiltInSourceID,
			Label:   bootstrap.BuiltInSourceLabel,
			Project: "market-kit",
			Kind:    "discovery",
			URL:     bootstrap.BuiltInSourceURL,
			Headers: map[string]string{},
		},
	}

	for _, builtin := range builtins {
		if seen[builtin.ID] {
			continue
		}
		sources = append(sources, builtin)
	}

	sort.SliceStable(sources, func(i, j int) bool {
		left := sources[i]
		right := sources[j]
		if left.Kind == right.Kind {
			return left.ID < right.ID
		}
		return left.Kind == "discovery"
	})
	return sources
}

func normalizeSyncSourceKind(source SyncSource) string {
	raw := strings.ToLower(strings.TrimSpace(source.Kind))
	switch raw {
	case "discovery", "sample":
		return raw
	}

	project := strings.ToLower(strings.TrimSpace(firstNonEmpty(source.Project, source.ID)))
	switch {
	case strings.Contains(project, "slipstream"),
		strings.Contains(project, "discovery"),
		strings.Contains(project, "bootstrap"):
		return "discovery"
	default:
		return "sample"
	}
}
