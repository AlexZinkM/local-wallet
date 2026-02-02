package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cwt/internal/model"
	"golang.org/x/crypto/scrypt"
)

const (
	// scrypt parameters for local wallet
	// Security is prioritized over performance
	//
	// N=2^18 (~256MB RAM, 0.5-2s) - optimal balance:
	//   - Maximum security while remaining compatible with mobile devices
	//   - Works on phones (4-16GB RAM) and desktops alike
	//   - Brute-force attacks remain extremely expensive
	//
	// Note: N=2^20 (~1GB) offers the highiest security but fails on mobile due to
	// Android memory limits per app (~256-512MB typically)
	scryptN      = 1 << 18
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32
	saltLen      = 32
	nonceLen     = 12
)

// EncryptWallet encrypts wallet data and writes it to .cwt
// password must be []byte for security (caller should zero it after use)
func EncryptWallet(filePath string, network, address, qrCode string, walletData *model.WalletData, password []byte) error {
	// Check file extension (should be .cwt)
	if !strings.HasSuffix(filePath, ".cwt") {
		return errors.New("file must have .cwt extension")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); err == nil {
		// File exists, check that it's not empty
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		if fileInfo.Size() > 0 {
			return fmt.Errorf("file is not empty: %w", os.ErrExist)
		}
	}

	// Generate salt and nonce
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Derive key from password
	key, err := scrypt.Key(password, salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Serialize wallet data
	plaintext, err := json.Marshal(walletData)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet data: %w", err)
	}
	defer clear(plaintext) // wipe plaintext bytes from memory

	// Encrypt
	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)

	// Create file structure
	cwtFile := model.CWTFile{
		Network:    network,
		Address:    address,
		QR:         qrCode,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		CipherText: base64.StdEncoding.EncodeToString(ciphertext),
	}

	// Serialize to JSON
	fileData, err := json.MarshalIndent(cwtFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cwt file: %w", err)
	}

	// Add UTF-8 BOM for proper display in Windows
	utf8BOM := []byte{0xEF, 0xBB, 0xBF}
	fileDataWithBOM := append(utf8BOM, fileData...)

	// Write to file
	if err := os.WriteFile(filePath, fileDataWithBOM, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

