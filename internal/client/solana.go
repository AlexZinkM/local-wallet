package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cwt/internal/common"
	"cwt/internal/config"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	usdcMintAddressMainnet = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v" // USDC mint address on Solana mainnet (does not work on devnet/testnet)
	usdcDecimals           = 6                                              // USDC always has 6 decimals
)

// SolanaClient is a client for working with Solana RPC
type SolanaClient struct {
	rpcClient     *rpc.Client
	rpcURL        string
	mintPublicKey solana.PublicKey
	ownerPubkey   solana.PublicKey // address passed to NewSolanaClient
}

// NewSolanaClient creates a new Solana client for the given address.
func NewSolanaClient(address string) (*SolanaClient, error) {
	ownerPubkey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana address: %w", err)
	}

	rpcURL := config.GetSolanaRPCURL()
	mintPubKey, err := solana.PublicKeyFromBase58(usdcMintAddressMainnet)
	if err != nil {
		return nil, fmt.Errorf("invalid USDC mint address: %w", err)
	}

	return &SolanaClient{
		rpcClient:     rpc.New(rpcURL),
		rpcURL:        rpcURL,
		mintPublicKey: mintPubKey,
		ownerPubkey:   ownerPubkey,
	}, nil
}

// GetBalance gets USDC (micro units) and SOL (lamports) balance for the client's address
func (c *SolanaClient) GetBalance() (usdcMicro uint64, solLamports uint64, err error) {
	solLamports, err = c.getSOLBalanceLamports()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get SOL balance: %w", err)
	}

	usdcMicro, err = c.getUSDCBalanceMicro()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get USDC balance: %w", err)
	}

	return usdcMicro, solLamports, nil
}

// getSOLBalanceLamports gets SOL balance in lamports
func (c *SolanaClient) getSOLBalanceLamports() (uint64, error) {
	balance, err := c.rpcClient.GetBalance(
		context.Background(),
		c.ownerPubkey,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get SOL balance: %w", err)
	}
	return balance.Value, nil
}

// getUSDCBalanceMicro gets USDC balance in micro units (10^-6 USDC)
func (c *SolanaClient) getUSDCBalanceMicro() (uint64, error) {
	ataAddress, _, err := solana.FindAssociatedTokenAddress(c.ownerPubkey, c.mintPublicKey)
	if err != nil {
		return 0, fmt.Errorf("failed to find associated token account address: %w", err)
	}

	balance, err := c.rpcClient.GetTokenAccountBalance(context.Background(), ataAddress, rpc.CommitmentConfirmed)
	if err != nil {
		if isATANotFoundError(err) {
			return 0, c.getATANotFoundError()
		}
		return 0, fmt.Errorf("failed to get token account balance: %w", err)
	}

	if balance.Value == nil {
		return 0, nil
	}

	amount, err := strconv.ParseUint(balance.Value.Amount, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse USDC balance amount: %w", err)
	}

	return amount, nil
}

// getTokenAccountRentExempt gets the minimum balance required for rent exemption of a token account
func (c *SolanaClient) getTokenAccountRentExempt() (string, error) {
	// Token account size is 165 bytes
	const tokenAccountSize = 165

	rentExempt, err := c.rpcClient.GetMinimumBalanceForRentExemption(
		context.Background(),
		tokenAccountSize,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return "", err
	}

	// Convert lamports to SOL
	return common.LamportsToSOL(rentExempt), nil
}

// TokenAccountInfo represents token account info from RPC
type TokenAccountInfo struct {
	Pubkey  string `json:"pubkey"`
	Account struct {
		Data struct {
			Parsed struct {
				Info struct {
					Mint        string `json:"mint,omitempty"`
					Owner       string `json:"owner,omitempty"`
					TokenAmount struct {
						Amount         string `json:"amount,omitempty"`
						Decimals       int    `json:"decimals,omitempty"`
						UiAmountString string `json:"uiAmountString,omitempty"`
					} `json:"tokenAmount,omitempty"`
				} `json:"info,omitempty"`
				Type string `json:"type,omitempty"`
			} `json:"parsed,omitempty"`
		} `json:"data"`
	} `json:"account"`
}

