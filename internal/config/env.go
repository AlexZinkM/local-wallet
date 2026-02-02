package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
)

// Config contains all configuration parameters for the application.
// Note: SolanaPassword is NOT stored here for security - use GetSolanaPasswordBytes() instead
type Config struct {
	Port           string `envconfig:"PORT" default:"8080"`
	PayCooldown    int    `envconfig:"PAY_COOLDOWN_MINUTES" default:"4"`
	SolanaFilePath string `envconfig:"SOLANA_FILE_PATH" required:"true"`
	SolanaRPCURL   string `envconfig:"SOLANA_RPC_URL" default:"https://api.mainnet-beta.solana.com"`
}

// cfg is the global configuration instance
var cfg *Config

// Init loads configuration from environment variables.
func Init() error {
	cfg = &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return fmt.Errorf("failed to process config: %w", err)
	}
	return nil
}

// Get returns the global configuration instance.
// Panics if Init() was not called.
func Get() *Config {
	if cfg == nil {
		panic("config not initialized, call Init() first")
	}
	return cfg
}

// GetPort returns port from configuration
func GetPort() string {
	return Get().Port
}

// GetPayCooldown returns cooldown in minutes from configuration
func GetPayCooldown() int {
	return Get().PayCooldown
}

// GetSolanaFilePath returns path to .cwt file from configuration
func GetSolanaFilePath() string {
	return Get().SolanaFilePath
}

// GetSolanaRPCURL returns Solana RPC URL from configuration
func GetSolanaRPCURL() string {
	return Get().SolanaRPCURL
}

// GetSolanaPasswordBytes returns password as []byte directly from environment variable.
// Returns an error if SOLANA_PASSWORD is not set.
// Note: os.Getenv() returns string which may be allocated in heap, but we minimize its lifetime
// by immediately copying to []byte. The original string will be GC'd after this function returns.
// Caller must zero the returned slice after use for security
func GetSolanaPasswordBytes() ([]byte, error) {
	envVal := os.Getenv("SOLANA_PASSWORD")
	if envVal == "" {
		return nil, errors.New("SOLANA_PASSWORD not set")
	}
	return []byte(envVal), nil
}
