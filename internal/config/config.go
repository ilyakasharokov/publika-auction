package config

import (
	"os"
	"time"

	"github.com/caarlos0/env"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	// Telegram Bot
	BotToken     string        `env:"PUBLIKA_AUCTION_BOT_TOKEN" envDefault:""`
	BotAddr      string        `env:"PUBLIKA_AUCTION_BOT_ADDR" envDefault:":8002"`
	TgEndpoint   string        `env:"PUBLIKA_AUCTION_BOT_TG_ENDPOINT" envDefault:"https://api.telegram.org/bot%s/%s"`
	UpdatePeriod time.Duration `env:"PUBLIKA_AUCTION_BOT_UPDATE_DATA_PERIOD" envDefault:"5m"`

	// Redis
	RedisAddr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RedisDB   int    `env:"REDIS_DB" envDefault:"0"`

	// MongoDB
	MongoURI string `env:"MONGO_URI" envDefault:"mongodb://localhost:27017"`
	MongoDB  string `env:"MONGO_DB" envDefault:"publika_auction"`

	// Logging
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// Graylog (optional)
	GraylogEndpoint string `env:"GRAYLOG_ENDPOINT" envDefault:""`
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.BotToken == "" {
		return ErrMissingBotToken
	}
	return nil
}

var (
	ErrMissingBotToken = os.ErrInvalid
)
