package app

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"
)

const sessionTTL = 24 * time.Hour

func subtleCompare(a, b string) bool {
	if len(a) != len(b) {
		subtle.ConstantTimeCompare([]byte(a), []byte(a))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func newID() string {
	if id, ok := randomHex(12); ok {
		return id
	}
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func randomHex(size int) (string, bool) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", false
	}
	return hex.EncodeToString(b), true
}
