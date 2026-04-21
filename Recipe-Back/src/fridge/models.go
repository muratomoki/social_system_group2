package fridge

import "time"

// Item は冷蔵庫の食材1品
type Item struct {
	Name       string    `json:"name"`
	Amount     string    `json:"amount"`
	ExpiryDate time.Time `json:"expiryDate"`
	Category   string    `json:"category"` // 例: "野菜", "肉", "乳製品"
}

// Inventory は冷蔵庫の在庫全体
type Inventory struct {
	Items     []Item    `json:"items"`
	FetchedAt time.Time `json:"fetchedAt"`
}
