# Solana Wallet Service V1

Go service for Solana wallet generation, balance viewing, and USDC/SOL transfers. Use it as a **desktop app** (HTTP API + Swagger UI) or as a **library** (import `solana` package).

## Project Priorities

1. **Security** — private keys encrypted with AES-256-GCM; desktop server binds to localhost only
2. **Precision** — all amounts use integers (lamports, micro-USDC), no float
3. **Dual use** — same codebase for desktop app and library

## Project Structure

```
cmd/app/
  └── main.go              # Application entry point

solana/                    # Library package — use these in your code
  ├── generate.go          # GenerateWallet
  ├── balance.go           # GetBalance
  ├── transactions.go      # GetTransactions
  └── pay.go               # PayUSDC, PaySOL
  

internal/
  ├── api/router.go        # Routing + Swagger UI
  ├── client/              # RPC / CoinGecko clients
  ├── config/env.go        # Environment variables
  ├── handler/             # HTTP handlers (call solana package)
  ├── crypto/              # Encryption / .cwt read-write
  └── model/               # DTOs (request/response types)
```

---

## Setup and running (desktop app)

**PowerShell (Windows):**
```powershell
$env:SOLANA_FILE_PATH="C:\path\to\wallet.cwt"
$env:SOLANA_PASSWORD="your-password"
go run cmd/app/main.go
```

**Bash (Linux/macOS):**
```bash
SOLANA_FILE_PATH=/path/to/wallet.cwt SOLANA_PASSWORD=your-password go run cmd/app/main.go
```

**Swagger UI:** after starting the app, open **http://127.0.0.1:8080/swagger/index.html** for full request/response schemas and try-it-out.

### Environment variables (desktop / Swagger)

| Variable               | Required | Description |
|------------------------|----------|-------------|
| `SOLANA_FILE_PATH`     | yes      | Absolute path to .cwt wallet file 
| `SOLANA_PASSWORD`      | yes      | Password for encrypting/decrypting private key 
| `PORT`                 | no       | Server port (default: `8080`) 
| `SOLANA_RPC_URL`       | no       | Solana RPC URL (default: public mainnet) 
| `PAY_COOLDOWN_MINUTES` | no       | Minutes between pay operations (default: `4`) 

---

## HTTP API (desktop app)

Details, request bodies, query params, and response shapes are in **Swagger**. Summary:

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/solana/generate` | Create new wallet, save to .cwt |
| GET | `/solana/balance` | Get SOL + USDC balance and RUB rate |
| GET | `/solana/transactions` | Get transaction history (filters in Swagger) |
| POST | `/solana/pay/usdc` | Send USDC |
| POST | `/solana/pay/sol` | Send SOL |

---

## Library (package `solana`)

Import `cwt/solana` and call these functions. You provide `filePath` (and `password` where needed); the library reads/decrypts the .cwt file and uses `SOLANA_RPC_URL` from the environment when talking to Solana (or default RPC).

### Generate

- **`GenerateWallet(filePath string, password []byte) (address string, err error)`**  
  Creates a new keypair, encrypts it, writes .cwt at `filePath`. Returns the public address. Use `[]byte(yourPassword)` and clear the slice after use.
- **`IsFileExistsError(err error) bool`**  
  Returns true if `err` is because the .cwt file already exists (so you can prompt to choose another path).
- **`FileExistsError`**  
  Error type when the target file already exists.

### Balance

- **`GetBalance(filePath string) (*model.SolanaBalanceResponse, error)`**  
  Reads address from .cwt (no password), fetches SOL and USDC balance and RUB rate. Returns `*model.SolanaBalanceResponse`.

### History

- **`GetTransactions(filePath string, req *model.LogRequest) (*model.LogResponse, error)`**  
  Reads address from .cwt, fetches transaction history with optional filters (type, txId, from, to, minAmount, maxAmount, currency). Request/response types are in `cwt/internal/model` (`LogRequest`, `LogResponse`, `Transaction`).

### Pay

- **`PayUSDC(filePath string, password []byte, toAddress, amount string, cooldownMinutes int) (*model.PayResponse, error)`**  
  Sends USDC to `toAddress`. `amount` is decimal string (e.g. `"10.50"`). `cooldownMinutes`: 0 to disable cooldown. Returns `TxID` in `*model.PayResponse`.
- **`PaySOL(filePath string, password []byte, toAddress, amount string, cooldownMinutes int) (*model.PayResponse, error)`**  
  Sends SOL; same pattern. Fee is 5000 lamports (0.000005 SOL); account for it when sending full balance.

**Models:** `PayResponse`, `PayRequest`, `LogRequest`, `LogResponse`, `SolanaBalanceResponse`, `Transaction` live in `internal/model`. Use them when calling the library and when mapping to your own types.

---

## Security and precision

- **Bind:** Desktop server listens on `127.0.0.1` only.
- **Encryption:** AES-256-GCM for private key in .cwt; password from env or caller.
- **Units:** 1 SOL = 10^9 lamports, 1 USDC = 10^6 micro-USDC; no float in calculations.

### .cwt file

Contains (among others): `network`, `address`, `QR` (base64), `salt`, `nonce`, `cipherText`. Salt and nonce are per-file random.