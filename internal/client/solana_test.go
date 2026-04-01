package client

import (
	"os"
	"testing"

	"github.com/AlexZinkM/local-wallet/internal/config"
	"github.com/gagliardetto/solana-go"
)

func TestMain(m *testing.M) {
	os.Setenv("SOLANA_FILE_PATH", "/tmp/test-wallet.cwt")
	if err := config.Init(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestNewSolanaClient_ValidAddress(t *testing.T) {
	addr := "11111111111111111111111111111111"
	client, err := NewSolanaClient(addr)
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.ownerPubkey.String() != addr {
		t.Errorf("owner mismatch: got %s", client.ownerPubkey.String())
	}
}

func TestNewSolanaClient_InvalidAddress(t *testing.T) {
	_, err := NewSolanaClient("invalid")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
}

func TestGetBalance(t *testing.T) {
	t.Skip("integration: requires network")
	client, err := NewSolanaClient("11111111111111111111111111111111")
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	usdc, sol, err := client.GetBalance()
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	_ = usdc
	_ = sol
}

func TestGetTransactions(t *testing.T) {
	t.Skip("integration: requires network")
	client, err := NewSolanaClient("11111111111111111111111111111111")
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	txs, err := client.GetTransactions()
	if err != nil {
		t.Fatalf("GetTransactions: %v", err)
	}
	_ = txs
}

func TestCreateUSDCTransaction_InvalidToAddress(t *testing.T) {
	client, err := NewSolanaClient("11111111111111111111111111111111")
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	key := make([]byte, 64)
	_, err = client.CreateUSDCTransaction("invalid", key, "1.0")
	if err == nil {
		t.Fatal("expected error for invalid to address")
	}
}

func TestCreateUSDCTransaction_InvalidKeyLength(t *testing.T) {
	client, err := NewSolanaClient("11111111111111111111111111111111")
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	key := make([]byte, 32)
	_, err = client.CreateUSDCTransaction("11111111111111111111111111111111", key, "1.0")
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
}

func TestCreateUSDCTransaction_KeyMismatch(t *testing.T) {
	walletA := solana.NewWallet()
	walletB := solana.NewWallet()
	client, err := NewSolanaClient(walletB.PublicKey().String())
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	_, err = client.CreateUSDCTransaction(walletA.PublicKey().String(), walletA.PrivateKey, "0.001")
	if err == nil {
		t.Fatal("expected error for key/address mismatch")
	}
}

func TestCreateSOLTransaction_InvalidToAddress(t *testing.T) {
	client, err := NewSolanaClient("11111111111111111111111111111111")
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	key := make([]byte, 64)
	_, err = client.CreateSOLTransaction("invalid", key, "0.001")
	if err == nil {
		t.Fatal("expected error for invalid to address")
	}
}

func TestCreateSOLTransaction_InvalidKeyLength(t *testing.T) {
	client, err := NewSolanaClient("11111111111111111111111111111111")
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	key := make([]byte, 32)
	_, err = client.CreateSOLTransaction("11111111111111111111111111111111", key, "0.001")
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
}

func TestCreateSOLTransaction_KeyMismatch(t *testing.T) {
	walletA := solana.NewWallet()
	walletB := solana.NewWallet()
	client, err := NewSolanaClient(walletB.PublicKey().String())
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	_, err = client.CreateSOLTransaction(walletA.PublicKey().String(), walletA.PrivateKey, "0.001")
	if err == nil {
		t.Fatal("expected error for key/address mismatch")
	}
}
func TestGetATANotFoundError(t *testing.T) {
	t.Skip("integration: getATANotFoundError calls getTokenAccountRentExempt which requires RPC")
	client, err := NewSolanaClient("11111111111111111111111111111111")
	if err != nil {
		t.Fatalf("NewSolanaClient: %v", err)
	}
	err = client.getATANotFoundError()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
