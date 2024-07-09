package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

var (
	BuildTime         string
	Version           string
	ErrDisplayAndExit = errors.New("display Version and exit")
)

type Config struct {
	Host           string `yaml:"host" envconfig:"ADDRESS"`
	Port           int    `yaml:"port" envconfig:"PORT"`
	Env            string `yaml:"env" envconfig:"ENVIRONMENT"`
	Status         string `yaml:"status" envconfig:"STATUS"`
	ActivationLink string `yaml:"activation_link" envconfig:"PASTE_ACTIVATION_LINK"`
	ResetLink      string `yaml:"reset_link" envconfig:"PASTE_RESET_LINK"`
	DB             struct {
		DSN          string `yaml:"dsn" envconfig:"PASTE_DB_DSN"`
		MaxOpenConns int    `yaml:"maxOpenConns" envconfig:"PASTE_DB_MAX_OPEN_CONNECTIONS"`
		MaxIdleConns int    `yaml:"maxIdleConns" envconfig:"PASTE_DB_MAX_IDLE_CONNECTIONS"`
		MaxIdleTime  string `yaml:"maxIdleTime" envconfig:"PASTE_DB_MAX_IDLE_TIME"`
	} `yaml:"db"`
	Limiter struct {
		RPS     float64 `yaml:"rps" envconfig:"API_LIMIT_RPS"`
		Burst   int     `yaml:"burst" envconfig:"API_LIMIT_BURST"`
		Enabled bool    `yaml:"enabled" envconfig:"API_LIMIT_ENABLED"`
	} `yaml:"limiter"`
	Logger struct {
		NeedDebug bool `yaml:"need_debug" envconfig:"API_LOGGER_DEBUG"`
	} `yaml:"logger"`
	CORS struct {
		TrustedOrigins []string `yaml:"trustedOrigins" envconfig:"PASTE_TRUSTED_ORIGINS"`
	} `yaml:"cors"`
	Redis struct {
		Host       string `yaml:"host" envconfig:"PASTE_REDIS_ADDRESS"`
		Port       string `yaml:"port" envconfig:"PASTE_REDIS_PORT"`
		Password   string `yaml:"password" envconfig:"PASTE_REDIS_PASSWORD"`
		DB         int    `yaml:"db" envconfig:"PASTE_REDIS_DB"`
		Timeout    string `yaml:"timeout" envconfig:"PASTE_REDIS_TIMEOUT"`
		Expiration string `yaml:"expiration" envconfig:"PASTE_REDIS_EXPIRATION"`
	} `yaml:"redis"`
	RabbitMQ struct {
		URL          string `yaml:"url" envconfig:"RABBITMQ_URL"`
		WaitTime     string `yaml:"wait_time" envconfig:"RABBITMQ_WAIT_TIME"`
		Timeout      string `yaml:"timeout" envconfig:"RABBITMQ_TIMEOUT"`
		Attempts     int    `yaml:"attempts" envconfig:"RABBITMQ_ATTEMPTS"`
		Exchange     string `yaml:"exchange" envconfig:"RABBITMQ_EXCHANGE"`
		ExchangeType string `yaml:"exchange_type" envconfig:"RABBITMQ_EXCHANGE_TYPE"`
		Queue        string `yaml:"queue" envconfig:"RABBITMQ_QUEUE"`
	} `yaml:"rabbitmq"`
}

func New() (*Config, error) {
	var cfg Config

	if err := loadConfig("configs/config.yml", &cfg); err != nil {
		return nil, err
	}
	if err := processEnvironment(&cfg); err != nil {
		return nil, err
	}
	if err := processFlags(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadConfig(filename string, cfg *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config file %s: %w", filename, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to decode YAML from config file %s: %w", filename, err)
	}

	return nil
}

func processEnvironment(cfg *Config) error {
	if err := envconfig.Process("", cfg); err != nil {
		return fmt.Errorf("failed to process environment variables: %w", err)
	}
	return nil
}

