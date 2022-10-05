package server

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address     string `env:"RUN_ADDRESS"`
	DatabaseURI string `env:"DATABASE_URI"`
}

func NewConfig() (*Config, error) {
	cfg := parseFlags()

	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseFlags() *Config {
	var cfg Config

	flag.StringVar(&cfg.Address, "a", "", "address and port for server")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database address")
	flag.Parse()

	return &cfg
}
