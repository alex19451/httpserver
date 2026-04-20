package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

func CalculateHash(data []byte, key string) string {
	if key == "" {
		return ""
	}

	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func CalculateHashFromReader(r io.Reader, key string) ([]byte, string, error) {
	if key == "" {
		data, err := io.ReadAll(r)
		return data, "", err
	}

	h := hmac.New(sha256.New, []byte(key))
	tee := io.TeeReader(r, h)
	data, err := io.ReadAll(tee)
	if err != nil {
		return nil, "", err
	}

	return data, hex.EncodeToString(h.Sum(nil)), nil
}

func VerifyHash(data []byte, expectedHash string, key string) bool {
	if key == "" {
		return true
	}

	actualHash := CalculateHash(data, key)
	return hmac.Equal([]byte(actualHash), []byte(expectedHash))
}
