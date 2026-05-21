package hashutil

import (
	"crypto/sha256"
	"fmt"
)

const HeaderName = "HashSHA256"

func HashBody(body []byte, key string) string {
	payload := append(body, key...)
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum)
}

func IsValid(body []byte, key string, received string) bool {
	return received == HashBody(body, key)
}
