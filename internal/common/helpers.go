package common

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	SOLDecimals  = 9 // SOL has 9 decimals (lamports)
	USDCDecimals = 6 // USDC has 6 decimals (micro)
)

// LamportsToSOL converts lamports to SOL string without float precision loss
func LamportsToSOL(lamports uint64) string {
	return formatWithDecimals(lamports, SOLDecimals)
}

// SOLToLamports converts SOL string to lamports without float precision loss
func SOLToLamports(sol string) (uint64, error) {
	return parseWithDecimals(sol, SOLDecimals)
}

// MicroToUSDC converts micro units to USDC string without float precision loss
func MicroToUSDC(micro uint64) string {
	return formatWithDecimals(micro, USDCDecimals)
}

// USDCToMicro converts USDC string to micro units without float precision loss
func USDCToMicro(usdc string) (uint64, error) {
	return parseWithDecimals(usdc, USDCDecimals)
}

// formatWithDecimals converts integer to decimal string by inserting decimal point
// Example: formatWithDecimals(24981836, 9) = "0.024981836"
func formatWithDecimals(value uint64, decimals int) string {
	s := fmt.Sprintf("%d", value)

	// Pad with leading zeros if needed
	for len(s) <= decimals {
		s = "0" + s
	}

	// Insert decimal point
	pos := len(s) - decimals
	return s[:pos] + "." + s[pos:]
}

// parseWithDecimals converts decimal string to integer by removing decimal point
// Example: parseWithDecimals("0.024981836", 9) = 24981836
func parseWithDecimals(s string, decimals int) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	parts := strings.Split(s, ".")

	if len(parts) == 1 {
		// No decimal point - multiply by 10^decimals
		n, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return 0, err
		}
		for i := 0; i < decimals; i++ {
			n *= 10
		}
		return n, nil
	}

	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid decimal format")
	}

	whole := parts[0]
	frac := parts[1]

	// Pad or truncate fractional part to exact decimals
	if len(frac) < decimals {
		frac += strings.Repeat("0", decimals-len(frac))
	} else if len(frac) > decimals {
		frac = frac[:decimals]
	}

	// Combine and parse
	combined := whole + frac
	return strconv.ParseUint(combined, 10, 64)
}

// CompareUSDCAmounts compares two USDC decimal string amounts without float precision loss.
// Returns: -1 if a < b, 0 if a == b, 1 if a > b, and error if parsing fails
func CompareUSDCAmounts(a, b string) (int, error) {
	aVal, err := parseWithDecimals(a, USDCDecimals)
	if err != nil {
		return 0, fmt.Errorf("failed to parse amount '%s': %w", a, err)
	}

	bVal, err := parseWithDecimals(b, USDCDecimals)
	if err != nil {
		return 0, fmt.Errorf("failed to parse amount '%s': %w", b, err)
	}

	if aVal < bVal {
		return -1, nil
	}
	if aVal > bVal {
		return 1, nil
	}
	return 0, nil
}
