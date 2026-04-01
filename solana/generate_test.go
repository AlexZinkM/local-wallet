package solana

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateWallet_InvalidExtension(t *testing.T) {
	_, err := GenerateWallet("/tmp/wallet.txt", []byte("password"))
	if err == nil {
		t.Fatal("expected error for .txt extension")
	}
	if err.Error() != "file must have .cwt extension" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateWallet_FileExists(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "wallet.cwt")
	if err := os.WriteFile(path, []byte("not empty"), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := GenerateWallet(path, []byte("password"))
	if err == nil {
		t.Fatal("expected error when file exists and not empty")
	}
	if !IsFileExistsError(err) {
		t.Errorf("expected FileExistsError, got %T: %v", err, err)
	}
}

func TestGenerateWallet_Success(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "wallet.cwt")
	addr, err := GenerateWallet(path, []byte("password"))
	if err != nil {
		t.Fatalf("GenerateWallet: %v", err)
	}
	if addr == "" {
		t.Fatal("expected non-empty address")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected .cwt file to be created")
	}
}

func TestIsFileExistsError(t *testing.T) {
	if IsFileExistsError(nil) {
		t.Error("nil should not be FileExistsError")
	}
	if !IsFileExistsError(&FileExistsError{Message: "test"}) {
		t.Error("FileExistsError should match")
	}
	if IsFileExistsError(errOther{}) {
		t.Error("other error should not match")
	}
}

type errOther struct{}

func (errOther) Error() string { return "other" }
