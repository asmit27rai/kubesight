package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Sampling SamplingConfig `yaml:"sampling"`
	Storage  StorageConfig  `yaml:"storage"`
}

type ServerConfig struct {
	Host string `yaml:"host" env:"SERVER_HOST" default:"0.0.0.0"`
	Port int    `yaml:"port" env:"SERVER_PORT" default:"8080"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers" env:"KAFKA_BROKERS" default:"localhost:9092"`
	Topics  Topics   `yaml:"topics"`
}

type Topics struct {
	Metrics string `yaml:"metrics" default:"k8s-metrics"`
	Logs    string `yaml:"logs" default:"k8s-logs"`
	Events  string `yaml:"events" default:"k8s-events"`
}

type SamplingConfig struct {
	DefaultRate     float64 `yaml:"default_rate" default:"0.05"`
	IncidentRate    float64 `yaml:"incident_rate" default:"0.5"`
	ReservoirSize   int     `yaml:"reservoir_size" default:"10000"`
	WindowSizeMin   int     `yaml:"window_size_min" default:"60"`
	AdaptiveEnabled bool    `yaml:"adaptive_enabled" default:"true"`
}

type StorageConfig struct {
	HLLPrecision int `yaml:"hll_precision" default:"14"`
	CMSWidth     int `yaml:"cms_width" default:"2048"`
	CMSDepth     int `yaml:"cms_depth" default:"5"`
	BloomSize    int `yaml:"bloom_size" default:"1000000"`
	BloomHashes  int `yaml:"bloom_hashes" default:"5"`
}

func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	config.Server.Host = getEnvOrDefault("SERVER_HOST", "0.0.0.0")
	config.Server.Port = 8080
	config.Kafka.Brokers = []string{getEnvOrDefault("KAFKA_BROKERS", "localhost:9092")}
	config.Kafka.Topics.Metrics = "k8s-metrics"
	config.Kafka.Topics.Logs = "k8s-logs"
	config.Kafka.Topics.Events = "k8s-events"
	config.Sampling.DefaultRate = 0.05
	config.Sampling.IncidentRate = 0.5
	config.Sampling.ReservoirSize = 10000
	config.Sampling.WindowSizeMin = 60
	config.Sampling.AdaptiveEnabled = true
	config.Storage.HLLPrecision = 14
	config.Storage.CMSWidth = 2048
	config.Storage.CMSDepth = 5
	config.Storage.BloomSize = 1000000
	config.Storage.BloomHashes = 5

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %v", err)
		}
	}

	return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
