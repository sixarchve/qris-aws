package qris

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Codec struct{}

func NewCodec() Codec {
	return Codec{}
}

func (Codec) GeneratePayload(amount int, merchantName string, qrID string) (string, error) {
	return GeneratePayload(amount, merchantName, qrID)
}

func (Codec) ParsePayload(payload string) (string, int, error) {
	return ParsePayload(payload)
}

func tlv(tag string, value string) string {
	length := fmt.Sprintf("%02d", len(value))
	return tag + length + value
}

func crc16(input string) string {
	var crc uint16 = 0xFFFF
	for i := 0; i < len(input); i++ {
		crc ^= uint16(input[i]) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

func cleanQRISValue(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII || unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)

	if len(value) > maxLength {
		return value[:maxLength]
	}

	return value
}

func GeneratePayload(amount int, merchantName string, qrID string) (string, error) {
	merchantName = cleanQRISValue(strings.ToUpper(merchantName), 25)
	qrID = cleanQRISValue(qrID, 99)
	if merchantName == "" {
		return "", errors.New("merchant name is required")
	}
	if qrID == "" {
		return "", errors.New("qr id is required")
	}
	if amount <= 0 {
		return "", errors.New("amount must be greater than zero")
	}

	payload := ""

	// payload format
	payload += tlv("00", "01")

	// dynamic QR
	payload += tlv("01", "12")

	// merchant info
	merchant := ""
	merchant += tlv("00", "ID.CO.QRIS.WWW")
	merchant += tlv("01", qrID)
	payload += tlv("26", merchant)

	// Merchant category code
	payload += tlv("52", "0000")

	// currency IDR
	payload += tlv("53", "360")

	// amount
	payload += tlv("54", strconv.Itoa(amount))

	// country
	payload += tlv("58", "ID")

	// merchant name
	payload += tlv("59", merchantName)

	// city
	payload += tlv("60", "MALANG")

	// CRC placeholder
	payload += "6304"

	crc := crc16(payload)
	payload += crc

	return payload, nil
}

func ParsePayload(payload string) (string, int, error) {
	if len(payload) < 8 {
		return "", 0, errors.New("invalid qris payload")
	}

	if payload[len(payload)-8:len(payload)-4] != "6304" {
		return "", 0, errors.New("invalid qris crc tag")
	}

	expectedCRC := crc16(payload[:len(payload)-4])
	if !strings.EqualFold(expectedCRC, payload[len(payload)-4:]) {
		return "", 0, errors.New("invalid qris crc")
	}

	values, err := parseTLV(payload[:len(payload)-8])
	if err != nil {
		return "", 0, err
	}

	merchantInfo, ok := values["26"]
	if !ok {
		return "", 0, errors.New("qris merchant info is required")
	}

	merchantValues, err := parseTLV(merchantInfo)
	if err != nil {
		return "", 0, err
	}

	qrID := merchantValues["01"]
	if qrID == "" {
		return "", 0, errors.New("qris merchant id is required")
	}

	amount, err := strconv.Atoi(values["54"])
	if err != nil || amount <= 0 {
		return "", 0, errors.New("qris amount is invalid")
	}

	return qrID, amount, nil
}

func parseTLV(payload string) (map[string]string, error) {
	values := map[string]string{}

	for i := 0; i < len(payload); {
		if i+4 > len(payload) {
			return nil, errors.New("invalid qris tlv")
		}

		tag := payload[i : i+2]
		length, err := strconv.Atoi(payload[i+2 : i+4])
		if err != nil || length < 0 {
			return nil, errors.New("invalid qris tlv length")
		}

		valueStart := i + 4
		valueEnd := valueStart + length
		if valueEnd > len(payload) {
			return nil, errors.New("invalid qris tlv value")
		}

		values[tag] = payload[valueStart:valueEnd]
		i = valueEnd
	}

	return values, nil
}
