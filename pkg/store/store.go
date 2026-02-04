package store

import (
	"context"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/vectorstores/redisvector"
)

var ollamaModel = "nomic-embed-text"
var ollamaURL = "http://127.0.0.1:11434"
var redisURL = "redis://localhost:6379"

func LoadStore(ctx context.Context, indexName string) (*redisvector.Store, error) {
	llm, err := ollama.New(
		ollama.WithModel(ollamaModel),
		ollama.WithServerURL(ollamaURL),
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
		redisvector.WithConnectionURL(redisURL),
		redisvector.WithIndexName(indexName, true),
		redisvector.WithEmbedder(e),
	)
}
