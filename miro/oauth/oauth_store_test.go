package oauth

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// =============================================================================
// TokenStore Tests
// =============================================================================

// mustSaveTokens saves tokens into store, failing the test on error.
func mustSaveTokens(t *testing.T, ctx context.Context, store TokenStore, tokens *TokenSet) {
	t.Helper()
	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

// mustLoadTokens loads tokens from store, failing the test on error.
func mustLoadTokens(t *testing.T, ctx context.Context, store TokenStore) *TokenSet {
	t.Helper()
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return loaded
}

// assertTokensMatch compares the identifying fields of loaded against want.
func assertTokensMatch(t *testing.T, loaded, want *TokenSet) {
	t.Helper()
	if loaded.AccessToken != want.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, want.AccessToken)
	}
	if loaded.RefreshToken != want.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, want.RefreshToken)
	}
	if loaded.UserID != want.UserID {
		t.Errorf("UserID = %q, want %q", loaded.UserID, want.UserID)
	}
}

// mustDeleteTokens deletes stored tokens, failing the test on error.
func mustDeleteTokens(t *testing.T, ctx context.Context, store TokenStore) {
	t.Helper()
	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestFileTokenStore(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, ".miro", "tokens.json")

	store := NewFileTokenStore(tokenPath)
	ctx := context.Background()

	// Initially should not exist
	if store.Exists(ctx) {
		t.Error("store should not exist initially")
	}

	// Save tokens
	tokens := &TokenSet{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		UserID:       "test-user",
	}
	mustSaveTokens(t, ctx, store, tokens)

	// Should exist now
	if !store.Exists(ctx) {
		t.Error("store should exist after save")
	}

	// Load tokens
	loaded := mustLoadTokens(t, ctx, store)
	assertTokensMatch(t, loaded, tokens)

	// Delete tokens
	mustDeleteTokens(t, ctx, store)

	if store.Exists(ctx) {
		t.Error("store should not exist after delete")
	}

	// Load should fail after delete
	if _, err := store.Load(ctx); err == nil {
		t.Error("Load() should fail after delete")
	}
}

func TestMemoryTokenStore(t *testing.T) {
	store := NewMemoryTokenStore()
	ctx := context.Background()

	// Initially should not exist
	if store.Exists(ctx) {
		t.Error("store should not exist initially")
	}

	// Save tokens
	tokens := &TokenSet{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mustSaveTokens(t, ctx, store, tokens)

	// Should exist now
	if !store.Exists(ctx) {
		t.Error("store should exist after save")
	}

	// Load tokens
	loaded := mustLoadTokens(t, ctx, store)
	if loaded.AccessToken != tokens.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tokens.AccessToken)
	}

	// Modify original should not affect stored copy
	tokens.AccessToken = "modified"
	loaded2, _ := store.Load(ctx)
	if loaded2.AccessToken == "modified" {
		t.Error("stored tokens should be a copy, not reference")
	}

	// Delete tokens
	mustDeleteTokens(t, ctx, store)

	if store.Exists(ctx) {
		t.Error("store should not exist after delete")
	}
}
