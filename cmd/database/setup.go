package main

import (
	"context"
	"log"
	"os"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/redisvector"
	"github.com/xellio/gora/pkg/config"
	"github.com/xellio/gora/pkg/store"
)

var cfg *config.Config

func main() {

	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		if cfg == nil {
			log.Fatal(err)
		}
		log.Println("Using default configuration")
	}

	ctx := context.Background()

	store, err := store.LoadStore(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = setupDatabase(ctx, store)
	if err != nil {
		log.Fatal(err)
	}
}

func setupDatabase(ctx context.Context, store *redisvector.Store) error {
	files, err := findDataFiles(cfg.Settings.DataRootPath)
	if err != nil {
		return err
	}

	for _, document := range files {
		err := populateDatabase(ctx, store, document)
		if err != nil {
			return err
		}
	}
	return nil
}

func findDataFiles(path string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(path)
	if err != nil {
		return files, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != ".gitkeep" {
			files = append(files, path+"/"+entry.Name())
		}
	}
	return files, nil
}

func populateDatabase(ctx context.Context, store *redisvector.Store, documentPath string) error {

	content, err := os.ReadFile(documentPath)
	if err != nil {
		return err
	}

	splitter := textsplitter.NewRecursiveCharacter()
	splitter.ChunkSize = cfg.Settings.RedisChunkSize
	splitter.ChunkOverlap = cfg.Settings.RedisChunkOverlap

	chunks, err := splitter.SplitText(string(content))
	if err != nil {
		return err
	}

	docs := make([]schema.Document, 0, len(chunks))
	for _, chunk := range chunks {
		docs = append(docs, schema.Document{
			PageContent: chunk,
			Metadata: map[string]any{
				"source": documentPath,
			},
		})
	}

	_, err = store.AddDocuments(ctx, docs)
	if err != nil {
		return err
	}

	return nil
}
