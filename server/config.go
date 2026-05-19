package server

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr               string
	SyncSourcesPath        string
	RequestTimeout         time.Duration
	FrontendDistDir        string
	SlipstreamDiscoveryURL string
	SlipstreamAdminCode    string
	AutoSyncEnabled        bool
	AutoSyncInterval       time.Duration
	AutoSyncSourceID       string
	RuntimeRegistryPath    string
}

func LoadConfig() Config {
	return Config{
		HTTPAddr:               firstNonEmpty(os.Getenv("MARKET_KIT_HTTP_ADDR"), ":18120"),
		SyncSourcesPath:        strings.TrimSpace(os.Getenv("MARKET_KIT_SYNC_SOURCES_PATH")),
		RequestTimeout:         parseDuration(firstNonEmpty(os.Getenv("MARKET_KIT_REQUEST_TIMEOUT"), "12s"), 12*time.Second),
		FrontendDistDir:        firstNonEmpty(os.Getenv("MARKET_KIT_FRONTEND_DIST"), filepathOrDefault()),
		SlipstreamDiscoveryURL: strings.TrimSpace(os.Getenv("MARKET_KIT_SLIPSTREAM_DISCOVERY_URL")),
		SlipstreamAdminCode:    strings.TrimSpace(os.Getenv("MARKET_KIT_SLIPSTREAM_ADMIN_CODE")),
		AutoSyncEnabled:        parseBool(firstNonEmpty(os.Getenv("MARKET_KIT_AUTOSYNC_ENABLED"), "true"), true),
		AutoSyncInterval:       parseDuration(firstNonEmpty(os.Getenv("MARKET_KIT_AUTOSYNC_INTERVAL"), "1m"), time.Minute),
		AutoSyncSourceID:       strings.TrimSpace(os.Getenv("MARKET_KIT_AUTOSYNC_SOURCE")),
		RuntimeRegistryPath:    firstNonEmpty(os.Getenv("MARKET_KIT_RUNTIME_REGISTRY_PATH"), filepath.Join("data", "runtime_generated_registry.json")),
	}
}

func filepathOrDefault() string {
	return "frontend/dist"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	return fallback
}

func parseBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	case "0", "false", "no", "n", "off", "disabled":
		return false
	default:
		return fallback
	}
}
