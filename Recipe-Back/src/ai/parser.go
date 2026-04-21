package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// aiRecipesResponse はAIが返すJSONの内部表現
type aiRecipesResponse struct {
	Recipes []aiRecipe `json:"recipes"`
}

type aiRecipe struct {
	Name        string         `json:"name"`
	URL         string         `json:"url"`
	Description string         `json:"description"`
	MatchScore  string         `json:"matchScore"`
	Ingredients []aiIngredient `json:"ingredients"`
	Steps       []aiStep       `json:"steps"`
}

type aiIngredient struct {
	Name       string `json:"name"`
	Amount     string `json:"amount"`
	IsInFridge bool   `json:"isInFridge"`
}

type aiStep struct {
	Order       int    `json:"order"`
	Description string `json:"description"`
}

// ParseRecipeSuggestions はAIのテキストレスポンスをRecipeスライスに変換する（純粋関数）
func ParseRecipeSuggestions(rawResponse string) ([]Recipe, error) {
	cleaned := extractJSON(rawResponse)
	if cleaned == "" {
		return nil, fmt.Errorf("no JSON found in AI response")
	}

	var parsed aiRecipesResponse
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("AI response JSON parse failed: %w", err)
	}

	if len(parsed.Recipes) == 0 {
		return nil, fmt.Errorf("AI returned no recipes")
	}

	recipes := make([]Recipe, 0, len(parsed.Recipes))
	for _, r := range parsed.Recipes {
		recipe, err := convertRecipe(r)
		if err != nil {
			return nil, fmt.Errorf("recipe conversion failed: %w", err)
		}
		recipes = append(recipes, recipe)
	}
	return recipes, nil
}

// extractJSON はAIレスポンスからJSONブロックを抽出する
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || start >= end {
		return ""
	}
	return s[start : end+1]
}

func convertRecipe(r aiRecipe) (Recipe, error) {
	if r.Name == "" {
		return Recipe{}, fmt.Errorf("recipe name is empty")
	}
	if !strings.HasPrefix(r.URL, "https://") {
		return Recipe{}, fmt.Errorf("recipe URL is invalid: %q", r.URL)
	}

	ingredients := make([]Ingredient, 0, len(r.Ingredients))
	for _, ing := range r.Ingredients {
		ingredients = append(ingredients, Ingredient{
			Name:       ing.Name,
			Amount:     ing.Amount,
			IsInFridge: ing.IsInFridge,
		})
	}

	steps := summarizeSteps(r.Steps)
	converted := make([]Step, 0, len(steps))
	for i, s := range steps {
		order := s.Order
		if order == 0 {
			order = i + 1
		}
		converted = append(converted, Step{
			Order:       order,
			Description: s.Description,
		})
	}

	matchScore := r.MatchScore
	if matchScore != "高" && matchScore != "中" && matchScore != "低" {
		matchScore = "中"
	}

	return Recipe{
		Name:        r.Name,
		URL:         r.URL,
		Description: r.Description,
		MatchScore:  matchScore,
		Ingredients: ingredients,
		Steps:       converted,
	}, nil
}

// summarizeSteps はAIが10件を超える手順を返した場合に10件に丸める
func summarizeSteps(steps []aiStep) []aiStep {
	const maxSteps = 10
	if len(steps) <= maxSteps {
		return steps
	}
	return steps[:maxSteps]
}
