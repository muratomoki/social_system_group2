package user

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store はユーザー設定の永続化インターフェース
type Store interface {
	GetPreferences(ctx context.Context) (Preferences, error)
	UpdatePreferences(ctx context.Context, prefs Preferences) error
}

type fileStore struct {
	filePath string
	mu       sync.RWMutex
}

// NewFileStore は指定パスにJSONファイルで設定を永続化するStoreを返す
func NewFileStore(filePath string) Store {
	return &fileStore{filePath: filePath}
}

func (s *fileStore) GetPreferences(ctx context.Context) (Preferences, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return Preferences{Condiments: []string{}}, nil
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("preferences file read failed: %w", err)
	}

	var prefs Preferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Preferences{}, fmt.Errorf("preferences file parse failed: %w", err)
	}
	if prefs.Condiments == nil {
		prefs.Condiments = []string{}
	}
	return prefs, nil
}

func (s *fileStore) UpdatePreferences(ctx context.Context, prefs Preferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefs.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("preferences marshal failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return fmt.Errorf("preferences directory creation failed: %w", err)
	}

	// アトミックな書き込み（一時ファイル → リネーム）
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("preferences temp file write failed: %w", err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("preferences file rename failed: %w", err)
	}
	return nil
}
