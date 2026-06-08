package configuration

import (
	"time"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	MongoURI string `env:"PUBLIKA_AUCTION_BOT_MONGO_URI" envDefault:"mongodb://localhost:27017"`
	MongoDB  string `env:"PUBLIKA_AUCTION_BOT_MONGO_DB" envDefault:"auction"`

	REDIS_ADDR string `env:"PUBLIKA_AUCTION_BOT_REDIS_ADDR" envDefault:"localhost:6379"`
	REDIS_PWD  string `env:"PUBLIKA_AUCTION_BOT_REDIS_PWD"`
	REDIS_DB   int    `env:"PUBLIKA_AUCTION_BOT_REDIS_DB"`

	TG_TOKEN    string `env:"PUBLIKA_AUCTION_BOT_TOKEN"`
	TG_ENDPOINT string `env:"PUBLIKA_AUCTION_BOT_TG_ENDPOINT" envDefault:"https://api.telegram.org/bot%s/%s"`

	ADDR string `env:"PUBLIKA_AUCTION_BOT_ADDR" envDefault:":8002"`

	AdminUser     string `env:"PUBLIKA_AUCTION_BOT_ADMIN_USER" envDefault:"admin"`
	AdminPassword string `env:"PUBLIKA_AUCTION_BOT_ADMIN_PASSWORD" envDefault:"changeme"`
	SessionSecret string `env:"PUBLIKA_AUCTION_BOT_SESSION_SECRET" envDefault:"change-this-secret-key"`

	BidStep int `env:"PUBLIKA_AUCTION_BOT_BID_STEP" envDefault:"2000"`

	UPDATE_DATA_PERIOD time.Duration `env:"PUBLIKA_AUCTION_BOT_UPDATE_DATA_PERIOD" envDefault:"5m"`
}

func New() Config {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	var c Config
	if err := env.Parse(&c); err != nil {
		panic(err)
	}
	return c
}
