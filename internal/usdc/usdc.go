package usdc

import (
	"fmt"
	"math/big"
	"strings"

	"usdc-watch/internal/eth"
)

const (
	// ContractAddress is the canonical USDC contract address on Ethereum mainnet.
	ContractAddress = "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	methodBalanceOf = "70a08231"
	decimals        = 6
	decimalFactor   = int64(1_000_000)
)

// EncodeBalanceOfCall builds the data payload for an ERC-20 balanceOf(address) call.
func EncodeBalanceOfCall(address string) (string, error) {
	addrHex, err := eth.AddressDataHex(address)
	if err != nil {
		return "", err
	}
	if len(addrHex) != 40 {
		return "", fmt.Errorf("unexpected address length: %d", len(addrHex))
	}
	return "0x" + methodBalanceOf + strings.Repeat("0", 64-len(addrHex)) + addrHex, nil
}

// ParseAmount converts a human-readable USDC amount into base units (6 decimals).
func ParseAmount(input string) (*big.Int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("amount is empty")
	}
	if strings.HasPrefix(trimmed, "-") {
		return nil, fmt.Errorf("amount cannot be negative")
	}
	if strings.HasPrefix(trimmed, "+") {
		trimmed = trimmed[1:]
	}
	parts := strings.SplitN(trimmed, ".", 2)
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if whole == "" {
		whole = "0"
	}
	if strings.ContainsAny(whole, " _") || strings.ContainsAny(frac, " _") {
		return nil, fmt.Errorf("amount cannot contain spaces or underscores")
	}
	if len(frac) > decimals {
		return nil, fmt.Errorf("amount has more than %d decimal places", decimals)
	}
	frac = frac + strings.Repeat("0", decimals-len(frac))
	combined := whole + frac
	amount := new(big.Int)
	if _, ok := amount.SetString(combined, 10); !ok {
		return nil, fmt.Errorf("invalid amount: %s", input)
	}
	return amount, nil
}

// FormatAmount renders a base-unit amount into a human-readable decimal string.
func FormatAmount(amount *big.Int) string {
	if amount == nil {
		return "0"
	}
	divisor := big.NewInt(decimalFactor)
	intPart := new(big.Int).Quo(amount, divisor)
	fracPart := new(big.Int).Mod(amount, divisor)
	if fracPart.Sign() == 0 {
		return intPart.String()
	}
	return fmt.Sprintf("%s.%06d", intPart.String(), fracPart.Int64())
}
