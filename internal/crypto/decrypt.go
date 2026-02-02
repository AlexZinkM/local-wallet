package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"cwt/internal/model"

	"golang.org/x/crypto/scrypt"
)

// DecryptWallet reads and decrypts .cwt file
// password must be []byte for security (caller should zero it after use)
func DecryptWallet(filePath string, password []byte) (*model.CWTFile, *model.WalletData, error) {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, errors.New("file does not exist")
		}
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check that file is not empty
	if fileInfo.Size() == 0 {
		return nil, nil, errors.New("file is empty")
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Skip UTF-8 BOM if present
	if len(fileData) >= 3 && fileData[0] == 0xEF && fileData[1] == 0xBB && fileData[2] == 0xBF {
		fileData = fileData[3:]
	}

	// Deserialize file structure
	var cwtFile model.CWTFile
	if err := json.Unmarshal(fileData, &cwtFile); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal cwt file: %w", err)
	}

	// Decode salt and nonce
	salt, err := base64.StdEncoding.DecodeString(cwtFile.Salt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(cwtFile.Nonce)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(cwtFile.CipherText)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Derive key from password
	key, err := scrypt.Key(password, salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, nil, errors.New("invalid password")
	}
	defer clear(plaintext) // wipe decrypted bytes from memory

	// Deserialize wallet data
	var walletData model.WalletData
	if err := json.Unmarshal(plaintext, &walletData); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}

	return &cwtFile, &walletData, nil
}

// ReadWalletAddress reads only the address from .cwt file (without decryption)
func ReadWalletAddress(filePath string) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("file does not exist")
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() == 0 {
		return "", errors.New("file is empty")
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Skip UTF-8 BOM if present
	if len(fileData) >= 3 && fileData[0] == 0xEF && fileData[1] == 0xBB && fileData[2] == 0xBF {
		fileData = fileData[3:]
	}

	var cwtFile model.CWTFile
	if err := json.Unmarshal(fileData, &cwtFile); err != nil {
		return "", fmt.Errorf("failed to unmarshal cwt file: %w", err)
	}

	return cwtFile.Address, nil
}
