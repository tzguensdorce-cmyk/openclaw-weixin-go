// Package store provides default persistence for account, context token, and sync cursor data.
// Author: jtai团队（曾能混&tang先森） <jwhna1@gmil.com>
// Official Site: https://jtai.cc
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const DefaultSubDir = "wechat"

// Account is the persisted login session.
type Account struct {
	Token     string `json:"token"`
	BaseURL   string `json:"baseUrl"`
	AccountID string `json:"accountId"`
	UserID    string `json:"userId,omitempty"`
	SavedAt   string `json:"savedAt"`
}

// ContextStore persists from_user_id -> context_token mappings.
type ContextStore interface {
	LoadContextTokens() (map[string]string, error)
	SaveContextTokens(map[string]string) error
}

// SyncStore persists the getupdates cursor.
type SyncStore interface {
	LoadSyncCursor() (string, error)
	SaveSyncCursor(string) error
}

// FileStore stores data in <BaseDir>/<SubDir>/.
type FileStore struct {
	BaseDir string
	SubDir  string
}

// NewFileStore creates a file-backed store rooted at baseDir.
func NewFileStore(baseDir string) *FileStore {
	return &FileStore{
		BaseDir: baseDir,
		SubDir:  DefaultSubDir,
	}
}

// Dir returns the effective store directory.
func (s *FileStore) Dir() string {
	subDir := s.SubDir
	if subDir == "" {
		subDir = DefaultSubDir
	}
	return filepath.Join(s.BaseDir, subDir)
}

func (s *FileStore) accountPath() string    { return filepath.Join(s.Dir(), "account.json") }
func (s *FileStore) contextPath() string    { return filepath.Join(s.Dir(), "ctx_tokens.json") }
func (s *FileStore) syncCursorPath() string { return filepath.Join(s.Dir(), "sync_buf.txt") }

func (s *FileStore) ensureDir() error {
	return os.MkdirAll(s.Dir(), 0o700)
}

// LoadAccount returns nil, nil when the account file does not exist.
func (s *FileStore) LoadAccount() (*Account, error) {
	data, err := os.ReadFile(s.accountPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var acct Account
	if err := json.Unmarshal(data, &acct); err != nil {
		return nil, err
	}
	return &acct, nil
}

// SaveAccount writes account.json with 0600 permissions.
func (s *FileStore) SaveAccount(acct *Account) error {
	if acct == nil {
		return fmt.Errorf("account is nil")
	}
	if err := s.ensureDir(); err != nil {
		return err
	}
	acct.SavedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(acct, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.accountPath(), data, 0o600)
}

// ClearAccount removes account.json if present.
func (s *FileStore) ClearAccount() error {
	err := os.Remove(s.accountPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// LoadContextTokens returns an empty map when the file does not exist.
func (s *FileStore) LoadContextTokens() (map[string]string, error) {
	data, err := os.ReadFile(s.contextPath())
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var tokens map[string]string
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, err
	}
	if tokens == nil {
		tokens = map[string]string{}
	}
	return tokens, nil
}

// SaveContextTokens writes ctx_tokens.json with 0600 permissions.
func (s *FileStore) SaveContextTokens(tokens map[string]string) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.contextPath(), data, 0o600)
}

// LoadSyncCursor returns an empty string when the file does not exist.
func (s *FileStore) LoadSyncCursor() (string, error) {
	data, err := os.ReadFile(s.syncCursorPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveSyncCursor writes sync_buf.txt with 0600 permissions.
func (s *FileStore) SaveSyncCursor(cursor string) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	return os.WriteFile(s.syncCursorPath(), []byte(cursor), 0o600)
}
