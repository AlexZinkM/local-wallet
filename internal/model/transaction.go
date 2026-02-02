package model

import (
	"fmt"
	"time"

	"cwt/internal/common"
)

// TransactionType transaction type
type TransactionType string

const (
	TransactionTypeDebit  TransactionType = "DEBIT"
	TransactionTypeCredit TransactionType = "CREDIT"
)

// Transaction represents a transaction
type Transaction struct {
	Type        TransactionType `json:"type"`
	TxID        string          `json:"txId"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	Amount      string          `json:"amount"`
	Currency    string          `json:"currency"`  // "USDC" or "SOL"
	OurFeeSOL   string          `json:"ourFeeSOL"` // SOL we paid as fee
	Timestamp   time.Time       `json:"timestamp"`
	BlockNumber int64           `json:"blockNumber"`
	Status      string          `json:"status"`
}

// LogResponse represents response for GET log/...
type LogResponse struct {
	Address         string        `json:"address"`
	TotalIncomeUSDC string        `json:"total_income_USDC"` // USDC only
	TotalSpentUSDC  string        `json:"total_spent_USDC"`  // USDC only
	Transactions    []Transaction `json:"transactions"`
}

// LogRequest represents request parameters for GET log/...
type LogRequest struct {
	Type      *TransactionType `form:"type"`
	TxID      *string          `form:"txId"`
	From      *time.Time       `form:"from"`
	To        *time.Time       `form:"to"`
	MinAmount *string          `form:"minAmount"`
	MaxAmount *string          `form:"maxAmount"`
	Currency  *string          `form:"currency"` // "USDC" or "SOL"
}

// Validate validates LogRequest filter parameters.
func (r *LogRequest) Validate() error {
	if r.Type != nil && *r.Type != TransactionTypeDebit && *r.Type != TransactionTypeCredit {
		return fmt.Errorf("type must be DEBIT or CREDIT")
	}
	if r.Currency != nil && *r.Currency != "USDC" && *r.Currency != "SOL" {
		return fmt.Errorf("currency must be USDC or SOL")
	}
	if r.From != nil && r.To != nil && r.To.Before(*r.From) {
		return fmt.Errorf("to date must be after or equal to from date")
	}
	if r.MinAmount != nil && r.MaxAmount != nil {
		cmp, err := common.CompareUSDCAmounts(*r.MinAmount, *r.MaxAmount)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		if cmp == 1 {
			return fmt.Errorf("minAmount must be less than or equal to maxAmount")
		}
	}
	return nil
}
