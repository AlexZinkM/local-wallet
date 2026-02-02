package solana

import (
	"fmt"
	"strconv"

	"github.com/AlexZinkM/local-wallet/internal/client"
	"github.com/AlexZinkM/local-wallet/internal/common"
	"github.com/AlexZinkM/local-wallet/internal/crypto"
	"github.com/AlexZinkM/local-wallet/internal/model"
)

// GetBalance gets wallet balance
func GetBalance(filePath string) (*model.SolanaBalanceResponse, error) {
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
	usdcMicro, solLamports, err := solanaClient.GetBalance()
	if err != nil {
		return nil, err
	}

	// Convert to display strings (no float precision loss)
	usdc := common.MicroToUSDC(usdcMicro)
	sol := common.LamportsToSOL(solLamports)

	// Get USDC/RUB rate
	rate, err := coingeckoClient.GetUSDCtoRUBrate()
	if err != nil {
		return nil, fmt.Errorf("failed to get rate: %w", err)
	}

	// Calculate RUB (use float only for display, not for critical operations)
	usdcFloat, _ := strconv.ParseFloat(usdc, 64)
	rateFloat, _ := strconv.ParseFloat(rate, 64)
	rub := fmt.Sprintf("%.2f", usdcFloat*rateFloat)

	return &model.SolanaBalanceResponse{
		Address: address,
		USDC:    usdc,
		SOL:     sol,
		Rate:    rate,
		RUB:     rub,
	}, nil
}
