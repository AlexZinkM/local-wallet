// One-off: decrypt old-format wallet, re-encrypt in new format, same salt+nonce. Output: new cipherText only.
// Usage: go run ./cmd/reencrypt_cipher
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/AlexZinkM/local-wallet/internal/model"

	"golang.org/x/crypto/scrypt"
)

const (
	scryptN      = 1 << 18
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32
)

func main() {
	password := []byte("dev")
	saltB64 := "jRnjq5L/lUjcv4V2fn53/e20aIB4Fm7X+p/GS1iCNmg="
	nonceB64 := "nzVu5jdpG9ACKYuu"
	cipherTextB64 := "7mnrsqNqZYcnyKxmDcyodJlrFrP3seEoz6wregFLjcscMFlfGp1tqE5JbusGh7fpGrgB5rxVXTdbqHk0jJoSP6khZJVjoFGG7ZpddywRZo4upUTnHZcy8Emi9LlGM5a+2OlfCorA4nLm2035CLys3MQebjDA6eHnlQYvG/Bit4iLiAqVBNwu208="

	salt, _ := base64.StdEncoding.DecodeString(saltB64)
	nonce, _ := base64.StdEncoding.DecodeString(nonceB64)
	ciphertext, _ := base64.StdEncoding.DecodeString(cipherTextB64)

	key, err := scrypt.Key(password, salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	block, _ := aes.NewCipher(key)
	aesGCM, _ := cipher.NewGCM(block)
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "decrypt failed:", err)
		os.Exit(1)
	}

	// Old format: privateKey is hex string in JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(plaintext, &raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pkStr, _ := raw["privateKey"].(string)
	createdAt, _ := raw["createdAt"].(string)
	if pkStr == "" || len(pkStr) != 64 {
		fmt.Fprintln(os.Stderr, "invalid old privateKey format")
		os.Exit(1)
	}

	seed, err := hex.DecodeString(pkStr)
	if err != nil || len(seed) != 32 {
		fmt.Fprintln(os.Stderr, "hex decode failed")
		os.Exit(1)
	}

	// New format: privateKey is []byte (JSON will base64-encode it)
	newWallet := &model.WalletData{
		PrivateKey: seed,
		CreatedAt:  createdAt,
	}
	newPlaintext, _ := json.Marshal(newWallet)

	// Re-encrypt with same key and same nonce (user requested same salt/nonce; only cipherText changes)
	newCiphertext := aesGCM.Seal(nil, nonce, newPlaintext, nil)
	fmt.Print(base64.StdEncoding.EncodeToString(newCiphertext))
}
