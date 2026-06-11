package qris

import (
	"strings"
	"testing"
)

func TestGeneratePayload_Valid(t *testing.T) {
	payload, err := GeneratePayload(50000, "Kantin FILKOM UB", "TEST001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payload == "" {
		t.Fatal("expected non-empty payload")
	}

	// Verify payload starts with format indicator "000201"
	if !strings.HasPrefix(payload, "0002") {
		t.Errorf("payload should start with tag 00, got: %s", payload[:4])
	}

	// Verify payload contains CRC tag "6304" near the end
	if !strings.Contains(payload, "6304") {
		t.Error("payload should contain CRC tag '6304'")
	}

	// Verify CRC is 4 hex characters at the end
	crc := payload[len(payload)-4:]
	for _, c := range crc {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			t.Errorf("CRC should be hex, got character: %c", c)
		}
	}
}

func TestGeneratePayload_ContainsMerchantID(t *testing.T) {
	payload, err := GeneratePayload(1000, "Test Store", "MYSHOP01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(payload, "MYSHOP01") {
		t.Error("payload should contain merchant QR ID 'MYSHOP01'")
	}
}

func TestGeneratePayload_ContainsAmount(t *testing.T) {
	payload, err := GeneratePayload(25000, "Test Store", "TEST001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Amount "25000" should appear in the payload
	if !strings.Contains(payload, "25000") {
		t.Error("payload should contain amount '25000'")
	}
}

func TestGeneratePayload_InvalidInputs(t *testing.T) {
	tests := []struct {
		name         string
		amount       int
		merchantName string
		qrID         string
	}{
		{"zero amount", 0, "Store", "TEST001"},
		{"negative amount", -100, "Store", "TEST001"},
		{"empty merchant name", 1000, "", "TEST001"},
		{"empty qr id", 1000, "Store", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GeneratePayload(tt.amount, tt.merchantName, tt.qrID)
			if err == nil {
				t.Error("expected error for invalid input")
			}
		})
	}
}

func TestParsePayload_RoundTrip(t *testing.T) {
	// Generate a payload
	payload, err := GeneratePayload(15000, "Kantin FILKOM UB", "TEST001")
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Parse it back
	qrID, amount, err := ParsePayload(payload)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if qrID != "TEST001" {
		t.Errorf("expected qrID 'TEST001', got '%s'", qrID)
	}

	if amount != 15000 {
		t.Errorf("expected amount 15000, got %d", amount)
	}
}

func TestParsePayload_MultipleAmounts(t *testing.T) {
	amounts := []int{1000, 5000, 50000, 100000, 999999}

	for _, expected := range amounts {
		payload, err := GeneratePayload(expected, "Store", "TEST001")
		if err != nil {
			t.Fatalf("generate error for amount %d: %v", expected, err)
		}

		_, amount, err := ParsePayload(payload)
		if err != nil {
			t.Fatalf("parse error for amount %d: %v", expected, err)
		}

		if amount != expected {
			t.Errorf("expected amount %d, got %d", expected, amount)
		}
	}
}

func TestParsePayload_InvalidPayloads(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{"empty string", ""},
		{"too short", "0002"},
		{"no CRC tag", "000201011226290015ID.CO.QRIS.WWW0107TEST001"},
		{"bad CRC", "000201011226290015ID.CO.QRIS.WWW0107TEST00152040000530336054051000058021D5917KANTIN FILKOM UB6006MALANG6304ZZZZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParsePayload(tt.payload)
			if err == nil {
				t.Error("expected error for invalid payload")
			}
		})
	}
}

func TestCRC16(t *testing.T) {
	// CRC should produce consistent 4-character hex output
	result := crc16("hello")
	if len(result) != 4 {
		t.Errorf("expected 4-char CRC, got %d chars: '%s'", len(result), result)
	}

	// Same input should produce same CRC
	result2 := crc16("hello")
	if result != result2 {
		t.Errorf("CRC not deterministic: '%s' vs '%s'", result, result2)
	}

	// Different input should produce different CRC
	result3 := crc16("world")
	if result == result3 {
		t.Error("different inputs produced same CRC - unexpected collision")
	}
}

func TestParseTLV_Valid(t *testing.T) {
	// "00" tag, length "02", value "01"
	tlvStr := "000201"
	values, err := parseTLV(tlvStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if values["00"] != "01" {
		t.Errorf("expected tag 00 = '01', got '%s'", values["00"])
	}
}

func TestParseTLV_MultipleTags(t *testing.T) {
	// Two TLV entries: 00 02 01 | 01 02 12
	tlvStr := "00020101021252040000"
	values, err := parseTLV(tlvStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if values["00"] != "01" {
		t.Errorf("expected tag 00 = '01', got '%s'", values["00"])
	}
	if values["01"] != "12" {
		t.Errorf("expected tag 01 = '12', got '%s'", values["01"])
	}
	if values["52"] != "0000" {
		t.Errorf("expected tag 52 = '0000', got '%s'", values["52"])
	}
}

func TestParseTLV_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{"truncated tag", "00"},
		{"truncated length", "000"},
		{"length exceeds payload", "000501"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTLV(tt.payload)
			if err == nil {
				t.Error("expected error for invalid TLV")
			}
		})
	}
}

func TestCleanQRISValue(t *testing.T) {
	// Test truncation
	result := cleanQRISValue("ABCDEFGHIJ", 5)
	if result != "ABCDE" {
		t.Errorf("expected 'ABCDE', got '%s'", result)
	}

	// Test whitespace trimming
	result = cleanQRISValue("  hello  ", 10)
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}

	// Test non-ASCII removal
	result = cleanQRISValue("hello\u00e9world", 20)
	if strings.Contains(result, "\u00e9") {
		t.Error("non-ASCII characters should be removed")
	}
}

func TestTLV_Format(t *testing.T) {
	result := tlv("26", "HELLO")
	expected := "2605HELLO"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}

	// Test single char value
	result = tlv("00", "A")
	expected = "0001A"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}
