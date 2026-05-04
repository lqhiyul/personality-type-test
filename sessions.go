package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const sessionTTL = 24 * time.Hour

type sessionStore struct {
	mu sync.Mutex
	m  map[string]time.Time
}

var adminSessions = sessionStore{m: map[string]time.Time{}}

func newSessionToken() string {
	if token, ok := randomHex(32); ok {
		return token
	}
	return newID()
}

func storeSession(token string) {
	adminSessions.mu.Lock()
	defer adminSessions.mu.Unlock()
	adminSessions.m[token] = time.Now().Add(sessionTTL)
}

func deleteSession(token string) {
	adminSessions.mu.Lock()
	defer adminSessions.mu.Unlock()
	delete(adminSessions.m, token)
}

func checkSession(token string) bool {
	adminSessions.mu.Lock()
	defer adminSessions.mu.Unlock()

	expiresAt, ok := adminSessions.m[token]
	if !ok {
		return false
	}
	if time.Now().After(expiresAt) {
		delete(adminSessions.m, token)
		return false
	}
	return true
}

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
