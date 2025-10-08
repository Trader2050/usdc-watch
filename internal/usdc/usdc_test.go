package usdc

import (
	"math/big"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{input: "0", expected: "0"},
		{input: "1", expected: "1000000"},
		{input: "1.23", expected: "1230000"},
		{input: "1.234567", expected: "1234567"},
		{input: "0002.5", expected: "2500000"},
		{input: "-1", wantErr: true},
		{input: "1.2345678", wantErr: true},
		{input: "abc", wantErr: true},
	}

	for _, tc := range tests {
		got, err := ParseAmount(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseAmount(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseAmount(%q) unexpected error: %v", tc.input, err)
		}
		if got.String() != tc.expected {
			t.Fatalf("ParseAmount(%q) = %s, expected %s", tc.input, got.String(), tc.expected)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		value    *big.Int
		expected string
	}{
		{big.NewInt(0), "0"},
		{big.NewInt(1_000_000), "1"},
		{big.NewInt(1_230_000), "1.230000"},
		{big.NewInt(1_234_567), "1.234567"},
	}

	for _, tc := range cases {
		got := FormatAmount(tc.value)
		if got != tc.expected {
			t.Fatalf("FormatAmount(%s) = %s, expected %s", tc.value.String(), got, tc.expected)
		}
	}
}

func TestEncodeBalanceOfCall(t *testing.T) {
	data, err := EncodeBalanceOfCall("0x0000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("EncodeBalanceOfCall error: %v", err)
	}
	expected := "0x70a082310000000000000000000000000000000000000000000000000000000000000001"
	if data != expected {
		t.Fatalf("EncodeBalanceOfCall mismatch: got %s, expected %s", data, expected)
	}
}
