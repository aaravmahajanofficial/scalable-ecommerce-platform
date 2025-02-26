package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Addr string `yaml:"address"`
}

type Database struct {
	Host     string `yaml:"PG_HOST" env:"PG_HOST" env-default:"localhost"`
	Port     string `yaml:"PG_PORT" env:"PG_PORT" env-default:"5432"`
	User     string `yaml:"PG_USER" env:"PG_USER" env-required:"true"`
	Password string `yaml:"PG_PASSWORD" env:"PG_PASSWORD" env-required:"true"`
	Name     string `yaml:"PG_DBNAME" env:"PG_DBNAME" env-required:"true"`
	SSLMode  string `yaml:"PG_SSLMODE" env:"PG_SSLMODE" env-default:"require"`
}

type RedisConnect struct {
	Host     string `yaml:"REDIS_HOST" env:"REDIS_HOST"`
	Username string `yaml:"REDIS_USER" env:"REDIS_USER" env-required:"true"`
	Password string `yaml:"REDIS_PASSWORD" env:"REDIS_PASSWORD" env-required:"true"`
	DB       int    `yaml:"REDIS_DB" env:"REDIS_DB" env-default:"0"`
}

type RateConfig struct {
	MaxAttempts int64         `yaml:"MAX_ATTEMPTS" env:"MAX_ATTEMPTS" env-default:"5"`
	WindowSize  time.Duration `yaml:"WINDOW_SIZE" env:"WINDOW_SIZE" env-default:"15s"`
}

type Config struct {
	Env          string `yaml:"env" env:"ENV" env-required:"true"`
	StoragePath  string `yaml:"storage_path" env-required:"true"`
	HTTPServer   `yaml:"http_server"`
	Database     Database     `yaml:"database"`
	RedisConnect RedisConnect `yaml:"redis"`
	RateConfig   RateConfig   `yaml:"rateConfig"`
}

func MustLoad() *Config {

	var configPath string

	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {

		flags := flag.String("config", "", "gets the config flag value")

		flag.Parse()

		configPath = *flags

		if configPath == "" {

			log.Fatal("Config path is not set")

		}

	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	err := cleanenv.ReadConfig(configPath, &cfg)

	if err != nil {

		log.Fatalf("can not read config file: %s", err.Error())
	}

	return &cfg

}

func (d *Database) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		d.User, d.Password, d.Host, d.Port, d.Name)
}
