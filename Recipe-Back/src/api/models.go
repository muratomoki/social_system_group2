package api

// SuggestRecipesRequest は POST /api/v1/recipes/suggest のリクエストボディ
// @Description レシピ提案リクエスト
type SuggestRecipesRequest struct {
	// 追加リクエスト（例: "30分以内で作れるもの"）。最大1000文字。
	AdditionalNotes string `json:"additionalNotes,omitempty"`
}

// SuggestRecipesResponse は POST /api/v1/recipes/suggest のレスポンスボディ
// @Description レシピ提案レスポンス
type SuggestRecipesResponse struct {
	Recipes []Recipe `json:"recipes"`
}

// Recipe はAIが提案する1件のレシピ
// @Description レシピ情報
type Recipe struct {
	// レシピ名
	Name string `json:"name"`
	// レシピ共有サイトの実在URL
	URL string `json:"url"`
	// 1〜2文の説明
	Description string `json:"description"`
	// 冷蔵庫食材との一致度: "高" / "中" / "低"
	MatchScore string `json:"matchScore"`
	// 材料リスト
	Ingredients []Ingredient `json:"ingredients"`
	// 調理手順リスト
	Steps []Step `json:"steps"`
}

// Ingredient はレシピの材料1品
// @Description 材料
type Ingredient struct {
	// 食材名
	Name string `json:"name"`
	// 分量（例: "200g"）
	Amount string `json:"amount"`
	// 冷蔵庫にある食材かどうか
	IsInFridge bool `json:"isInFridge"`
}

// Step は調理手順の1ステップ
// @Description 調理手順
type Step struct {
	// 手順番号（1始まり）
	Order int `json:"order"`
	// 手順の説明
	Description string `json:"description"`
}

// Preferences はユーザーの料理設定
// @Description ユーザー設定
type Preferences struct {
	// 常備調味料リスト（例: ["醤油", "みりん"]）
	Condiments []string `json:"condiments"`
	// 個人の趣向・メモ（例: "辛いものが好き、魚は苦手"）
	PersonalNotes string `json:"personalNotes"`
}

// ErrorResponse は標準エラーレスポンス
// @Description エラーレスポンス
type ErrorResponse struct {
	// ユーザー向けエラーメッセージ
	Error string `json:"error"`
	// マシン向けエラーコード
	Code string `json:"code"`
	// 追加情報（任意）
	Details string `json:"details,omitempty"`
}
