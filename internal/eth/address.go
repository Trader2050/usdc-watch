package eth

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// NormalizeAddress ensures the provided string is a valid 20-byte Ethereum address.
func NormalizeAddress(addr string) (string, error) {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return "", fmt.Errorf("address is empty")
	}
	if strings.HasPrefix(trimmed, "0x") || strings.HasPrefix(trimmed, "0X") {
		trimmed = trimmed[2:]
	}
	if len(trimmed) != 40 {
		return "", fmt.Errorf("address must be 40 hex characters, got %d", len(trimmed))
	}
	if _, err := hex.DecodeString(trimmed); err != nil {
		return "", fmt.Errorf("invalid hex address: %w", err)
	}
	return "0x" + strings.ToLower(trimmed), nil
}

// AddressDataHex returns the lowercase hexadecimal address without the 0x prefix.
func AddressDataHex(addr string) (string, error) {
	normalized, err := NormalizeAddress(addr)
	if err != nil {
		return "", err
	}
	return normalized[2:], nil
}
