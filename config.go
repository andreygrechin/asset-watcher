package main

import (
	"log"
	"strings"

	env "github.com/caarlos0/env/v11"
)

// Config represents the configuration structure.
type Config struct {
	OrgID           string `env:"ASSET_WATCHER_ORG_ID,required,notEmpty"`
	Debug           bool   `env:"ASSET_WATCHER_DEBUG"                    envDefault:"false"`
	OutputFormat    string `env:"ASSET_WATCHER_OUTPUT_FORMAT"            envDefault:"table"`
	ExcludeReserved bool   `env:"ASSET_WATCHER_EXCLUDE_RESERVED"         envDefault:"false"`
	ExcludeProjects string `env:"ASSET_WATCHER_EXCLUDE_PROJECTS"         envDefault:""`
	IncludeProjects string `env:"ASSET_WATCHER_INCLUDE_PROJECTS"         envDefault:""`
}

// GetConfig returns the configuration structure.
func GetConfig() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v\n", err)
	}

	if cfg.ExcludeProjects != "" && cfg.IncludeProjects != "" {
		log.Fatal("cannot set both ASSET_WATCHER_EXCLUDE_PROJECTS and ASSET_WATCHER_INCLUDE_PROJECTS at the same time\n")
	}

	if strings.ToLower(cfg.OutputFormat) != "table" && strings.ToLower(cfg.OutputFormat) != "json" {
		log.Fatalf("invalid value for ASSET_WATCHER_OUTPUT_FORMAT: %s. "+
			"Allowed values are 'table' or 'json'\n", cfg.OutputFormat)
	}

	return &cfg
}
