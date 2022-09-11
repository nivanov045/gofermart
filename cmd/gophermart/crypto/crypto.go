package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type crypto struct {
	key string
}

func New(key string) *crypto {
	return &crypto{key: key}
}

func (crypto *crypto) CreateHash(s string) string {
	h := hmac.New(sha256.New, []byte(crypto.key))
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
