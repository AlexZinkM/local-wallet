package api

import (
	"net/http"

	"cwt/internal/handler"

	httpSwagger "github.com/swaggo/http-swagger"
)

// SetupRouter sets up router with handlers
func SetupRouter() (http.Handler, error) {
	solanaHandler, err := handler.NewSolanaHandler()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	// Swagger UI
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Solana endpoints
	mux.HandleFunc("/solana/generate", solanaHandler.Generate)
	mux.HandleFunc("/solana/balance", solanaHandler.GetBalance)
	mux.HandleFunc("/solana/transactions", solanaHandler.TransactionHistory)
	mux.HandleFunc("/solana/pay/usdc", solanaHandler.PayUSDC)
	mux.HandleFunc("/solana/pay/sol", solanaHandler.PaySOL)

	return mux, nil
}
