package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"cwt/internal/config"
	"cwt/internal/model"
	"cwt/solana"
)

// SolanaHandler holds configuration for Solana operations
type SolanaHandler struct {
	filePath        string
	cooldownMinutes int
}

// NewSolanaHandler creates a new SolanaHandler with config values
func NewSolanaHandler() (*SolanaHandler, error) {
	filePath := config.GetSolanaFilePath()
	if filePath == "" {
		return nil, errors.New("SOLANA_FILE_PATH not set")
	}

	return &SolanaHandler{
		filePath:        filePath,
		cooldownMinutes: config.GetPayCooldown(),
	}, nil
}

// Generate handles POST /solana/generate
// @Summary      Generate new wallet
// @Description  Generates a new Solana wallet and saves it to .cwt or .txt file
// @Tags         solana
// @Accept       json
// @Produce      json
// @Success      200  {object}  model.GenerateResponse
// @Router       /solana/generate [post]
func (h *SolanaHandler) Generate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. should be POST", http.StatusMethodNotAllowed)
		return
	}

	// Get password as []byte, use it, then zero it immediately
	passwordBytes, err := config.GetSolanaPasswordBytes()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer clear(passwordBytes) // Always clear password from memory

	address, err := solana.GenerateWallet(h.filePath, passwordBytes)
	if err != nil {
		if solana.IsFileExistsError(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.GenerateResponse{
		Success: true,
		Message: "Wallet generated successfully",
		Address: address,
	})
}

// GetBalance handles GET /solana/balance
// @Summary      Get wallet balance (RUB = USDC * rate)
// @Description  Gets USDC and SOL wallet balance with USDC/RUB rate
// @Tags         solana
// @Produce      json
// @Success      200  {object}  model.SolanaBalanceResponse
// @Router       /solana/balance [get]
func (h *SolanaHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed. Should be GET", http.StatusMethodNotAllowed)
		return
	}

	balance, err := solana.GetBalance(h.filePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(balance)
}

// PayUSDC handles POST /solana/pay/usdc
// @Summary      Send USDC
// @Description  Sends a USDC transaction to the specified address
// @Tags         solana
// @Accept       json
// @Produce      json
// @Param        request  body      model.PayRequest  true  "Payment data"
// @Success      200      {object}  model.PayResponse
// @Router       /solana/pay/usdc [post]
func (h *SolanaHandler) PayUSDC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Should be POST", http.StatusMethodNotAllowed)
		return
	}

	var req model.PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Get password as []byte, use it, then zero it immediately
	passwordBytes, err := config.GetSolanaPasswordBytes()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer clear(passwordBytes) // Always clear password from memory

	payResp, err := solana.PayUSDC(h.filePath, passwordBytes, req.ToAddress, req.Amount, h.cooldownMinutes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payResp)
}

// PaySOL handles POST /solana/pay/sol
// @Summary      Send SOL
// @Description  Sends a SOL transaction to the specified address
// @Tags         solana
// @Accept       json
// @Produce      json
// @Param        request  body      model.PayRequest  true  "Payment data"
// @Success      200      {object}  model.PayResponse
// @Router       /solana/pay/sol [post]
func (h *SolanaHandler) PaySOL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Get password as []byte, use it, then zero it immediately
	passwordBytes, err := config.GetSolanaPasswordBytes()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer clear(passwordBytes) // Always clear password from memory

	payResp, err := solana.PaySOL(h.filePath, passwordBytes, req.ToAddress, req.Amount, h.cooldownMinutes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payResp)
}

// History handles GET /solana/history/usdc
// @Summary      Get wallet transactions
// @Description  Gets list of wallet transactions with filtering capability (USDC and SOL)
// @Tags         solana
// @Produce      json
// @Param        type       query     string   false  "Transaction type: DEBIT or CREDIT"
// @Param        txId       query     string   false  "Transaction ID"
// @Param        from       query     string   false  "Start date (YYYY-MM-DD)"
// @Param        to         query     string   false  "End date (YYYY-MM-DD)"
// @Param        minAmount  query     string   false  "Minimum amount"
// @Param        maxAmount  query     string   false  "Maximum amount"
// @Param        currency   query     string   false  "Filter by currency: USDC or SOL"
// @Success      200  {object}  model.LogResponse
// @Router       /solana/transactions [get]
func (h *SolanaHandler) TransactionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed. Should be GET", http.StatusMethodNotAllowed)
		return
	}

	var req model.LogRequest

	// Parse date parameters (YYYY-MM-DD)
	const dateLayout = "2006-01-02"
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := time.Parse(dateLayout, fromStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid from date: use YYYY-MM-DD (e.g. 2006-01-02)"})
			return
		}
		req.From = &t
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := time.Parse(dateLayout, toStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid to date: use YYYY-MM-DD (e.g. 2006-01-02)"})
			return
		}
		// End of day so filter is inclusive
		t = t.Add(24*time.Hour - time.Nanosecond)
		req.To = &t
	}

	// Parse transaction type
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		txType := model.TransactionType(typeStr)
		req.Type = &txType
	}

	// Parse txId
	if txID := r.URL.Query().Get("txId"); txID != "" {
		req.TxID = &txID
	}

	// Parse amounts
	if minAmount := r.URL.Query().Get("minAmount"); minAmount != "" {
		req.MinAmount = &minAmount
	}
	if maxAmount := r.URL.Query().Get("maxAmount"); maxAmount != "" {
		req.MaxAmount = &maxAmount
	}

	// Parse currency
	if currency := r.URL.Query().Get("currency"); currency != "" {
		req.Currency = &currency
	}

	// Validate
	if err := req.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	logResp, err := solana.GetTransactions(h.filePath, &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(logResp)
}
