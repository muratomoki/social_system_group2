package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/social-system-group2/recipe-back/src/ai"
	"github.com/social-system-group2/recipe-back/src/api"
	"github.com/social-system-group2/recipe-back/src/config"
	"github.com/social-system-group2/recipe-back/src/fridge"
	"github.com/social-system-group2/recipe-back/src/user"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	fridgeClient := fridge.NewHTTPClient(cfg.FridgeAPIBaseURL)
	userStore := user.NewFileStore(cfg.UserPrefsFilePath)
	aiClient := ai.NewAnthropicClient(cfg.AnthropicAPIKey, cfg.ClaudeModel, cfg.CacheEnabled)

	recipeHandler := api.NewRecipeHandler(aiClient, fridgeClient, userStore)
	prefsHandler := api.NewPreferencesHandler(userStore)
	router := api.NewRouter(recipeHandler, prefsHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}
