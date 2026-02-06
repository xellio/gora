package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/redisvector"
	"github.com/xellio/gora/pkg/config"
	"github.com/xellio/gora/pkg/store"
)

var cfg *config.Config

func main() {
	var err error
	cfg, err = config.LoadConfig("config.yml")
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
		log.Println("Populating database with document", document)
		err := populateDatabase(ctx, store, document)
		if err != nil {
			return err
		}
	}
	return nil
}

func findDataFiles(path string) ([]string, error) {
	var files []string
	log.Println("Reading directory", path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return files, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != ".gitkeep" && entry.Name() != ".DS_Store" {
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

	cleanedContent := cleanText(string(content))

	splitter := textsplitter.NewRecursiveCharacter()
	splitter.ChunkSize = cfg.Settings.RedisChunkSize
	splitter.ChunkOverlap = cfg.Settings.RedisChunkOverlap

	chunks, err := splitter.SplitText(cleanedContent)
	if err != nil {
		return err
	}

	docs := make([]schema.Document, 0, len(chunks))
	for _, chunk := range chunks {
		if len(strings.TrimSpace(chunk)) < 5 {
			continue
		}

		pageContent := chunk
		questions := generateSyntheticQuestions(ctx, chunk)
		if questions != "" {
			pageContent = fmt.Sprintf("Questions: %s\n\nContent: %s", questions, chunk)
		}

		docs = append(docs, schema.Document{
			PageContent: pageContent,
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

func generateSyntheticQuestions(ctx context.Context, input string) string {
	prompt := fmt.Sprintf(`Analyze the following technical documentation chunk and generate 3 short,
    diverse user questions that are answered by this text.
    Respond ONLY with the questions, separated by semicolons.

    TEXT:
    %s

    QUESTIONS:`, input)

	llm, err := ollama.New(
		ollama.WithModel(cfg.Settings.OllamaModel),
		ollama.WithServerURL(cfg.Settings.OllamaURL),
	)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	response, err := llm.Call(ctx, prompt)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	log.Printf("Generated questions: %s\n", response)
	return strings.TrimSpace(response)
}

func cleanText(input string) string {
	// replace double spaces with a single one
	reSpaces := regexp.MustCompile(`[ \t]{2,}`)
	input = reSpaces.ReplaceAllString(input, " ")

	// remove multiple newlines with a single one
	reNewlines := regexp.MustCompile(`\n{3,}`)
	input = reNewlines.ReplaceAllString(input, "\n\n")

	return strings.TrimSpace(input)
}
