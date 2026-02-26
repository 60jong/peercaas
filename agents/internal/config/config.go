package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Worker   WorkerConfig   `mapstructure:"worker"`
	Client   ClientConfig   `mapstructure:"client"`
}

type ServerConfig struct {
	Name string `mapstructure:"name"`
}

type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

type WorkerConfig struct {
	WorkerID    string `mapstructure:"worker_id"`
	ResultQueue string `mapstructure:"result_queue"`
	Concurrency int    `mapstructure:"concurrency"`
	HubURL      string `mapstructure:"hub_url"`
}

type ClientConfig struct {
	PublishQueue   string        `mapstructure:"publish_queue"`
	ReportInterval time.Duration `mapstructure:"report_interval"`
}

func (r *RabbitMQConfig) GetURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		r.Username, r.Password, r.Host, r.Port, r.VHost,
	)
}

func Load(configName string) *Config {
	// SetConfigFile로 명시적 경로 지정 (AddConfigPath + SetConfigName 방식 우회)
	candidates := []string{
		fmt.Sprintf("configs/%s.yaml", configName),
		fmt.Sprintf("%s.yaml", configName),
		fmt.Sprintf("../configs/%s.yaml", configName),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			viper.SetConfigFile(path)
			log.Printf("Config file found: %s", path)
			break
		}
	}

	viper.SetEnvPrefix("AGENT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("worker.worker_id", "WORKER_ID")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Config file '%s' not found. Relying on Env vars.", configName)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Config unmarshal failed: %v", err)
	}

	return &cfg
}
