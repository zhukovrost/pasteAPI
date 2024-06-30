package config

import (
	"errors"
	"flag"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strings"
)

var (
	BuildTime         string
	Version           string
	NeedDebug         bool
	ErrDisplayAndExit = errors.New("display Version and exit")
)

type Config struct {
	Port int    `yaml:"port"`
	Env  string `yaml:"env"`
	DB   struct {
		DSN          string `yaml:"dsn" envconfig:"PASTE_DB_DSN"`
		MaxOpenConns int    `yaml:"maxOpenConns"`
		MaxIdleConns int    `yaml:"maxIdleConns"`
		MaxIdleTime  string `yaml:"maxIdleTime"`
	} `yaml:"db"`
	Limiter struct {
		RPS     float64 `yaml:"rps"`
		Burst   int     `yaml:"burst"`
		Enabled bool    `yaml:"enabled"`
	} `yaml:"limiter"`
	SMTP struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"user" envconfig:"PASTE_SMTP_USER"`
		Password string `envconfig:"PASTE_SMTP_PASSWORD"`
		Sender   string `yaml:"sender"`
	} `yaml:"smtp"`
	CORS struct {
		TrustedOrigins []string `yaml:"trustedOrigins" envconfig:"TRUSTED_ORIGINS"`
	} `yaml:"cors"`
}

func New() (*Config, error) {
	var cfg Config

	err := loadConfig("configs/config.yml", &cfg)
	if err != nil {
		return nil, err
	}

	// Load configuration from environment variables
	err = envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	err = processFlags(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadConfig(filename string, cfg *Config) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
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

	flag.StringVar(&cfg.SMTP.Host, "smtp-host", cfg.SMTP.Host, "SMTP host")
	flag.IntVar(&cfg.SMTP.Port, "smtp-port", cfg.SMTP.Port, "SMTP port")
	flag.StringVar(&cfg.SMTP.Username, "smtp-username", cfg.SMTP.Username, "SMTP username")
	flag.StringVar(&cfg.SMTP.Password, "smtp-password", cfg.SMTP.Password, "SMTP password")
	flag.StringVar(&cfg.SMTP.Sender, "smtp-sender", cfg.SMTP.Sender, "SMTP sender")

	flag.BoolVar(&NeedDebug, "debug", false, "turns on debug level (log)")

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
