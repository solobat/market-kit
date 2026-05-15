package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type SyncSource struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Project string            `json:"project"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type syncSourceFile struct {
	Sources []SyncSource `json:"sources"`
}

func loadSyncSources(config Config) ([]SyncSource, error) {
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
		return out, nil
	}

	return nil, nil
}
