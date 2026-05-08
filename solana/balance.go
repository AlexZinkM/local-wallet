package solana

import (
	"context"
	"fmt"
	"strconv"

	"github.com/AlexZinkM/local-wallet/internal/client"
	"github.com/AlexZinkM/local-wallet/internal/common"
	"github.com/AlexZinkM/local-wallet/internal/crypto"
	"github.com/AlexZinkM/local-wallet/internal/model"
)

// GetBalance gets wallet balance
func GetBalance(ctx context.Context, filePath string) (*model.SolanaBalanceResponse, error) {
	// Read address from file
	address, err := crypto.ReadWalletAddress(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet address: %w", err)
	}

	// Create clients
	solanaClient, err := client.NewSolanaClient(address)
	if err != nil {
		return nil, err
	}
	coingeckoClient := client.NewCoinGeckoClient()

	// Get USDC (micro) and SOL (lamports) balance
	usdcMicro, solLamports, err := solanaClient.GetBalance(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to display strings (no float precision loss)
	usdc := common.MicroToUSDC(usdcMicro)
	sol := common.LamportsToSOL(solLamports)

	response := &model.SolanaBalanceResponse{
		Address: address,
		USDC:    usdc,
		SOL:     sol,
	}

	// CoinGecko is optional: if rate fetch fails, still return balance without rate fields.
	rate, err := coingeckoClient.GetUSDCtoRUBrate()
	if err != nil {
		return response, nil
	}

	// Calculate RUB (use float only for display, not for critical operations).
	usdcFloat, _ := strconv.ParseFloat(usdc, 64)
	rateFloat, _ := strconv.ParseFloat(rate, 64)
	rub := fmt.Sprintf("%.2f", usdcFloat*rateFloat)
	response.Rate = rate
	response.RUB = rub

	return response, nil
}
