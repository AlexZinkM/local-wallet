package solana

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/AlexZinkM/local-wallet/internal/client"
	"github.com/AlexZinkM/local-wallet/internal/common"
	"github.com/AlexZinkM/local-wallet/internal/crypto"
	"github.com/AlexZinkM/local-wallet/internal/model"
)

// GetTransactions gets wallet transactions with filtering
func GetTransactions(ctx context.Context, filePath string, req *model.LogRequest) (*model.LogResponse, error) {
	// Read address from file
	address, err := crypto.ReadWalletAddress(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet address: %w", err)
	}

	// Create client
	solanaClient, err := client.NewSolanaClient(address)
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	// Get all transactions
	solanaTxs, err := solanaClient.GetTransactions(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to model format
	resultTransactions := make([]model.Transaction, 0, len(solanaTxs))
	for _, tx := range solanaTxs {
		// Filter by type
		if req.Type != nil {
			if string(*req.Type) != tx.Type {
				continue
			}
		}

		// Filter by txId
		if req.TxID != nil && *req.TxID != tx.TxID {
			continue
		}

		// Filter by currency
		if req.Currency != nil && *req.Currency != tx.Currency {
			continue
		}

		// Filter by dates
		if req.From != nil && tx.Timestamp.Before(*req.From) {
			continue
		}
		if req.To != nil && tx.Timestamp.After(*req.To) {
			continue
		}

		// Filter by amount (using integer comparison to avoid float precision issues)
		if req.MinAmount != nil {
			decimals := common.USDCDecimals
			if tx.Currency == "SOL" {
				decimals = common.SOLDecimals
			}
			cmp, err := common.CompareAmounts(tx.Amount, *req.MinAmount, decimals)
			if err != nil {
				return nil, fmt.Errorf("failed to compare min amount: %w", err)
			}
			if cmp < 0 {
				continue
			}
		}
		if req.MaxAmount != nil {
			decimals := common.USDCDecimals
			if tx.Currency == "SOL" {
				decimals = common.SOLDecimals
			}
			cmp, err := common.CompareAmounts(tx.Amount, *req.MaxAmount, decimals)
			if err != nil {
				return nil, fmt.Errorf("failed to compare max amount: %w", err)
			}
			if cmp > 0 {
				continue
			}
		}

		resultTransactions = append(resultTransactions, model.Transaction{
			Type:        model.TransactionType(tx.Type),
			TxID:        tx.TxID,
			From:        tx.From,
			To:          tx.To,
			Amount:      tx.Amount,
			Currency:    tx.Currency,
			OurFeeSOL:   tx.OurFeeSOL,
			Timestamp:   tx.Timestamp,
			BlockNumber: tx.BlockNumber,
			Status:      tx.Status,
		})
	}

	// Sort by time DESC (newest first)
	sort.Slice(resultTransactions, func(i, j int) bool {
		return resultTransactions[i].Timestamp.After(resultTransactions[j].Timestamp)
	})

	// Calculate filtered_total_*_USDC using exact integer arithmetic (micro USDC units).
	var totalIncomeMicro, totalSpentMicro uint64
	for _, tx := range resultTransactions {
		if tx.Currency != "USDC" || tx.Status != "success" {
			continue
		}
		micro, err := common.USDCToMicro(tx.Amount)
		if err != nil {
			log.Printf("failed to parse USDC amount for tx %s: %v", tx.TxID, err)
			continue
		}
		switch tx.Type {
		case model.TransactionTypeDebit:
			totalIncomeMicro += micro
		case model.TransactionTypeCredit:
			totalSpentMicro += micro
		}
	}

	return &model.LogResponse{
		Address:                 address,
		FilteredTotalIncomeUSDC: common.MicroToUSDC(totalIncomeMicro),
		FilteredTotalSpentUSDC:  common.MicroToUSDC(totalSpentMicro),
		Transactions:            resultTransactions,
	}, nil
}
