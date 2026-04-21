package ai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Client はAI（Claude）へのレシピ提案依頼インターフェース
type Client interface {
	SuggestRecipes(ctx context.Context, req SuggestRequest) ([]Recipe, error)
}

type anthropicClient struct {
	client       *anthropic.Client
	model        string
	cache        *responseCache
	cacheEnabled bool
}

// NewAnthropicClient はAnthropicAPIを使うClientを返す
func NewAnthropicClient(apiKey, model string, cacheEnabled bool) Client {
	c := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &anthropicClient{
		client:       c,
		model:        model,
		cache:        newResponseCache(10 * time.Minute),
		cacheEnabled: cacheEnabled,
	}
}

func (c *anthropicClient) SuggestRecipes(ctx context.Context, req SuggestRequest) ([]Recipe, error) {
	if err := validateSuggestRequest(req); err != nil {
		return nil, err
	}

	if c.cacheEnabled {
		key := cacheKey(req)
		if cached, ok := c.cache.get(key); ok {
			return cached, nil
		}
	}

	recipes, err := c.callAPI(ctx, req)
	if err != nil {
		return nil, err
	}

	if c.cacheEnabled {
		c.cache.set(cacheKey(req), recipes)
	}
	return recipes, nil
}

func (c *anthropicClient) callAPI(ctx context.Context, req SuggestRequest) ([]Recipe, error) {
	userMsg := BuildUserMessage(req)
	sysPrompt := BuildSystemPrompt()

	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.Model(c.model)),
		MaxTokens: anthropic.F(int64(4096)),
		System: anthropic.F([]anthropic.TextBlockParam{
			{
				Type: anthropic.F(anthropic.TextBlockParamTypeText),
				Text: anthropic.F(sysPrompt),
				CacheControl: anthropic.F(anthropic.CacheControlEphemeralParam{
					Type: anthropic.F(anthropic.CacheControlEphemeralTypeEphemeral),
				}),
			},
		}),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API call failed: %w", err)
	}

	rawText := extractTextContent(msg)
	if rawText == "" {
		return nil, fmt.Errorf("anthropic API returned empty response")
	}

	recipes, err := ParseRecipeSuggestions(rawText)
	if err != nil {
		return nil, fmt.Errorf("recipe parse failed: %w", err)
	}
	return recipes, nil
}

func extractTextContent(msg *anthropic.Message) string {
	for _, block := range msg.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}

func validateSuggestRequest(req SuggestRequest) error {
	if len(req.AdditionalNotes) > 1000 {
		return fmt.Errorf("additionalNotes exceeds maximum length of 1000 characters")
	}
	return nil
}

// cacheKey はSuggestRequestの内容からキャッシュキーを生成する
func cacheKey(req SuggestRequest) string {
	data, _ := json.Marshal(req)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

// --- インメモリキャッシュ ---

type cacheEntry struct {
	recipes   []Recipe
	expiresAt time.Time
}

type responseCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

func newResponseCache(ttl time.Duration) *responseCache {
	return &responseCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

func (rc *responseCache) get(key string) ([]Recipe, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	entry, ok := rc.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.recipes, true
}

func (rc *responseCache) set(key string, recipes []Recipe) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries[key] = cacheEntry{
		recipes:   recipes,
		expiresAt: time.Now().Add(rc.ttl),
	}
}
