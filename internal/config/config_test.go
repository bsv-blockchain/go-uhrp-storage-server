package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/config"
)

// configEnvVars lists every variable Load reads, so each test starts from a clean environment.
func configEnvVars() []string {
	return []string{
		"SERVER_PRIVATE_KEY", "HTTP_PORT", "HOSTING_DOMAIN", "BSV_NETWORK", "WALLET_STORAGE_URL",
		"EXCHANGE_RATE_URL", "PRICE_PER_GB_MO", "MIN_HOSTING_MINUTES", "LOG_LEVEL", "LOG_FORMAT", "PUBLIC_DIR", "SLAP_TRACKERS",
	}
}

func clearEnv(t *testing.T) {
	t.Helper()

	for _, key := range configEnvVars() {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Empty(t, cfg.ServerPrivateKey)
	assert.Equal(t, "8080", cfg.HTTPPort)
	assert.Empty(t, cfg.HostingDomain)
	assert.Equal(t, "mainnet", cfg.BSVNetwork)
	assert.Empty(t, cfg.WalletStorageURL)
	assert.Empty(t, cfg.ExchangeRateURL)
	assert.InDelta(t, 0.03, cfg.PricePerGBMonth, 1e-12)
	assert.Equal(t, 0, cfg.MinHostingMinutes)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Equal(t, "public", cfg.PublicDir)
	assert.Equal(t, []string{
		"https://overlay-us-1.bsvb.tech",
		"https://overlay-eu-1.bsvb.tech",
		"https://overlay-ap-1.bsvb.tech",
		"https://users.bapp.dev",
	}, cfg.SLAPTrackers)
}

func TestLoadFromEnvironment(t *testing.T) {
	clearEnv(t)
	t.Setenv("SERVER_PRIVATE_KEY", "abc123")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("HOSTING_DOMAIN", "files.example.com")
	t.Setenv("BSV_NETWORK", "testnet")
	t.Setenv("WALLET_STORAGE_URL", "https://storage.example.com")
	t.Setenv("EXCHANGE_RATE_URL", "http://127.0.0.1:9999/rate")
	t.Setenv("PRICE_PER_GB_MO", "0.25")
	t.Setenv("MIN_HOSTING_MINUTES", "30")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("PUBLIC_DIR", "/srv/uhrp")
	t.Setenv("SLAP_TRACKERS", " https://a.example , ,https://b.example ")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "abc123", cfg.ServerPrivateKey)
	assert.Equal(t, "9090", cfg.HTTPPort)
	assert.Equal(t, "files.example.com", cfg.HostingDomain)
	assert.Equal(t, "testnet", cfg.BSVNetwork)
	assert.Equal(t, "https://storage.example.com", cfg.WalletStorageURL)
	assert.Equal(t, "http://127.0.0.1:9999/rate", cfg.ExchangeRateURL)
	assert.InDelta(t, 0.25, cfg.PricePerGBMonth, 1e-12)
	assert.Equal(t, 30, cfg.MinHostingMinutes)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "text", cfg.LogFormat)
	assert.Equal(t, "/srv/uhrp", cfg.PublicDir)
	assert.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.SLAPTrackers)
}

func TestLoadInvalidNumbersFallBackToDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("PRICE_PER_GB_MO", "cheap")
	t.Setenv("MIN_HOSTING_MINUTES", "ten")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.InDelta(t, 0.03, cfg.PricePerGBMonth, 1e-12)
	assert.Equal(t, 0, cfg.MinHostingMinutes)
}

func TestLoadDefaultTrackersAreNotShared(t *testing.T) {
	clearEnv(t)

	first, err := config.Load()
	require.NoError(t, err)

	first.SLAPTrackers[0] = "mutated"

	second, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "https://overlay-us-1.bsvb.tech", second.SLAPTrackers[0])
}
