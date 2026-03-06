// Package config provides configuration loading and management for PeerCaaS agents.
package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the unified configuration for all agent types.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Worker   WorkerConfig   `mapstructure:"worker"`
	Client   ClientConfig   `mapstructure:"client"`
}

// ServerConfig holds general server-related settings.
type ServerConfig struct {
	Name string `mapstructure:"name"`
}

// RabbitMQConfig contains connection details for the RabbitMQ message broker.
type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

// WorkerConfig defines specific settings for the worker agent.
type WorkerConfig struct {
	WorkerID      string  `mapstructure:"worker_id"`
	WorkerKey     string  `mapstructure:"worker_key"`
	ResultQueue   string  `mapstructure:"result_queue"`
	Concurrency   int     `mapstructure:"concurrency"`
	HubURL        string  `mapstructure:"hub_url"`
	MaxCPU        float64 `mapstructure:"max_cpu"`
	MaxMemoryMb   int64   `mapstructure:"max_memory_mb"`
	VMURL         string  `mapstructure:"vm_url"`
	VMUser        string  `mapstructure:"vm_user"`
	VMPass        string  `mapstructure:"vm_pass"`
}

// ClientConfig defines specific settings for the client agent.
type ClientConfig struct {
	PublishQueue   string        `mapstructure:"publish_queue"`
	ReportInterval time.Duration `mapstructure:"report_interval"`
}

// GetURL returns the RabbitMQ connection string formatted from the configuration.
func (r *RabbitMQConfig) GetURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		r.Username, r.Password, r.Host, r.Port, r.VHost,
	)
}

// GenerateWorkerID creates a deterministic, unique ID based on the WorkerKey.
func (w *WorkerConfig) GenerateWorkerID() string {
	if w.WorkerKey == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(w.WorkerKey))
	// Use first 12 characters of the hex hash for a readable yet unique ID (prefix wk- for worker)
	return fmt.Sprintf("wk-%x", hash)[:15]
}

// Load reads the configuration from files and environment variables.
// It searches for config files in standard locations and overrides them with environment variables.
func Load(configName string) (*Config, error) {
	v := viper.New()

	candidates := []string{
		fmt.Sprintf("configs/%s.yaml", configName),
		fmt.Sprintf("%s.yaml", configName),
		fmt.Sprintf("../configs/%s.yaml", configName),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			v.SetConfigFile(path)
			break
		}
	}

	v.SetEnvPrefix("AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind specific environment variables to match naming conventions
	_ = v.BindEnv("worker.worker_id", "WORKER_ID")
	_ = v.BindEnv("worker.worker_key", "WORKER_KEY")
	_ = v.BindEnv("worker.max_cpu", "MAX_CPU")
	_ = v.BindEnv("worker.max_memory_mb", "MAX_MEMORY_MB")

	if err := v.ReadInConfig(); err != nil {
		// Log warning but continue if configuration can be satisfied by environment variables
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return &cfg, nil
}
