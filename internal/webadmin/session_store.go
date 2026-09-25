package webadmin

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiredAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	ttl      time.Duration
}

func NewSessionStore(ttl time.Duration) *SessionStore {
	s := &SessionStore{
		sessions: map[string]Session{},
		ttl:      ttl,
	}
	go s.cleanLoop()
	return s
}

func (s *SessionStore) Create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	token := hex.EncodeToString(b)
	now := time.Now()

	s.mu.Lock()
	s.sessions[token] = Session{
		Token:     token,
		CreatedAt: now,
		ExpiredAt: now.Add(s.ttl),
	}
	s.mu.Unlock()
	return token, nil
}

func (s *SessionStore) Get(token string) (Session, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiredAt) {
		return Session{}, false
	}
	return sess, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	s.deleteSession(token)
	s.mu.Unlock()
}

func (s *SessionStore) cleanLoop() {
	t := time.NewTicker(time.Minute)
	for range t.C {
		now := time.Now()
		s.mu.Lock()
		for token, session := range s.sessions {
			if now.After(session.ExpiredAt) {
				s.deleteSession(token)
			}
		}
		s.mu.Unlock()
	}
}

func (s *SessionStore) deleteSession(token string) {
	delete(s.sessions, token)
}
