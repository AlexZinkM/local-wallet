package model

// PayRequest represents request for POST pay/...
type PayRequest struct {
	ToAddress string `json:"toAddress" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
}

// PayResponse represents response for POST pay/...
type PayResponse struct {
	TxID string `json:"txId"`
}
