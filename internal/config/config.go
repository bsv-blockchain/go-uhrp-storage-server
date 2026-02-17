package config

import (
	"os"
	"strconv"
)

// Config holds all server configuration.
type Config struct {
	ServerPrivateKey  string
	HTTPPort          string
	HostingDomain     string
	BSVNetwork        string
	WalletStorageURL  string
	PricePerGBMonth   float64
	MinHostingMinutes int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ServerPrivateKey:  os.Getenv("SERVER_PRIVATE_KEY"),
		HTTPPort:          getEnvDefault("HTTP_PORT", "8080"),
		HostingDomain:     os.Getenv("HOSTING_DOMAIN"),
		BSVNetwork:        getEnvDefault("BSV_NETWORK", "mainnet"),
		WalletStorageURL:  os.Getenv("WALLET_STORAGE_URL"),
		PricePerGBMonth:   getEnvFloat("PRICE_PER_GB_MO", 0.03),
		MinHostingMinutes: getEnvInt("MIN_HOSTING_MINUTES", 0),
	}
	return cfg, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return def
}
