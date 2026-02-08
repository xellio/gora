package store

import (
	"context"
	"fmt"

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

	indexName := cfg.Settings.RedisIndexName
	if cfg.Settings.AppendEmbedModelNameToIndex {
		indexName = fmt.Sprintf("%s_%s", indexName, cfg.Settings.OllamaModelEmbed)
	}

	return redisvector.New(
		ctx,
		redisvector.WithConnectionURL(cfg.Settings.RedisURL),
		redisvector.WithIndexName(indexName, true),
		redisvector.WithEmbedder(e),
	)
}
