package hashutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const HeaderName = "HashSHA256"

func SignBody(body []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func VerifySignature(body []byte, signature string, key string) bool {
	expected := SignBody(body, key)
	return hmac.Equal([]byte(expected), []byte(signature))
}
