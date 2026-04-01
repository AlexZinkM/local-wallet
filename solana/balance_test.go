package solana

import (
	"testing"
)

func TestGetBalance_FileNotFound(t *testing.T) {
	_, err := GetBalance("/nonexistent/path/wallet.cwt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
