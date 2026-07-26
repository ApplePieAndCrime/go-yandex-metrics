package hashutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HeaderName содержит имя HTTP-заголовка с подписью тела сообщения.
const HeaderName = "HashSHA256"

// SignBody вычисляет HMAC-SHA256 тела сообщения и возвращает подпись в шестнадцатеричном виде.
func SignBody(body []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature проверяет HMAC-SHA256 подпись тела сообщения.
func VerifySignature(body []byte, signature string, key string) bool {
	expected := SignBody(body, key)
	return hmac.Equal([]byte(expected), []byte(signature))
}
