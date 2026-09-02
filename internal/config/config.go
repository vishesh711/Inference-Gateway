package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all gateway configuration
type Config struct {
	Engine    EngineConfig    `yaml:"engine"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Cache     CacheConfig     `yaml:"cache"`
	Cost      CostConfig      `yaml:"cost"`
	Server    ServerConfig    `yaml:"server"`
}

// EngineConfig holds LLM engine connection settings
type EngineConfig struct {
	URL         string        `yaml:"url"`
	Timeout     time.Duration `yaml:"timeout"`
	MaxRetries  int           `yaml:"max_retries"`
}

// SchedulerConfig controls admission and concurrency
type SchedulerConfig struct {
	MaxInFlight      int           `yaml:"max_in_flight"`
	QueueSize        int           `yaml:"queue_size"`
	EmbedMaxBatch    int           `yaml:"embed_max_batch"`
	EmbedMaxWaitMs   int           `yaml:"embed_max_wait_ms"`
}

// CacheConfig controls response caching
type CacheConfig struct {
	Enabled    bool          `yaml:"enabled"`
	MaxEntries int           `yaml:"max_entries"`
	TTL        time.Duration `yaml:"ttl"`
}

// CostConfig holds cost accounting parameters
type CostConfig struct {
	GPUHourlyRate float64 `yaml:"gpu_hourly_rate"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port            int           `yaml:"port"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Set defaults
	if cfg.Engine.Timeout == 0 {
		cfg.Engine.Timeout = 120 * time.Second
	}
	if cfg.Engine.MaxRetries == 0 {
		cfg.Engine.MaxRetries = 1
	}
	if cfg.Scheduler.MaxInFlight == 0 {
		cfg.Scheduler.MaxInFlight = 8
	}
	if cfg.Scheduler.QueueSize == 0 {
		cfg.Scheduler.QueueSize = 100
	}
	if cfg.Scheduler.EmbedMaxBatch == 0 {
		cfg.Scheduler.EmbedMaxBatch = 32
	}
	if cfg.Scheduler.EmbedMaxWaitMs == 0 {
		cfg.Scheduler.EmbedMaxWaitMs = 20
	}
	if cfg.Cache.MaxEntries == 0 {
		cfg.Cache.MaxEntries = 1000
	}
	if cfg.Cache.TTL == 0 {
		cfg.Cache.TTL = 300 * time.Second
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8000
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 30 * time.Second
	}

	return &cfg, nil
}
