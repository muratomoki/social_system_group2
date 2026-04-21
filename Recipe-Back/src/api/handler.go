package api

import (
	"encoding/json"
	"net/http"
	"unicode/utf8"

	"github.com/social-system-group2/recipe-back/src/ai"
	"github.com/social-system-group2/recipe-back/src/fridge"
	"github.com/social-system-group2/recipe-back/src/user"
)

// RecipeHandler は POST /api/v1/recipes/suggest を処理する
// @Tags recipes
type RecipeHandler struct {
	aiClient     ai.Client
	fridgeClient fridge.Client
	userStore    user.Store
}

// NewRecipeHandler はRecipeHandlerを返す
func NewRecipeHandler(aiClient ai.Client, fridgeClient fridge.Client, userStore user.Store) *RecipeHandler {
	return &RecipeHandler{
		aiClient:     aiClient,
		fridgeClient: fridgeClient,
		userStore:    userStore,
	}
}

// SuggestRecipes はレシピ提案を返す
// @Summary レシピ提案
// @Description 冷蔵庫の食材・常備調味料・個人の趣向をもとにAIがレシピを3件提案する
// @Accept json
// @Produce json
// @Param request body SuggestRecipesRequest false "追加リクエスト（任意）"
// @Success 200 {object} SuggestRecipesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /api/v1/recipes/suggest [post]
func (h *RecipeHandler) SuggestRecipes(w http.ResponseWriter, r *http.Request) {
	var req SuggestRecipesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Request body is invalid JSON", "")
		return
	}

	if utf8.RuneCountInString(req.AdditionalNotes) > 1000 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR",
			"Validation failed", "additionalNotes exceeds maximum length of 1000 characters")
		return
	}

	inventory, err := h.fridgeClient.GetInventory(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"Failed to get fridge inventory", err.Error())
		return
	}

	prefs, err := h.userStore.GetPreferences(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"Failed to get user preferences", err.Error())
		return
	}

	aiRecipes, err := h.aiClient.SuggestRecipes(r.Context(), ai.SuggestRequest{
		FridgeItems:     inventory.Items,
		Condiments:      prefs.Condiments,
		PersonalNotes:   prefs.PersonalNotes,
		AdditionalNotes: req.AdditionalNotes,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI_ERROR",
			"Failed to get recipe suggestions from AI", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuggestRecipesResponse{Recipes: convertRecipes(aiRecipes)})
}

// PreferencesHandler は GET/PUT /api/v1/preferences を処理する
// @Tags preferences
type PreferencesHandler struct {
	userStore user.Store
}

// NewPreferencesHandler はPreferencesHandlerを返す
func NewPreferencesHandler(userStore user.Store) *PreferencesHandler {
	return &PreferencesHandler{userStore: userStore}
}

// GetPreferences はユーザー設定を返す
// @Summary ユーザー設定取得
// @Description 常備調味料と個人の趣向を取得する
// @Produce json
// @Success 200 {object} Preferences
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/preferences [get]
func (h *PreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	prefs, err := h.userStore.GetPreferences(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"Failed to get preferences", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, Preferences{
		Condiments:    prefs.Condiments,
		PersonalNotes: prefs.PersonalNotes,
	})
}

// UpdatePreferences はユーザー設定を更新する
// @Summary ユーザー設定更新
// @Description 常備調味料と個人の趣向を更新する
// @Accept json
// @Produce json
// @Param request body Preferences true "ユーザー設定"
// @Success 200 {object} Preferences
// @Failure 400 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/preferences [put]
func (h *PreferencesHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	var req Preferences
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Request body is invalid JSON", "")
		return
	}

	if len(req.Condiments) > 50 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR",
			"Validation failed", "condiments: max 50 items allowed")
		return
	}
	for _, c := range req.Condiments {
		if utf8.RuneCountInString(c) > 50 {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR",
				"Validation failed", "condiments: each item must be 50 characters or less")
			return
		}
	}
	if utf8.RuneCountInString(req.PersonalNotes) > 2000 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR",
			"Validation failed", "personalNotes exceeds maximum length of 2000 characters")
		return
	}

	if req.Condiments == nil {
		req.Condiments = []string{}
	}

	updated := user.Preferences{
		Condiments:    req.Condiments,
		PersonalNotes: req.PersonalNotes,
	}
	if err := h.userStore.UpdatePreferences(r.Context(), updated); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"Failed to update preferences", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, req)
}

// HealthHandler はヘルスチェックを返す
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func convertRecipes(aiRecipes []ai.Recipe) []Recipe {
	result := make([]Recipe, 0, len(aiRecipes))
	for _, r := range aiRecipes {
		ingredients := make([]Ingredient, 0, len(r.Ingredients))
		for _, ing := range r.Ingredients {
			ingredients = append(ingredients, Ingredient{
				Name:       ing.Name,
				Amount:     ing.Amount,
				IsInFridge: ing.IsInFridge,
			})
		}
		steps := make([]Step, 0, len(r.Steps))
		for _, s := range r.Steps {
			steps = append(steps, Step{Order: s.Order, Description: s.Description})
		}
		result = append(result, Recipe{
			Name:        r.Name,
			URL:         r.URL,
			Description: r.Description,
			MatchScore:  r.MatchScore,
			Ingredients: ingredients,
			Steps:       steps,
		})
	}
	return result
}

// --- ヘルパー ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message, details string) {
	writeJSON(w, status, ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}