func processFlags(cfg *Config) error {
	flag.IntVar(&cfg.Port, "port", cfg.Port, "API server port")
	flag.StringVar(&cfg.Env, "env", cfg.Env, "Environment (development|staging|production)")

	flag.StringVar(&cfg.DB.DSN, "db-dsn", cfg.DB.DSN, "PostgreSQL DSN")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db-max-open-conns", cfg.DB.MaxOpenConns, "PostgreSQL max open connections")
	flag.IntVar(&cfg.DB.MaxIdleConns, "db-max-idle-conns", cfg.DB.MaxIdleConns, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.DB.MaxIdleTime, "db-max-idle-time", cfg.DB.MaxIdleTime, "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.Limiter.RPS, "limiter-rps", cfg.Limiter.RPS, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", cfg.Limiter.Burst, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", cfg.Limiter.Enabled, "Enable rate limiter")

	//flag.StringVar(&cfg.SMTP.Host, "smtp-host", cfg.SMTP.Host, "SMTP host")
	//flag.IntVar(&cfg.SMTP.Port, "smtp-port", cfg.SMTP.Port, "SMTP port")
	//flag.StringVar(&cfg.SMTP.Username, "smtp-username", cfg.SMTP.Username, "SMTP username")
	//flag.StringVar(&cfg.SMTP.Password, "smtp-password", cfg.SMTP.Password, "SMTP password")
	//flag.StringVar(&cfg.SMTP.Sender, "smtp-sender", cfg.SMTP.Sender, "SMTP sender")
	//flag.StringVar(&cfg.SMTP.Timeout, "smtp-timout", cfg.SMTP.Timeout, "SMTP timeout")

	flag.StringVar(&cfg.Redis.Host, "redis-address", cfg.Redis.Host, "Redis address")
	flag.StringVar(&cfg.Redis.Port, "redis-port", cfg.Redis.Port, "Redis port")
	flag.StringVar(&cfg.Redis.Password, "redis-password", cfg.Redis.Password, "Redis password")
	flag.IntVar(&cfg.Redis.DB, "redis-db", cfg.Redis.DB, "Redis database")
	flag.StringVar(&cfg.Redis.Timeout, "redis-timeout", cfg.Redis.Timeout, "Redis timeout")
	flag.StringVar(&cfg.Redis.Expiration, "redis-expiration", cfg.Redis.Expiration, "Redis expiration duration")

	flag.StringVar(&cfg.RabbitMQ.URL, "rabbitmq-url", cfg.RabbitMQ.URL, "rabbitmq url")
	flag.StringVar(&cfg.RabbitMQ.WaitTime, "rabbitmq-wait-time", cfg.RabbitMQ.WaitTime, "rabbitmq wait time between connection tries")
	flag.IntVar(&cfg.RabbitMQ.Attempts, "rabbitmq-attempts", cfg.RabbitMQ.Attempts, "rabbitmq attempt amount to connect")
	flag.StringVar(&cfg.RabbitMQ.Exchange, "rabbitmq-exchange", cfg.RabbitMQ.Exchange, "rabbitmq exchange")
	flag.StringVar(&cfg.RabbitMQ.ExchangeType, "rabbitmq-exchange-type", cfg.RabbitMQ.ExchangeType, "rabbitmq exchange type")
	flag.StringVar(&cfg.RabbitMQ.Queue, "rabbitmq-queue", cfg.RabbitMQ.Queue, "rabbitmq queue name")

	flag.BoolVar(&cfg.Logger.NeedDebug, "debug", cfg.Logger.NeedDebug, "turns on debug level (log)")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.CORS.TrustedOrigins = strings.Fields(val)
		return nil
	})

	displayVersion := flag.Bool("version", false, "Display Version and exit")

	flag.Parse()

	if *displayVersion {
		return ErrDisplayAndExit
	}

	return nil
}
