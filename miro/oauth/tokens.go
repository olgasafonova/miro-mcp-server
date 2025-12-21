package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// TokenStore defines the interface for token persistence.
type TokenStore interface {
	// Save persists the token set
	Save(ctx context.Context, tokens *TokenSet) error

	// Load retrieves the stored token set
	Load(ctx context.Context) (*TokenSet, error)

	// Delete removes stored tokens
	Delete(ctx context.Context) error

	// Exists returns true if tokens are stored
	Exists(ctx context.Context) bool
}

// FileTokenStore stores tokens in a JSON file.
type FileTokenStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileTokenStore creates a new file-based token store.
func NewFileTokenStore(path string) *FileTokenStore {
	return &FileTokenStore{path: path}
}

// Save writes tokens to the file.
func (s *FileTokenStore) Save(ctx context.Context, tokens *TokenSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	// Write to temp file first, then rename (atomic)
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write tokens: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	return nil
}

// Load reads tokens from the file.
func (s *FileTokenStore) Load(ctx context.Context) (*TokenSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no tokens stored: %w", err)
		}
		return nil, fmt.Errorf("failed to read tokens: %w", err)
	}

	var tokens TokenSet
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	return &tokens, nil
}

// Delete removes the token file.
func (s *FileTokenStore) Delete(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete tokens: %w", err)
	}
	return nil
}

// Exists returns true if the token file exists.
func (s *FileTokenStore) Exists(ctx context.Context) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.path)
	return err == nil
}

// MemoryTokenStore stores tokens in memory (for testing).
type MemoryTokenStore struct {
	tokens *TokenSet
	mu     sync.RWMutex
}

// NewMemoryTokenStore creates a new in-memory token store.
func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{}
}

// Save stores tokens in memory.
func (s *MemoryTokenStore) Save(ctx context.Context, tokens *TokenSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy to prevent external modifications
	copied := *tokens
	s.tokens = &copied
	return nil
}

// Load retrieves tokens from memory.
func (s *MemoryTokenStore) Load(ctx context.Context) (*TokenSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tokens == nil {
		return nil, fmt.Errorf("no tokens stored")
	}

	// Return a copy
	copied := *s.tokens
	return &copied, nil
}

// Delete clears tokens from memory.
func (s *MemoryTokenStore) Delete(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens = nil
	return nil
}

// Exists returns true if tokens are stored.
func (s *MemoryTokenStore) Exists(ctx context.Context) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tokens != nil
}
