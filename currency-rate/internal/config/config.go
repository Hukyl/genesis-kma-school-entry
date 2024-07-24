package config

import (
	"log/slog"
	"os"
)

type Config struct {
	Debug                   bool
	CurrencyBeaconAPIKey    string
	RateRefreshCronSpec     string
	RateQueueName           string
	UserQueueName           string
	UserCompensateQueueName string
}

func getOrError(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("value is not set", slog.Any("key", key))
	}
	return value
}

func NewFromEnv() Config {
	return Config{
		Debug:                   getOrError("DEBUG") == "true",
		CurrencyBeaconAPIKey:    getOrError("CURRENCY_BEACON_API_KEY"),
		RateRefreshCronSpec:     getOrError("RATE_REFRESH_CRON_SPEC"),
		RateQueueName:           getOrError("RATE_QUEUE_NAME"),
		UserQueueName:           getOrError("USER_QUEUE_NAME"),
		UserCompensateQueueName: getOrError("USER_COMPENSATE_QUEUE_NAME"),
	}
}
