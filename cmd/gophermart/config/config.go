package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServiceAddress string `env:"RUN_ADDRESS"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseURI    string `env:"DATABASE_URI"`
	Key            string `env:"KEY"`
	DebugMode      bool
}

func BuildConfig() (Config, error) {
	var cfg Config
	cfg.buildFromFlags()
	err := cfg.buildFromEnv()
	return cfg, err
}

func (cfg *Config) buildFromFlags() {
	flag.StringVar(&cfg.ServiceAddress, "a", "127.0.0.1:8080", "service address")
	flag.StringVar(&cfg.AccrualAddress, "r", "", "accrual system address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database dsn")
	flag.StringVar(&cfg.Key, "k", "1337qwerty", "key for passwords hashing")
	flag.BoolVar(&cfg.DebugMode, "deb", false, "is debug mode enabled")
	flag.Parse()
}

func (cfg *Config) buildFromEnv() error {
	err := env.Parse(cfg)
	if err != nil {
		log.Println("config::buildFromEnv::error: in env parsing:", err)
	}
	return err
}
