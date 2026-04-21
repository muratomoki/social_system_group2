package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AnthropicAPIKey   string
	FridgeAPIBaseURL  string
	UserPrefsFilePath string
	Port              string
	ClaudeModel       string
	CacheEnabled      bool
}

func Load() (*Config, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required")
	}

	fridgeURL := os.Getenv("FRIDGE_API_BASE_URL")
	if fridgeURL == "" {
		return nil, fmt.Errorf("FRIDGE_API_BASE_URL is required")
	}

	prefsPath := os.Getenv("USER_PREFS_FILE_PATH")
	if prefsPath == "" {
		prefsPath = "./data/preferences.json"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-opus-4-7"
	}

	cacheEnabled := true
	if v := os.Getenv("AI_CACHE_ENABLED"); v != "" {
		var err error
		cacheEnabled, err = strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("AI_CACHE_ENABLED must be true or false: %w", err)
		}
	}

	return &Config{
		AnthropicAPIKey:   apiKey,
		FridgeAPIBaseURL:  fridgeURL,
		UserPrefsFilePath: prefsPath,
		Port:              port,
		ClaudeModel:       model,
		CacheEnabled:      cacheEnabled,
	}, nil
}
