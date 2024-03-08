package configuration

import (
	"time"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	DBLogin       string        `env:"PUBLIKA_AUCTION_BOT_DBLGN"`
	DBPwd         string        `env:"PUBLIKA_AUCTION_BOT_DBPWD"`
	DBHost        string        `env:"PUBLIKA_AUCTION_BOT_DBHOST"`
	DBPort        string        `env:"PUBLIKA_AUCTION_BOT_DBPORT"`
	DBSid         string        `env:"PUBLIKA_AUCTION_BOT_DBSID"`
	DBTimeout     time.Duration `env:"PUBLIKA_AUCTION_BOT_DBTIMEOUT"`
	DBMaxOpenConn int           `env:"PUBLIKA_AUCTION_BOT_DBMAXOPENCONN"`

	DBAppLogin       string        `env:"PUBLIKA_AUCTION_BOT_DB_APP_LGN"`
	DBAppPwd         string        `env:"PUBLIKA_AUCTION_BOT_DB_APP_PWD"`
	DBAppHost        string        `env:"PUBLIKA_AUCTION_BOT_DB_APP_HOST"`
	DBAppPort        string        `env:"PUBLIKA_AUCTION_BOT_DB_APP_PORT"`
	DBAppSid         string        `env:"PUBLIKA_AUCTION_BOT_DB_APP_SID"`
	DBAppTimeout     time.Duration `env:"PUBLIKA_AUCTION_BOT_DB_APP_TIMEOUT"`
	DBAppMaxOpenConn int           `env:"PUBLIKA_AUCTION_BOT_DB_APP_MAXOPENCONN"`

	REDIS_ADDR string `env:"PUBLIKA_AUCTION_BOT_REDIS_ADDR"`
	REDIS_PWD  string `env:"PUBLIKA_AUCTION_BOT_REDIS_PWD"`
	REDIS_DB   int    `env:"PUBLIKA_AUCTION_BOT_REDIS_DB"`

	UPDATE_DATA_PERIOD time.Duration `env:"PUBLIKA_AUCTION_BOT_UPDATE_DATA_PERIOD"`

	TG_TOKEN    string `env:"PUBLIKA_AUCTION_BOT_TOKEN"`
	TG_ENDPOINT string `env:"PUBLIKA_AUCTION_BOT_TG_ENDPOINT"`

	Ticker time.Duration `env:"PUBLIKA_AUCTION_BOT_PUSH_PERIOD"`

	ADDR string `env:"PUBLIKA_AUCTION_BOT_ADDR"`
}

func New() Config {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	var c Config
	err := env.Parse(&c)

	if err != nil {
		panic(err)
	}
	return c
}
