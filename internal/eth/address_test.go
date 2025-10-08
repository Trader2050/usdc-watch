package eth

import "testing"

func TestNormalizeAddress(t *testing.T) {
	addr, err := NormalizeAddress(" 0xABCDEFabcdefABCDEFabcdefABCDEFabcdefABCD ")
	if err != nil {
		t.Fatalf("NormalizeAddress returned error: %v", err)
	}
	expected := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	if addr != expected {
		t.Fatalf("NormalizeAddress = %s, expected %s", addr, expected)
	}
}

func TestNormalizeAddressErrors(t *testing.T) {
	cases := []string{"", "0x123", "xyz", "0xGG"}
	for _, input := range cases {
		if _, err := NormalizeAddress(input); err == nil {
			t.Fatalf("NormalizeAddress(%q) expected error", input)
		}
	}
}

func TestAddressDataHex(t *testing.T) {
	data, err := AddressDataHex("0x0000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("AddressDataHex error: %v", err)
	}
	expected := "0000000000000000000000000000000000000001"
	if data != expected {
		t.Fatalf("AddressDataHex = %s, expected %s", data, expected)
	}
}
