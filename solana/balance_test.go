package solana

import (
	"context"
	"testing"
)

func TestGetBalance_FileNotFound(t *testing.T) {
	_, err := GetBalance(context.Background(), "/nonexistent/path/wallet.cwt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
