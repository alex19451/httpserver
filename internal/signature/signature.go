package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func CalculateHash(data []byte, key string) string {
	if key == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func VerifyHash(data []byte, expectedHash string, key string) bool {
	if key == "" {
		return true
	}
	actualHash := CalculateHash(data, key)
	return hmac.Equal([]byte(actualHash), []byte(expectedHash))
}
