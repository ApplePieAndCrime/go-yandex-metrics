package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
)

const HeaderName = "HashSHA256"

func HashBody(body []byte, key string) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func IsValid(body []byte, key string, received string) bool {
	return received == HashBody(body, key)
}
