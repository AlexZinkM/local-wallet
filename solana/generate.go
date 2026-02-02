package solana

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AlexZinkM/local-wallet/internal/crypto"
	"github.com/AlexZinkM/local-wallet/internal/model"

	"github.com/gagliardetto/solana-go"
	"github.com/skip2/go-qrcode"
)

const (
	networkSolana = "solana"
)

// FileExistsError is an error when file already exists and is not empty
type FileExistsError struct {
	Message string
}

func (e *FileExistsError) Error() string {
	return e.Message
}

// IsFileExistsError checks if error is FileExistsError
func IsFileExistsError(err error) bool {
	_, ok := err.(*FileExistsError)
	return ok
}

// GenerateWallet generates a new Solana wallet and saves it to .cwt file.
// Returns the generated public address on success.
// password must be []byte for security (caller should zero it after use)
func GenerateWallet(filePath string, password []byte) (address string, err error) {
	// Check file extension (.cwt)
	ext := filepath.Ext(filePath) // e.g. "wallet.cwt" → ".cwt"
	if ext != ".cwt" {
		return "", fmt.Errorf("file must have .cwt extension")
	}

	// Check file existence
	if _, err := os.Stat(filePath); err == nil {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}
		if fileInfo.Size() > 0 {
			return "", &FileExistsError{Message: "file is not empty"}
		}
	}

	// Generate new Solana keypair
	wallet := solana.NewWallet()
	defer clear(wallet.PrivateKey)

	// Get address (public key)
	address = wallet.PublicKey().String()

	// Generate QR code
	qrCode, err := generateQRCode(address)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Prepare wallet data - PrivateKey stored as []byte (will be base64 encoded in JSON)
	walletData := &model.WalletData{
		PrivateKey: wallet.PrivateKey,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	// Encrypt and write to file
	if err := crypto.EncryptWallet(filePath, networkSolana, address, qrCode, walletData, password); err != nil {
		return "", fmt.Errorf("failed to encrypt wallet: %w", err)
	}

	return address, nil
}

// generateQRCode generates QR code of address in base64
func generateQRCode(address string) (string, error) {
	qr, err := qrcode.New(address, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to create QR code: %w", err)
	}

	// Get PNG image
	png, err := qr.PNG(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate PNG: %w", err)
	}

	// Encode to base64
	return base64.StdEncoding.EncodeToString(png), nil
}
