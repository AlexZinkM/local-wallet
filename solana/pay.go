package solana

import (
	"fmt"
	"sync"
	"time"

	"github.com/AlexZinkM/local-wallet/internal/client"
	"github.com/AlexZinkM/local-wallet/internal/common"
	"github.com/AlexZinkM/local-wallet/internal/crypto"
	"github.com/AlexZinkM/local-wallet/internal/model"

	"github.com/gagliardetto/solana-go"
)

const (
	solFeeLamports = 5000 // Fee in lamports (0.000005 SOL)
)

var (
	lastPayTime time.Time
	payMutex    sync.Mutex
)

// PayUSDC sends a USDC transaction
// password must be []byte for security (caller should zero it after use)
func PayUSDC(filePath string, password []byte, toAddress, amount string, cooldownMinutes int) (*model.PayResponse, error) {
	// Validate recipient address
	if !isValidSolanaAddress(toAddress) {
		return nil, fmt.Errorf("invalid Solana address")
	}

	// Check cooldown
	payMutex.Lock()
	defer payMutex.Unlock()

	if !lastPayTime.IsZero() {
		cooldownDuration := time.Duration(cooldownMinutes) * time.Minute
		if time.Since(lastPayTime) < cooldownDuration {
			remaining := cooldownDuration - time.Since(lastPayTime)
			return nil, fmt.Errorf("cooldown active, please wait %v", remaining.Round(time.Second))
		}
	}

	// Read address from file
	address, err := crypto.ReadWalletAddress(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet address: %w", err)
	}

	// Decrypt private key
	_, walletData, err := crypto.DecryptWallet(filePath, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt wallet: %w", err)
	}

	// Always clear private key from memory
	defer clear(walletData.PrivateKey)

	// Verify private key length (we store full 64-byte key)
	if len(walletData.PrivateKey) != 64 {
		return nil, fmt.Errorf("invalid private key length")
	}

	// Get public key from address
	fromPubkey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	wallet := solana.PrivateKey(walletData.PrivateKey)

	// Verify wallet matches from address
	if !wallet.PublicKey().Equals(fromPubkey) {
		return nil, fmt.Errorf("private key does not match address")
	}

	// Create client
	solanaClient, err := client.NewSolanaClient(address)
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	// Check balance (raw units: USDC micro, SOL lamports)
	usdcBalMicro, solBalLamports, err := solanaClient.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}

	// Convert amount to micro units (string-based, no float precision loss)
	usdcAmountMicro, err := common.USDCToMicro(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Check USDC sufficiency
	if usdcBalMicro < usdcAmountMicro {
		return nil, fmt.Errorf("insufficient USDC balance")
	}

	// Check SOL sufficiency for fee
	if solBalLamports < solFeeLamports {
		return nil, fmt.Errorf("insufficient SOL for transaction fee (fee: %s SOL). Have: %s SOL",
			common.LamportsToSOL(solFeeLamports), common.LamportsToSOL(solBalLamports))
	}

	// Create and send transaction
	txID, err := solanaClient.CreateUSDCTransaction(toAddress, walletData.PrivateKey, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Save transaction time
	lastPayTime = time.Now()

	return &model.PayResponse{
		TxID: txID,
	}, nil
}

// PaySOL sends a SOL transaction
// password must be []byte for security (caller should zero it after use)
func PaySOL(filePath string, password []byte, toAddress, amount string, cooldownMinutes int) (*model.PayResponse, error) {
	// Validate recipient address
	if !isValidSolanaAddress(toAddress) {
		return nil, fmt.Errorf("invalid Solana address")
	}

	// Check cooldown
	payMutex.Lock()
	defer payMutex.Unlock()

	if !lastPayTime.IsZero() {
		cooldownDuration := time.Duration(cooldownMinutes) * time.Minute
		if time.Since(lastPayTime) < cooldownDuration {
			remaining := cooldownDuration - time.Since(lastPayTime)
			return nil, fmt.Errorf("cooldown active, please wait %v", remaining.Round(time.Second))
		}
	}

	// Read address from file
	address, err := crypto.ReadWalletAddress(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet address: %w", err)
	}

	// Decrypt private key
	_, walletData, err := crypto.DecryptWallet(filePath, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt wallet: %w", err)
	}

	// Always clear private key from memory
	defer clear(walletData.PrivateKey)

	// Verify private key length (we store full 64-byte key)
	if len(walletData.PrivateKey) != 64 {
		return nil, fmt.Errorf("invalid private key length")
	}

	// Get public key from address
	fromPubkey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	// Use full 64-byte private key directly
	wallet := solana.PrivateKey(walletData.PrivateKey)

	// Verify wallet matches from address
	if !wallet.PublicKey().Equals(fromPubkey) {
		return nil, fmt.Errorf("private key does not match address")
	}

	// Create client
	solanaClient, err := client.NewSolanaClient(address)
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	// Check balance (lamports)
	_, solBalLamports, err := solanaClient.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}

	// Convert amount to lamports (string-based, no float precision loss)
	solAmountLamports, err := common.SOLToLamports(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Check SOL sufficiency (amount + fee)
	requiredLamports := solAmountLamports + solFeeLamports
	if solBalLamports < requiredLamports {
		// Calculate max amount user can send
		var maxLamports uint64
		if solBalLamports > solFeeLamports {
			maxLamports = solBalLamports - solFeeLamports
		}
		return nil, fmt.Errorf("insufficient SOL balance. Transaction fee: %s SOL. Max you can send: %s SOL",
			common.LamportsToSOL(solFeeLamports), common.LamportsToSOL(maxLamports))
	}

	// Create and send transaction
	txID, err := solanaClient.CreateSOLTransaction(toAddress, walletData.PrivateKey, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Save transaction time
	lastPayTime = time.Now()

	return &model.PayResponse{
		TxID: txID,
	}, nil
}

// isValidSolanaAddress validates a Solana address
func isValidSolanaAddress(address string) bool {
	// Try to parse as Solana public key
	_, err := solana.PublicKeyFromBase58(address)
	return err == nil
}
