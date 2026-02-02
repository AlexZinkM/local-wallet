package model

// SolanaBalanceResponse represents response for GET /solana/balance
type SolanaBalanceResponse struct {
	Address string `json:"address"`
	USDC    string `json:"usdc"`
	SOL     string `json:"sol"`
	Rate    string `json:"rate"`
	RUB     string `json:"usdc_amount_in_rub"`
}
