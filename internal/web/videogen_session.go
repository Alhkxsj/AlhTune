package web

import (
	"os"
	"sync"
	"time"
)

// RenderSession represents a video rendering session
type RenderSession struct {
	ID        string
	Dir       string
	AudioPath string
	Total     int
	mu        sync.Mutex
}

// SessionManager manages video rendering sessions
type SessionManager struct {
	sessions map[string]*RenderSession
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*RenderSession),
	}
}

// Create creates a new render session
func (m *SessionManager) Create(id, dir, audioPath string) *RenderSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &RenderSession{
		ID:        id,
		Dir:       dir,
		AudioPath: audioPath,
	}
	m.sessions[id] = session
	return session
}

// Get retrieves a session by ID
func (m *SessionManager) Get(id string) (*RenderSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// Delete removes a session
func (m *SessionManager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

// Cleanup removes expired sessions
func (m *SessionManager) Cleanup(maxAge time.Duration) {
	m.mu.RLock()
	sessionIDs := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	m.mu.RUnlock()

	now := time.Now()
	for _, id := range sessionIDs {
		m.mu.RLock()
		session, ok := m.sessions[id]
		m.mu.RUnlock()

		if !ok {
			continue
		}

		info, err := os.Stat(session.Dir)
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			os.RemoveAll(session.Dir)
			m.Delete(id)
		}
	}
}
