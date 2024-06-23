package config

type Config struct {
	port uint8
	env  string
}

func New() (*Config, error) {
	return &Config{}, nil
}
