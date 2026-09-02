package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all gateway configuration
type Config struct {
	Backends  []BackendConfig `yaml:"backends"`  // Multiple backends
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Cache     CacheConfig     `yaml:"cache"`
	Cost      CostConfig      `yaml:"cost"`
	Server    ServerConfig    `yaml:"server"`
	Health    HealthConfig    `yaml:"health"`
}

// BackendConfig holds configuration for a single backend
type BackendConfig struct {
	ID       string        `yaml:"id"`
	URL      string        `yaml:"url"`
	Model    string        `yaml:"model"`
	Capacity int           `yaml:"capacity"`
	Timeout  time.Duration `yaml:"timeout"`
	MaxRetries int         `yaml:"max_retries"`
}

// EngineConfig holds LLM engine connection settings (deprecated, use Backends)
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

// HealthConfig holds health check settings
type HealthConfig struct {
	CheckInterval time.Duration `yaml:"check_interval"`
	CheckTimeout  time.Duration `yaml:"check_timeout"`
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
	// Support legacy single engine config
	if len(cfg.Backends) == 0 {
		// Legacy format compatibility - will be added by main.go if Engine is set
	}
	
	// Backend defaults
	for i := range cfg.Backends {
		if cfg.Backends[i].Timeout == 0 {
			cfg.Backends[i].Timeout = 120 * time.Second
		}
		if cfg.Backends[i].MaxRetries == 0 {
			cfg.Backends[i].MaxRetries = 1
		}
		if cfg.Backends[i].Capacity == 0 {
			cfg.Backends[i].Capacity = 8
		}
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
	if cfg.Health.CheckInterval == 0 {
		cfg.Health.CheckInterval = 10 * time.Second
	}
	if cfg.Health.CheckTimeout == 0 {
		cfg.Health.CheckTimeout = 5 * time.Second
	}

	return &cfg, nil
}