// GetTransactions gets transactions for the client's address (USDC SPL token only)
func (c *SolanaClient) GetTransactions() ([]SolanaTransaction, error) {
	// Get ATA address
	ataAddress, _, err := solana.FindAssociatedTokenAddress(c.ownerPubkey, c.mintPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find associated token account address: %w", err)
	}

	// Check if ATA exists by trying to get balance
	_, err = c.rpcClient.GetTokenAccountBalance(context.Background(), ataAddress, rpc.CommitmentConfirmed)
	if err != nil {
		if isATANotFoundError(err) {
			// If account doesn't exist, return empty list
			return []SolanaTransaction{}, nil
		}
		return nil, fmt.Errorf("failed to check token account: %w", err)
	}

	// Collect all signatures from both main address and ATA
	signatureSet := make(map[string]bool)
	limit := 100

	// Get signatures for main address
	sigs, err := c.rpcClient.GetSignaturesForAddressWithOpts(
		context.Background(),
		c.ownerPubkey,
		&rpc.GetSignaturesForAddressOpts{
			Limit: &limit,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, sig := range sigs {
		signatureSet[sig.Signature.String()] = true
	}

	// Get signatures for ATA
	tokenAccountSigs, err := c.rpcClient.GetSignaturesForAddressWithOpts(
		context.Background(),
		ataAddress,
		&rpc.GetSignaturesForAddressOpts{
			Limit: &limit,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, sig := range tokenAccountSigs {
		signatureSet[sig.Signature.String()] = true
	}

	// Filter and parse transactions
	transactions := make([]SolanaTransaction, 0, 8)

	for sigStr := range signatureSet {
		sig, err := solana.SignatureFromBase58(sigStr)
		if err != nil {
			return nil, err
		}

		// Get transaction details (support versioned transactions)
		// maxVersion is hardcoded - no point making it env var because
		// new version support requires library update and rebuild anyway
		maxVersion := uint64(0)
		tx, err := c.rpcClient.GetTransaction(
			context.Background(),
			sig,
			&rpc.GetTransactionOpts{
				Encoding:                       solana.EncodingBase64,
				MaxSupportedTransactionVersion: &maxVersion,
			},
		)
		if err != nil {
			return nil, err
		}

		// Parse transaction for both USDC and SOL transfers
		txList, err := c.parseTransaction(tx, sig)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, txList...)
	}

	return transactions, nil
}

// parseTransaction parses transaction and extracts USDC or SOL transfer data
// Logic: If USDC movement exists, any SOL change is fee. Otherwise, SOL change is a transfer.
func (c *SolanaClient) parseTransaction(tx *rpc.GetTransactionResult, signature solana.Signature) ([]SolanaTransaction, error) {
	ownerPubkeyStr := c.ownerPubkey.String()

	// Get common transaction metadata
	timestamp := time.Now()
	if tx.BlockTime != nil {
		timestamp = time.Unix(int64(*tx.BlockTime), 0)
	}

	status := "success"
	if tx.Meta != nil && tx.Meta.Err != nil {
		status = "failed"
	}

	// --- Calculate owner's SOL delta (needed for both USDC fee and SOL transfers) ---
	var ownerSOLDelta int64
	decodedTx, err := tx.Transaction.GetTransaction()
	if err == nil && tx.Meta != nil {
		accountKeys := decodedTx.Message.AccountKeys
		for i, key := range accountKeys {
			if key.Equals(c.ownerPubkey) {
				preBal := tx.Meta.PreBalances[i]
				postBal := tx.Meta.PostBalances[i]
				if postBal >= preBal {
					ownerSOLDelta = int64(postBal - preBal)
				} else {
					ownerSOLDelta = -int64(preBal - postBal)
				}
				break
			}
		}
	}

	// --- Parse USDC transfers ---
	usdcDeltas := make(map[string]int64)

	if tx.Meta != nil && tx.Meta.PreTokenBalances != nil {
		for _, pre := range tx.Meta.PreTokenBalances {
			if pre.Mint.Equals(c.mintPublicKey) && pre.Owner != nil {
				amt, _ := strconv.ParseUint(pre.UiTokenAmount.Amount, 10, 64)
				usdcDeltas[pre.Owner.String()] -= int64(amt)
			}
		}
	}

	if tx.Meta != nil && tx.Meta.PostTokenBalances != nil {
		for _, post := range tx.Meta.PostTokenBalances {
			if post.Mint.Equals(c.mintPublicKey) && post.Owner != nil {
				amt, _ := strconv.ParseUint(post.UiTokenAmount.Amount, 10, 64)
				usdcDeltas[post.Owner.String()] += int64(amt)
			}
		}
	}

	// Check if owner has USDC balance change
	ourUSDCDelta := usdcDeltas[ownerPubkeyStr]

	if ourUSDCDelta != 0 {
		// USDC transaction - any SOL change is the fee
		var from, to string
		var amount uint64
		var txType string

		if ourUSDCDelta > 0 {
			txType = "DEBIT"
			amount = uint64(ourUSDCDelta)
			to = ownerPubkeyStr
			for owner, delta := range usdcDeltas {
				if delta < 0 {
					from = owner
					break
				}
			}
		} else {
			txType = "CREDIT"
			amount = uint64(-ourUSDCDelta)
			from = ownerPubkeyStr
			for owner, delta := range usdcDeltas {
				if delta > 0 {
					to = owner
					break
				}
			}
		}

		// Fee = total SOL cost we paid (CREDIT only; DEBIT shows "0")
		feeStr := "0"
		if txType == "CREDIT" && ownerSOLDelta < 0 {
			feeStr = common.LamportsToSOL(uint64(-ownerSOLDelta))
		}

		return []SolanaTransaction{{
			Type:        txType,
			TxID:        signature.String(),
			From:        from,
			To:          to,
			Amount:      common.MicroToUSDC(amount),
			Currency:    "USDC",
			OurFeeSOL:   feeStr,
			Timestamp:   timestamp,
			BlockNumber: int64(tx.Slot),
			Status:      status,
		}}, nil
	}

	// --- No USDC movement - check for SOL transfer ---
	if ownerSOLDelta == 0 || decodedTx == nil || tx.Meta == nil {
		return nil, nil
	}

	// For pure SOL transactions, separate fee from transfer amount
	// Fee payer is typically index 0
	accountKeys := decodedTx.Message.AccountKeys
	ownerIndex := -1
	for i, key := range accountKeys {
		if key.Equals(c.ownerPubkey) {
			ownerIndex = i
			break
		}
	}

	isFeePayer := ownerIndex == 0
	actualSOLDelta := ownerSOLDelta
	if isFeePayer {
		actualSOLDelta = ownerSOLDelta + int64(tx.Meta.Fee)
	}

	// Only show SOL transaction if there's an actual transfer (not just fee)
	if actualSOLDelta == 0 {
		return nil, nil
	}

	var from, to string
	var amount uint64
	var txType string

	if actualSOLDelta > 0 {
		// Received SOL
		txType = "DEBIT"
		amount = uint64(actualSOLDelta)
		to = ownerPubkeyStr
		// Find sender
		for i, key := range accountKeys {
			pre := tx.Meta.PreBalances[i]
			post := tx.Meta.PostBalances[i]
			if pre > post && !key.Equals(c.ownerPubkey) {
				from = key.String()
				break
			}

		}
	} else {
		// Sent SOL
		txType = "CREDIT"
		amount = uint64(-actualSOLDelta)
		from = ownerPubkeyStr
		// Find receiver
		for i, key := range accountKeys {
				pre := tx.Meta.PreBalances[i]
				post := tx.Meta.PostBalances[i]
				if post > pre && !key.Equals(c.ownerPubkey) {
					to = key.String()
					break
				}
		}
	}

	// Fee = SOL we paid (only for CREDIT when we're fee payer; DEBIT shows "0")
	feeStr := "0"
	if txType == "CREDIT" && isFeePayer && tx.Meta != nil {
		feeStr = common.LamportsToSOL(tx.Meta.Fee)
	}

	return []SolanaTransaction{{
		Type:        txType,
		TxID:        signature.String(),
		From:        from,
		To:          to,
		Amount:      common.LamportsToSOL(amount),
		Currency:    "SOL",
		OurFeeSOL:   feeStr,
		Timestamp:   timestamp,
		BlockNumber: int64(tx.Slot),
		Status:      status,
	}}, nil
}

// CreateUSDCTransaction creates and signs a USDC transfer transaction
// privateKeyBytes must be full 64-byte Solana private key (caller should zero it after use)
func (c *SolanaClient) CreateUSDCTransaction(toAddress string, privateKeyBytes []byte, amount string) (string, error) {

	toPubkey, err := solana.PublicKeyFromBase58(toAddress)
	if err != nil {
		return "", fmt.Errorf("invalid to address: %w", err)
	}

	// Validate private key (full 64-byte key)
	if len(privateKeyBytes) != 64 {
		return "", fmt.Errorf("invalid private key length: expected 64 bytes")
	}

	// Use full private key directly
	wallet := solana.PrivateKey(privateKeyBytes)

	// Verify wallet matches from address
	if !wallet.PublicKey().Equals(c.ownerPubkey) {
		return "", fmt.Errorf("private key does not match our address")
	}

	// Convert to token amount
	amountUint64, err := common.USDCToMicro(amount)
	if err != nil {
		return "", err
	}

	// Get latest blockhash (GetRecentBlockhash is deprecated, use GetLatestBlockhash)
	recent, err := c.rpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Get source ATA address
	sourceTokenAccount, _, err := solana.FindAssociatedTokenAddress(c.ownerPubkey, c.mintPublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to find source token account address: %w", err)
	}

	// Check if source ATA exists by trying to get balance
	_, err = c.rpcClient.GetTokenAccountBalance(context.Background(), sourceTokenAccount, rpc.CommitmentConfirmed)
	if err != nil {
		if isATANotFoundError(err) {
			return "", c.getATANotFoundError()
		}
		return "", fmt.Errorf("failed to check source token account: %w", err)
	}

	// Get or create destination token account
	destTokenAccount, _, err := solana.FindAssociatedTokenAddress(toPubkey, c.mintPublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to find destination token account: %w", err)
	}

	// Check if destination account exists, if not create it
	destAccountInfo, err := c.rpcClient.GetAccountInfo(context.Background(), destTokenAccount)
	if err != nil && !isATANotFoundError(err) {
		return "", fmt.Errorf("failed to get destination account info: %w", err)
	}

	needCreateATA := isATANotFoundError(err) || destAccountInfo.Value == nil
	if needCreateATA {
		// Create associated token account instruction
		createATAInstruction := associatedtokenaccount.NewCreateInstruction(
			c.ownerPubkey,   // payer
			toPubkey,        // owner
			c.mintPublicKey, // mint
		).Build()

		// Create transfer instruction
		transferInstruction := token.NewTransferCheckedInstruction(
			amountUint64,
			usdcDecimals,
			sourceTokenAccount,
			c.mintPublicKey,
			destTokenAccount,
			c.ownerPubkey,
			[]solana.PublicKey{},
		).Build()

		// Create transaction with both instructions
		tx, err := solana.NewTransaction(
			[]solana.Instruction{createATAInstruction, transferInstruction},
			recent.Value.Blockhash,
			solana.TransactionPayer(c.ownerPubkey),
		)
		if err != nil {
			return "", fmt.Errorf("failed to create transaction: %w", err)
		}

		// Sign transaction
		_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
			if wallet.PublicKey().Equals(key) {
				return &wallet
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed to sign transaction: %w", err)
		}

		// Send transaction
		sig, err := c.rpcClient.SendTransactionWithOpts(
			context.Background(),
			tx,
			rpc.TransactionOpts{
				SkipPreflight:       false, // Transaction validation befor node
				PreflightCommitment: rpc.CommitmentFinalized,
			},
		)
		if err != nil {
			return "", fmt.Errorf("failed to send transaction: %w", err)
		}

		return sig.String(), nil
	}

	// Destination account exists, just transfer
	transferInstruction := token.NewTransferCheckedInstruction(
		amountUint64,
		usdcDecimals,
		sourceTokenAccount,
		c.mintPublicKey,
		destTokenAccount,
		c.ownerPubkey,
		[]solana.PublicKey{},
	).Build()

	// Create transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{transferInstruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(c.ownerPubkey),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if wallet.PublicKey().Equals(key) {
			return &wallet
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.rpcClient.SendTransactionWithOpts(
		context.Background(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return sig.String(), nil
}

// CreateSOLTransaction creates and signs a SOL transfer transaction
// privateKeyBytes must be full 64-byte Solana private key (caller should zero it after use)
func (c *SolanaClient) CreateSOLTransaction(toAddress string, privateKeyBytes []byte, amount string) (string, error) {

	toPubkey, err := solana.PublicKeyFromBase58(toAddress)
	if err != nil {
		return "", fmt.Errorf("invalid to address: %w", err)
	}

	// Validate private key (full 64-byte key)
	if len(privateKeyBytes) != 64 {
		return "", fmt.Errorf("invalid private key length: expected 64 bytes")
	}

	// Use full private key directly
	wallet := solana.PrivateKey(privateKeyBytes)

	// Verify wallet matches from address
	if !wallet.PublicKey().Equals(c.ownerPubkey) {
		return "", fmt.Errorf("private key does not match our address")
	}

	// Convert SOL to lamports (1 SOL = 1,000,000,000 lamports)
	lamports, err := common.SOLToLamports(amount)
	if err != nil {
		return "", err
	}

	// Get latest blockhash
	recent, err := c.rpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create transfer instruction
	transferInstruction := system.NewTransferInstruction(
		lamports,
		c.ownerPubkey,
		toPubkey,
	).Build()

	// Create transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{transferInstruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(c.ownerPubkey),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if wallet.PublicKey().Equals(key) {
			return &wallet
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.rpcClient.SendTransactionWithOpts(
		context.Background(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return sig.String(), nil
}

// SolanaTransaction represents a Solana transaction
type SolanaTransaction struct {
	Type        string
	TxID        string
	From        string
	To          string
	Amount      string
	Currency    string // "USDC" or "SOL"
	OurFeeSOL   string // SOL we paid as fee
	Timestamp   time.Time
	BlockNumber int64
	Status      string
}

// isATANotFoundError checks if error indicates that token account doesn't exist
func isATANotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "could not find account") ||
		strings.Contains(errStr, "not found")
}

// getATANotFoundError returns formatted error for missing USDC account
func (c *SolanaClient) getATANotFoundError() error {
	rentExempt, err := c.getTokenAccountRentExempt()
	if err != nil {
		return err
	}
	return fmt.Errorf("USDC token account not found for address %s. Please deposit any amount of USDC to this Solana address to create the account (requires rent exempt: %s SOL from the sender)", c.ownerPubkey.String(), rentExempt)
}
