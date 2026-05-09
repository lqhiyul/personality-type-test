package main

import (
	"sync"
	"time"
)

const (
	userSessionCookieName = "user_session"
	userSessionTTL        = 7 * 24 * time.Hour
)

type userSession struct {
	userID    int64
	expiresAt time.Time
}

type userSessionStore struct {
	mu       sync.Mutex
	sessions map[string]userSession
	ttl      time.Duration
	now      func() time.Time
}

func newUserSessionStore(ttl time.Duration) *userSessionStore {
	if ttl <= 0 {
		ttl = userSessionTTL
	}
	return &userSessionStore{
		sessions: map[string]userSession{},
		ttl:      ttl,
		now:      time.Now,
	}
}

func (s *userSessionStore) create(userID int64) (string, time.Time) {
	token := newSessionToken()
	expiresAt := s.now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = userSession{userID: userID, expiresAt: expiresAt}
	return token, expiresAt
}

func (s *userSessionStore) delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *userSessionStore) userID(token string) (int64, bool) {
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[token]
	if !ok {
		return 0, false
	}
	if now.After(session.expiresAt) {
		delete(s.sessions, token)
		return 0, false
	}
	return session.userID, true
}
