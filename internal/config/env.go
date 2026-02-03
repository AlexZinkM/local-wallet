package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
	"golang.org/x/term"
)

// Config contains all configuration parameters for the application.
// Note: Password is prompted at runtime and stored in memory - use GetSolanaPasswordBytes()
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

var passwordBytes []byte

// PromptForPassword prompts the user for the wallet password in the terminal.
// The password is read without echoing (hidden input) and stored in memory.
// Call this at startup before the server begins handling requests.
func PromptForPassword() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("stdin is not a terminal: run the app interactively to enter password")
	}
	fmt.Fprint(os.Stderr, "Enter wallet password: ")
	defer fmt.Fprintln(os.Stderr)

	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	if len(raw) == 0 {
		return errors.New("password cannot be empty")
	}

	passwordBytes = make([]byte, len(raw))
	copy(passwordBytes, raw)
	clear(raw)
	return nil
}

// GetSolanaPasswordBytes returns the password stored in memory (from PromptForPassword).
// Returns an error if the password was not set.
// Caller must zero the returned slice after use for security.
func GetSolanaPasswordBytes() ([]byte, error) {
	if len(passwordBytes) == 0 {
		return nil, errors.New("password not set: call PromptForPassword at startup")
	}
	out := make([]byte, len(passwordBytes))
	copy(out, passwordBytes)
	return out, nil
}
