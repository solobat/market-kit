package server

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr        string
	SyncSourcesPath string
	RequestTimeout  time.Duration
	FrontendDistDir string
}

func LoadConfig() Config {
	return Config{
		HTTPAddr:        firstNonEmpty(os.Getenv("MARKET_KIT_HTTP_ADDR"), ":18120"),
		SyncSourcesPath: strings.TrimSpace(os.Getenv("MARKET_KIT_SYNC_SOURCES_PATH")),
		RequestTimeout:  parseDuration(firstNonEmpty(os.Getenv("MARKET_KIT_REQUEST_TIMEOUT"), "12s"), 12*time.Second),
		FrontendDistDir: firstNonEmpty(os.Getenv("MARKET_KIT_FRONTEND_DIST"), filepathOrDefault()),
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
