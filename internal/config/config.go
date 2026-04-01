package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Browser   BrowserConfig   `yaml:"browser"`
	Queue     QueueConfig     `yaml:"queue"`
	Auth      AuthConfig      `yaml:"auth"`
	Plugins   PluginsConfig   `yaml:"plugins"`
	MCP       MCPConfig       `yaml:"mcp"`
	Webhook   WebhookConfig   `yaml:"webhook"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Log       LogConfig       `yaml:"log"`
	Health    HealthConfig    `yaml:"health"`
	Storage   StorageConfig   `yaml:"storage"`
	Recording RecordingConfig `yaml:"recording"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type BrowserConfig struct {
	MaxInstances  int           `yaml:"max_instances"`
	LaunchTimeout time.Duration `yaml:"launch_timeout"`
	IdleTimeout   time.Duration `yaml:"idle_timeout"`
	Headless      bool          `yaml:"headless"`
	NoSandbox     bool          `yaml:"no_sandbox"`
	SharePool     bool          `yaml:"share_pool"` // inject CDP endpoint into plugins
	WarmPoolSize  int           `yaml:"warm_pool_size"` // pre-launch N browsers
	RecycleAfter  int           `yaml:"recycle_after"`  // recycle browser after N uses
}

type QueueConfig struct {
	MaxConcurrent int `yaml:"max_concurrent"`
	MaxPending    int `yaml:"max_pending"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	APIKey  string `yaml:"api_key"`
}

type PluginsConfig struct {
	Dir         string        `yaml:"dir"`
	LoadTimeout time.Duration `yaml:"load_timeout"`
	CallTimeout time.Duration `yaml:"call_timeout"` // per-call timeout, 0 = use request ctx
	PoolSize    int           `yaml:"pool_size"`    // concurrent processes per plugin
}

type MCPConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Transport string `yaml:"transport"`
}

type WebhookConfig struct {
	Enabled       bool          `yaml:"enabled"`
	MaxRetries    int           `yaml:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval"`
}

type SchedulerConfig struct {
	Enabled bool `yaml:"enabled"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// HealthConfig controls system-level overload protection (inspired by Browserless)
type HealthConfig struct {
	Enabled     bool `yaml:"enabled"`
	CPULimit    int  `yaml:"cpu_limit"`    // percent, default 90
	MemoryLimit int  `yaml:"memory_limit"` // percent, default 85
}

// StorageConfig selects persistence backend
type StorageConfig struct {
	Driver     string `yaml:"driver"`      // "memory" | "sqlite"
	SQLitePath string `yaml:"sqlite_path"` // path to .db file
}

// RecordingConfig controls CDP screencast recording
type RecordingConfig struct {
	Enabled bool   `yaml:"enabled"`
	Format  string `yaml:"format"`  // "screencast" | "frames" | "both"
	FPS     int    `yaml:"fps"`     // frames per second for screencast
	Quality int    `yaml:"quality"` // 0-100
	Dir     string `yaml:"dir"`     // output directory
}

// TelemetryConfig for OpenTelemetry tracing
type TelemetryConfig struct {
	Enabled      bool   `yaml:"enabled"`
	OTLPEndpoint string `yaml:"otlp_endpoint"`
	ServiceName  string `yaml:"service_name"`
}

func Default() *Config {
	return &Config{
		Server:    ServerConfig{Port: 3000, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second},
		Browser:   BrowserConfig{MaxInstances: 5, LaunchTimeout: 30 * time.Second, IdleTimeout: 5 * time.Minute, Headless: true, NoSandbox: true, SharePool: true, WarmPoolSize: 2, RecycleAfter: 50},
		Queue:     QueueConfig{MaxConcurrent: 10, MaxPending: 100},
		Auth:      AuthConfig{Enabled: false},
		Plugins:   PluginsConfig{Dir: "./bin/plugins", LoadTimeout: 10 * time.Second, CallTimeout: 60 * time.Second, PoolSize: 3},
		MCP:       MCPConfig{Enabled: false, Transport: "stdio"},
		Webhook:   WebhookConfig{Enabled: false, MaxRetries: 3, RetryInterval: 5 * time.Second},
		Scheduler: SchedulerConfig{Enabled: false},
		Log:       LogConfig{Level: "info", Format: "json"},
		Health:    HealthConfig{Enabled: true, CPULimit: 90, MemoryLimit: 85},
		Storage:   StorageConfig{Driver: "memory", SQLitePath: "./spectra.db"},
		Recording: RecordingConfig{Enabled: false, Format: "both", FPS: 5, Quality: 80, Dir: "./recordings"},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config: %w", err)
			}
		}
	}

	applyEnv(cfg)
	return cfg, validate(cfg)
}

func applyEnv(cfg *Config) {
	envInt := func(key string, dst *int) {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	envBool := func(key string, dst *bool) {
		if v := os.Getenv(key); v != "" {
			*dst = strings.EqualFold(v, "true")
		}
	}
	envStr := func(key string, dst *string) {
		if v := os.Getenv(key); v != "" {
			*dst = v
		}
	}

	envInt("SPECTRA_SERVER_PORT", &cfg.Server.Port)
	envInt("SPECTRA_BROWSER_MAX_INSTANCES", &cfg.Browser.MaxInstances)
	envBool("SPECTRA_BROWSER_HEADLESS", &cfg.Browser.Headless)
	envBool("SPECTRA_BROWSER_SHARE_POOL", &cfg.Browser.SharePool)
	envInt("SPECTRA_QUEUE_MAX_CONCURRENT", &cfg.Queue.MaxConcurrent)
	envBool("SPECTRA_AUTH_ENABLED", &cfg.Auth.Enabled)
	envStr("SPECTRA_AUTH_API_KEY", &cfg.Auth.APIKey)
	envStr("SPECTRA_PLUGINS_DIR", &cfg.Plugins.Dir)
	envInt("SPECTRA_PLUGINS_POOL_SIZE", &cfg.Plugins.PoolSize)
	envStr("SPECTRA_LOG_LEVEL", &cfg.Log.Level)
	envStr("SPECTRA_LOG_FORMAT", &cfg.Log.Format)
	envBool("SPECTRA_MCP_ENABLED", &cfg.MCP.Enabled)
	envBool("SPECTRA_WEBHOOK_ENABLED", &cfg.Webhook.Enabled)
	envBool("SPECTRA_SCHEDULER_ENABLED", &cfg.Scheduler.Enabled)
	envBool("SPECTRA_HEALTH_ENABLED", &cfg.Health.Enabled)
	envStr("SPECTRA_STORAGE_DRIVER", &cfg.Storage.Driver)
	envStr("SPECTRA_STORAGE_SQLITE_PATH", &cfg.Storage.SQLitePath)
	envBool("SPECTRA_RECORDING_ENABLED", &cfg.Recording.Enabled)
}

func validate(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}
	if cfg.Browser.MaxInstances < 1 {
		return fmt.Errorf("browser max_instances must be >= 1")
	}
	if cfg.Queue.MaxConcurrent < 1 {
		return fmt.Errorf("queue max_concurrent must be >= 1")
	}
	if cfg.Queue.MaxPending < 1 {
		return fmt.Errorf("queue max_pending must be >= 1")
	}
	if cfg.Plugins.PoolSize < 1 {
		cfg.Plugins.PoolSize = 1
	}
	return nil
}
