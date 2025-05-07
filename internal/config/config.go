package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Addr                    string        `yaml:"ADDRESS"`
	ReadTimeout             time.Duration `yaml:"READ_TIMEOUT"`
	WriteTimeout            time.Duration `yaml:"WRITE_TIMEOUT"`
	IdleTimeout             time.Duration `yaml:"IDLE_TIMEOUT"`
	ShutdownTimeout         time.Duration `yaml:"SHUTDOWN_TIMEOUT"`
	GracefulShutdownTimeout time.Duration `yaml:"GRACEFUL_SHUTDOWN_TIMEOUT"`
}

type Database struct {
	Host            string        `env:"PG_HOST"            env-default:"0.0.0.0" yaml:"PG_HOST"`
	Port            string        `env:"PG_PORT"            env-default:"5432"    yaml:"PG_PORT"`
	User            string        `env:"PG_USER"            env-required:"true"   yaml:"PG_USER"`
	Password        string        `env:"PG_PASSWORD"        env-required:"true"   yaml:"PG_PASSWORD"`
	Name            string        `env:"PG_DBNAME"          env-required:"true"   yaml:"PG_DBNAME"`
	SSLMode         string        `env:"PG_SSLMODE"         env-default:"require" yaml:"PG_SSLMODE"`
	MaxOpenConns    int           `env:"MAX_OPEN_CONNS"     env-default:"25"      yaml:"MAX_OPEN_CONNS"`
	MaxIdleConns    int           `env:"MAX_IDLE_CONNS"     env-default:"10"      yaml:"MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `env:"CONN_MAX_LIFETIME"  env-default:"5m"      yaml:"CONN_MAX_LIFETIME"`
	ConnMaxIdleTime time.Duration `env:"CONN_MAX_IDLE_TIME" env-default:"1m"      yaml:"CONN_MAX_IDLE_TIME"`
}

type RedisConnect struct {
	Host     string `env:"REDIS_HOST"     yaml:"REDIS_HOST"`
	Username string `env:"REDIS_USER"     env-required:"true" yaml:"REDIS_USER"`
	Password string `env:"REDIS_PASSWORD" env-required:"true" yaml:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB"       env-default:"0"     yaml:"REDIS_DB"`
	Port     string `env:"REDIS_PORT"     env-default:"6379"  yaml:"REDIS_PORT"`
}

type RateConfig struct {
	MaxAttempts int64         `env:"MAX_ATTEMPTS" env-default:"5"   yaml:"MAX_ATTEMPTS"`
	WindowSize  time.Duration `env:"WINDOW_SIZE"  env-default:"15s" yaml:"WINDOW_SIZE"`
}

type Stripe struct {
	APIKey              string   `env:"STRIPE_API_KEY"              env-default:""                   yaml:"STRIPE_API_KEY"`
	WebhookSecret       string   `env:"STRIPE_WEBHOOK_SECRET"       env-default:""                   yaml:"STRIPE_WEBHOOK_SECRET"`
	PaymentMethods      []string `env:"STRIPE_PAYMENT_METHODS"      env-default:"card,bank_transfer" yaml:"STRIPE_PAYMENT_METHODS"`
	SupportedCurrencies []string `env:"STRIPE_SUPPORTED_CURRENCIES" env-default:"inr, usd, eur"      yaml:"STRIPE_SUPPORTED_CURRENCIES"`
}

type SendGrid struct {
	APIKey     string `env:"API_KEY"    env-default:""                     yaml:"API_KEY"`
	FromEmail  string `env:"FROM_EMAIL" env-default:"noreply@example.com"  yaml:"FROM_EMAIL"`
	FromName   string `env:"FROM_NAME"  env-default:"Notification Service" yaml:"FROM_NAME"`
	SMSEnabled bool   `env:"SMSENABLED" env-default:"false"                yaml:"SMSENABLED"`
}

type Security struct {
	JWTKey         string `env:"JWT_KEY"          env-required:"true" yaml:"JWT_KEY"`
	JWTExpiryHours int    `env:"JWT_EXPIRY_HOURS" env-default:"24"    yaml:"JWT_EXPIRY_HOURS"`
}

type OTelConfig struct {
	ServiceName      string  `env:"OTEL_SERVICE_NAME"       env-default:"scalable-ecommerce-platform"     yaml:"SERVICE_NAME"`
	ExporterEndpoint string  `env:"OTEL_EXPORTER_ENDPOINT"  env-default:"http://localhost:4318/v1/traces" yaml:"EXPORTER_ENDPOINT"`
	SamplerRatio     float64 `env:"OTEL_TRACES_SAMPLER_ARG" env-default:"1.0"                             yaml:"SAMPLER_RATIO"`
}

type CacheConfig struct {
	DefaultTTL time.Duration `env:"CACHE_DEFAULT_TTL" env-default:"5m" yaml:"default_ttl"`
}

type Config struct {
	Env          string       `env:"ENV"          env-required:"true" yaml:"env"`
	HTTPServer   HTTPServer   `yaml:"http_server"`
	Database     Database     `yaml:"database"`
	RedisConnect RedisConnect `yaml:"redis"`
	RateConfig   RateConfig   `yaml:"rateConfig"`
	Stripe       Stripe       `yaml:"stripe"`
	SendGrid     SendGrid     `yaml:"sendgrid"`
	Security     Security     `yaml:"security"`
	OTel         OTelConfig   `yaml:"otel"`
	Cache        CacheConfig  `yaml:"cache"`
}

func MustLoad() *Config {
	var configPath string

	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {
		flags := flag.String("config", "", "gets the config flag value")

		flag.Parse()

		configPath = *flags

		if configPath == "" {
			defaultPath := "./config/local.yaml"
			if _, err := os.Stat(defaultPath); err == nil {
				configPath = defaultPath
				log.Printf("Config path not specified, using default: %s", configPath)
			} else {
				log.Fatal("Config path is not set and default ./config/local.yaml not found")
			}
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	} else if err != nil {
		log.Fatalf("error accessing config file at %s: %v", configPath, err)
	}

	var cfg Config

	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("cannot read config file: %s", err.Error())
	}

	// Environment variables can override the defaults
	err = cleanenv.ReadEnv(&cfg)
	if err != nil {
		log.Fatalf("cannot read environment variables: %s", err.Error())
	}

	return &cfg
}

func LoadConfigFromPath(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, errors.New("config path is empty")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	var cfg Config

	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %s", err.Error())
	}

	err = cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot read environment variables: %s", err.Error())
	}

	return &cfg, nil
}

func (d *Database) GetDSN() string {
	return fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Name, d.SSLMode)
}

func (r *RedisConnect) GetDSN() string {
	return fmt.Sprintf("redis://%s:%s@%s:%s",
		r.Username,
		r.Password,
		r.Host,
		r.Port,
	)
}
