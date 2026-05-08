package solana

import (
	"context"
	"testing"
)

func TestPayUSDC_InvalidAddress(t *testing.T) {
	_, err := PayUSDC(context.Background(), "/tmp/wallet.cwt", []byte("pass"), "invalid", "1.0", "")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
	if err.Error() != "invalid Solana address" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPaySOL_InvalidAddress(t *testing.T) {
	_, err := PaySOL(context.Background(), "/tmp/wallet.cwt", []byte("pass"), "invalid", "0.001", "")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
	if err.Error() != "invalid Solana address" {
		t.Errorf("unexpected error: %v", err)
	}
}
