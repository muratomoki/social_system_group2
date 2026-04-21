package ai

// Recipe はAIが提案するレシピ（ai内部型）
type Recipe struct {
	Name        string
	URL         string
	Description string
	MatchScore  string
	Ingredients []Ingredient
	Steps       []Step
}

// Ingredient はレシピの材料1品
type Ingredient struct {
	Name       string
	Amount     string
	IsInFridge bool
}

// Step は調理手順の1ステップ
type Step struct {
	Order       int
	Description string
}
