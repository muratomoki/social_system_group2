package user

import "time"

// Preferences はユーザーの料理設定（永続化される）
type Preferences struct {
	Condiments    []string  `json:"condiments"`
	PersonalNotes string    `json:"personalNotes"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
