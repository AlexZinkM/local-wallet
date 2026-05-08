package solana

import (
	"context"
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
	statusPendingSubmissionUnknown = "PENDING_SUBMISSION_UNKNOWN"
	statusSuccess                  = "SUCCESS"
	statusFailedOnChain            = "FAILED_ON_CHAIN"
)

var (
	payMutex        sync.Mutex
	idempotencyData = map[string]paymentAttempt{}
)

type paymentAttempt struct {
	Currency  string
	ToAddress string
	Amount    string
	TxID      string
	Status    string
	Message   string
	UpdatedAt time.Time
}

func attemptResponse(attempt paymentAttempt) *model.PayResponse {
	return &model.PayResponse{
		TxID:    attempt.TxID,
		Status:  attempt.Status,
		Message: attempt.Message,
	}
}

func resolveAttemptStatus(ctx context.Context, filePath string, attempt paymentAttempt) (paymentAttempt, error) {
	if attempt.TxID == "" || attempt.Status != statusPendingSubmissionUnknown {
		return attempt, nil
	}

	address, err := crypto.ReadWalletAddress(filePath)
	if err != nil {
		return attempt, fmt.Errorf("failed to read wallet address: %w", err)
	}

	solanaClient, err := client.NewSolanaClient(address)
	if err != nil {
		return attempt, fmt.Errorf("failed to create Solana client: %w", err)
	}

	status, err := solanaClient.GetTransactionStatus(ctx, attempt.TxID)
	if err != nil {
		return attempt, err
	}

	if status == client.TxStatusSuccess {
		attempt.Status = statusSuccess
		attempt.Message = "transaction confirmed"
		attempt.UpdatedAt = time.Now()
		return attempt, nil
	}
	if status == client.TxStatusFailed {
		attempt.Status = statusFailedOnChain
		attempt.Message = "transaction failed on chain"
		attempt.UpdatedAt = time.Now()
		return attempt, nil
	}

	return attempt, nil
}

func precheckIdempotency(ctx context.Context, filePath string, idempotencyKey, currency, toAddress, amount string) (*model.PayResponse, error) {
	payMutex.Lock()

	if idempotencyKey != "" {
		if attempt, ok := idempotencyData[idempotencyKey]; ok {
			if attempt.Currency != currency || attempt.ToAddress != toAddress || attempt.Amount != amount {
				payMutex.Unlock()
				return nil, fmt.Errorf("idempotency key already used with different payment payload")
			}
			payMutex.Unlock()
			updatedAttempt, err := resolveAttemptStatus(ctx, filePath, attempt)
			if err == nil && updatedAttempt != attempt {
				payMutex.Lock()
				idempotencyData[idempotencyKey] = updatedAttempt
				payMutex.Unlock()
				attempt = updatedAttempt
			}
			return attemptResponse(attempt), nil
		}
	}
	payMutex.Unlock()

	return nil, nil
}

func storeAttempt(idempotencyKey, currency, toAddress, amount string, resp *model.PayResponse) {
	if idempotencyKey == "" || resp == nil {
		return
	}
	idempotencyData[idempotencyKey] = paymentAttempt{
		Currency:  currency,
		ToAddress: toAddress,
		Amount:    amount,
		TxID:      resp.TxID,
		Status:    resp.Status,
		Message:   resp.Message,
		UpdatedAt: time.Now(),
	}
}

func storePendingAttempt(idempotencyKey, currency, toAddress, amount, txID string) *model.PayResponse {
	resp := &model.PayResponse{
		TxID:    txID,
		Status:  statusPendingSubmissionUnknown,
		Message: "transaction submission timed out; status is unknown, retry with the same Idempotency-Key to check progress",
	}
	payMutex.Lock()
	storeAttempt(idempotencyKey, currency, toAddress, amount, resp)
	payMutex.Unlock()
	return resp
}

func storeSuccessAttempt(idempotencyKey, currency, toAddress, amount, txID string) *model.PayResponse {
	resp := &model.PayResponse{
		TxID:    txID,
		Status:  statusSuccess,
		Message: "transaction submitted",
	}
	payMutex.Lock()
	storeAttempt(idempotencyKey, currency, toAddress, amount, resp)
	payMutex.Unlock()
	return resp
}

// PayUSDC sends a USDC transaction
// password must be []byte for security (caller should zero it after use)
func PayUSDC(ctx context.Context, filePath string, password []byte, toAddress, amount string, idempotencyKey string) (*model.PayResponse, error) {
	// Validate recipient address
	if !isValidSolanaAddress(toAddress) {
		return nil, fmt.Errorf("invalid Solana address")
	}

	existingResp, err := precheckIdempotency(ctx, filePath, idempotencyKey, "USDC", toAddress, amount)
	if err != nil {
		return nil, err
	}
	if existingResp != nil {
		return existingResp, nil
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
	usdcBalMicro, solBalLamports, err := solanaClient.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}

	// Convert amount to micro units (string-based, no float precision loss)
	usdcAmountMicro, err := common.USDCToMicro(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	if usdcAmountMicro == 0 {
		return nil, fmt.Errorf("amount is too small: minimum is 0.000001 USDC")
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
	txID, err := solanaClient.CreateUSDCTransaction(ctx, toAddress, walletData.PrivateKey, amount)
	if err != nil {
		if client.IsTempororyTxSubmissionError(err) {
			txID = client.ExtractTempororyTxID(err)
			return storePendingAttempt(idempotencyKey, "USDC", toAddress, amount, txID), nil
		}
		return nil, err
	}

	return storeSuccessAttempt(idempotencyKey, "USDC", toAddress, amount, txID), nil
}

// PaySOL sends a SOL transaction
// password must be []byte for security (caller should zero it after use)
func PaySOL(ctx context.Context, filePath string, password []byte, toAddress, amount string, idempotencyKey string) (*model.PayResponse, error) {
	// Validate recipient address
	if !isValidSolanaAddress(toAddress) {
		return nil, fmt.Errorf("invalid Solana address")
	}

	existingResp, err := precheckIdempotency(ctx, filePath, idempotencyKey, "SOL", toAddress, amount)
	if err != nil {
		return nil, err
	}
	if existingResp != nil {
		return existingResp, nil
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
	_, solBalLamports, err := solanaClient.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}

	// Convert amount to lamports (string-based, no float precision loss)
	solAmountLamports, err := common.SOLToLamports(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	if solAmountLamports == 0 {
		return nil, fmt.Errorf("amount is too small: minimum is 0.000000001 SOL")
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
	txID, err := solanaClient.CreateSOLTransaction(ctx, toAddress, walletData.PrivateKey, amount)
	if err != nil {
		if client.IsTempororyTxSubmissionError(err) {
			txID = client.ExtractTempororyTxID(err)
			return storePendingAttempt(idempotencyKey, "SOL", toAddress, amount, txID), nil
		}
		return nil, err
	}

	return storeSuccessAttempt(idempotencyKey, "SOL", toAddress, amount, txID), nil
}

// isValidSolanaAddress validates a Solana address
func isValidSolanaAddress(address string) bool {
	// Try to parse as Solana public key
	_, err := solana.PublicKeyFromBase58(address)
	return err == nil
}
