package ai

import (
	"context"
)

type stubClient struct{}

// NewStubClient はデバッグ用のスタブClientを返す
func NewStubClient() Client {
	return &stubClient{}
}

func (s *stubClient) SuggestRecipes(_ context.Context, req SuggestRequest) ([]Recipe, error) {
	return []Recipe{
		{
			Name:        "[STUB] 豚肉と野菜の炒め物",
			URL:         "https://cookpad.com/recipe/0000001",
			Description: "（デバッグ用スタブレスポンス）冷蔵庫の食材で作るシンプルな炒め物。",
			MatchScore:  "高",
			Ingredients: []Ingredient{
				{Name: "豚肉", Amount: "200g", IsInFridge: true},
				{Name: "キャベツ", Amount: "1/4個", IsInFridge: true},
				{Name: "醤油", Amount: "大さじ2", IsInFridge: false},
			},
			Steps: []Step{
				{Order: 1, Description: "豚肉を一口大に切る"},
				{Order: 2, Description: "フライパンで炒め、醤油で味付けする"},
			},
		},
		{
			Name:        "[STUB] 卵チャーハン",
			URL:         "https://cookpad.com/recipe/0000002",
			Description: "（デバッグ用スタブレスポンス）シンプルな卵チャーハン。",
			MatchScore:  "中",
			Ingredients: []Ingredient{
				{Name: "ご飯", Amount: "茶碗2杯", IsInFridge: true},
				{Name: "卵", Amount: "2個", IsInFridge: true},
				{Name: "塩", Amount: "少々", IsInFridge: false},
			},
			Steps: []Step{
				{Order: 1, Description: "卵を溶いてフライパンで炒める"},
				{Order: 2, Description: "ご飯を加えてほぐしながら炒め、塩で味付けする"},
			},
		},
		{
			Name:        "[STUB] 味噌汁",
			URL:         "https://cookpad.com/recipe/0000003",
			Description: "（デバッグ用スタブレスポンス）基本の味噌汁。追加リクエスト: " + req.AdditionalNotes,
			MatchScore:  "低",
			Ingredients: []Ingredient{
				{Name: "豆腐", Amount: "1/2丁", IsInFridge: true},
				{Name: "味噌", Amount: "大さじ2", IsInFridge: false},
				{Name: "だし", Amount: "400ml", IsInFridge: false},
			},
			Steps: []Step{
				{Order: 1, Description: "だしを沸かして豆腐を加える"},
				{Order: 2, Description: "味噌を溶き入れて完成"},
			},
		},
	}, nil
}
