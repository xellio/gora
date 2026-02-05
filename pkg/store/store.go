package store

import (
	"context"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/vectorstores/redisvector"
	"github.com/xellio/gora/pkg/config"
)

func LoadStore(ctx context.Context, cfg *config.Config) (*redisvector.Store, error) {
	llm, err := ollama.New(
		ollama.WithModel(cfg.Settings.OllamaModelEmbed),
		ollama.WithServerURL(cfg.Settings.OllamaURL),
	)
	if err != nil {
		return nil, err
	}

	e, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, err
	}

	return redisvector.New(
		ctx,
		redisvector.WithConnectionURL(cfg.Settings.RedisURL),
		redisvector.WithIndexName(cfg.Settings.RedisIndexName, true),
		redisvector.WithEmbedder(e),
	)
}
