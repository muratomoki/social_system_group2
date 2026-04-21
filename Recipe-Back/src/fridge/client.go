package fridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Client は冷蔵庫在庫管理システムとのインターフェース
type Client interface {
	GetInventory(ctx context.Context) (Inventory, error)
}

type httpClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient は FRIDGE_API_BASE_URL に接続するHTTPクライアントを返す
func NewHTTPClient(baseURL string) Client {
	return &httpClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetInventory は冷蔵庫システムから在庫を取得する。
// システムが利用不可の場合はエラーにせず空のInventoryを返す（グレースフルデグラデーション）。
func (c *httpClient) GetInventory(ctx context.Context) (Inventory, error) {
	url := c.baseURL + "/inventory"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return emptyInventory(), nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("fridge system unavailable", "error", err)
		return emptyInventory(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("fridge system returned non-200", "status", resp.StatusCode)
		return emptyInventory(), nil
	}

	var inventory Inventory
	if err := json.NewDecoder(resp.Body).Decode(&inventory); err != nil {
		return Inventory{}, fmt.Errorf("fridge inventory decode failed: %w", err)
	}
	inventory.FetchedAt = time.Now()
	return inventory, nil
}

func emptyInventory() Inventory {
	return Inventory{Items: []Item{}, FetchedAt: time.Now()}
}
