package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the service.
type Config struct {
	HTTPPort          string
	LogLevel          string
	
	// Kafka Config
	KafkaBrokers      []string
	KafkaTopic        string
	KafkaDLQTopic     string
	KafkaMaxRetries   int
	KafkaRetryBackoff time.Duration
	KafkaWriteTimeout time.Duration

	// Ingestion Config
	WorkerPoolSize    int
	QueueSize         int
	RateLimitRPS      float64
	RateLimitBurst    int
}

// LoadFromEnv loads configuration from environment variables.
// It uses sensible defaults for missing values.
func LoadFromEnv() *Config {
	return &Config{
		HTTPPort:          getEnv("HTTP_PORT", "8080"),
		LogLevel:          getEnv("LOG_LEVEL", "INFO"),
		
		KafkaBrokers:      strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:        getEnv("KAFKA_TOPIC", "events"),
		KafkaDLQTopic:     getEnv("KAFKA_DLQ_TOPIC", "events-dlq"),
		KafkaMaxRetries:   getEnvInt("KAFKA_MAX_RETRIES", 3),
		KafkaRetryBackoff: getEnvDuration("KAFKA_RETRY_BACKOFF", 100*time.Millisecond),
		KafkaWriteTimeout: getEnvDuration("KAFKA_WRITE_TIMEOUT", 10*time.Second),
		
		WorkerPoolSize:    getEnvInt("WORKER_POOL_SIZE", 10),
		QueueSize:         getEnvInt("QUEUE_SIZE", 1000),
		RateLimitRPS:      getEnvFloat("RATE_LIMIT_RPS", 1000.0),
		RateLimitBurst:    getEnvInt("RATE_LIMIT_BURST", 100),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
