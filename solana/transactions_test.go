package solana

import (
	"os"
	"testing"

	"github.com/AlexZinkM/local-wallet/internal/config"
	"github.com/AlexZinkM/local-wallet/internal/model"
)

func init() {
	os.Setenv("SOLANA_FILE_PATH", "/tmp/test-wallet.cwt")
	if err := config.Init(); err != nil {
		panic(err)
	}
}

func TestGetTransactions_FileNotFound(t *testing.T) {
	_, err := GetTransactions("/nonexistent/path/wallet.cwt", &model.LogRequest{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
