package model

// CWTFile represents .cwt file structure
type CWTFile struct {
	Network    string `json:"network"`
	Address    string `json:"address"`
	QR         string `json:"QR"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	CipherText string `json:"cipherText"`
}

// WalletData represents decrypted wallet data
type WalletData struct {
	PrivateKey []byte `json:"privateKey"` // 64 bytes seed (stored as base64 in JSON)
	CreatedAt  string `json:"createdAt"`
}

