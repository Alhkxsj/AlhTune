package utils

import (
	"encoding/json"
	"os"
	"sync"
)

// CookieManager manages cookies for different music sources
// It provides thread-safe operations for loading, saving, and retrieving cookies
type CookieManager struct {
	cookies map[string]string
	mu      sync.RWMutex
}

// NewCookieManager creates a new CookieManager instance
func NewCookieManager() *CookieManager {
	return &CookieManager{
		cookies: make(map[string]string),
	}
}

// Load loads cookies from the specified file
// If the file doesn't exist or cannot be read, it silently continues
func (m *CookieManager) Load(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		// File not found is not an error for initial load
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &m.cookies); err != nil {
		return err
	}

	return nil
}

// Save saves cookies to the specified file
// Creates the directory if it doesn't exist
func (m *CookieManager) Save(filePath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create directory if it doesn't exist
	dir := filePath
	if idx := lastIndexOf(filePath, '/'); idx >= 0 {
		dir = filePath[:idx]
	}
	if dir != "" && dir != filePath {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(m.cookies, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// Get retrieves a cookie for the specified source
// Returns empty string if source not found
func (m *CookieManager) Get(source string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cookies[source]
}

// Set sets a cookie for the specified source
func (m *CookieManager) Set(source, cookie string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cookie == "" {
		delete(m.cookies, source)
	} else {
		m.cookies[source] = cookie
	}
}

// SetAll sets multiple cookies at once
// Empty cookie values will delete the corresponding entries
func (m *CookieManager) SetAll(cookies map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, v := range cookies {
		if v == "" {
			delete(m.cookies, k)
		} else {
			m.cookies[k] = v
		}
	}
}

// GetAll returns a copy of all cookies
func (m *CookieManager) GetAll() map[string]string {
	return m.copyCookies()
}

// GetSources returns a list of all source names that have cookies
func (m *CookieManager) GetSources() []string {
	cookies := m.copyCookies()
	sources := make([]string, 0, len(cookies))
	for source := range cookies {
		sources = append(sources, source)
	}
	return sources
}

// copyCookies returns a copy of all cookies (internal helper)
func (m *CookieManager) copyCookies() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string, len(m.cookies))
	for k, v := range m.cookies {
		result[k] = v
	}
	return result
}

// Has checks if a cookie exists for the specified source
func (m *CookieManager) Has(source string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.cookies[source]
	return exists
}

// Delete removes a cookie for the specified source
func (m *CookieManager) Delete(source string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cookies, source)
}

// Clear removes all cookies
func (m *CookieManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cookies = make(map[string]string)
}

// Count returns the number of cookies
func (m *CookieManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.cookies)
}

// lastIndexOf finds the last occurrence of a rune in a string
func lastIndexOf(s string, r rune) int {
	for i := len(s) - 1; i >= 0; i-- {
		if rune(s[i]) == r {
			return i
		}
	}
	return -1
}
