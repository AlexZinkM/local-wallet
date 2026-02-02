package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	coingeckoAPI = "https://api.coingecko.com/api/v3"
)

// CoinGeckoClient client for CoinGecko API
type CoinGeckoClient struct {
	baseURL string
	client  *http.Client
}

// NewCoinGeckoClient creates a new CoinGecko client
func NewCoinGeckoClient() *CoinGeckoClient {
	return &CoinGeckoClient{
		baseURL: coingeckoAPI,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// PriceResponse response from CoinGecko API
type PriceResponse struct {
	USDCoin struct {
		Rub float64 `json:"rub"`
	} `json:"usd-coin"`
}

// GetUSDCToRUBRate gets USDC to RUB exchange rate
func (c *CoinGeckoClient) GetUSDCtoRUBrate() (string, error) {
	url := fmt.Sprintf("%s/simple/price?ids=usd-coin&vs_currencies=rub", c.baseURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get rate: status %d", resp.StatusCode)
	}

	var priceResp PriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return "", fmt.Errorf("failed to decode rate: %w", err)
	}

	rate := strconv.FormatFloat(priceResp.USDCoin.Rub, 'f', 2, 64)
	return rate, nil
}
