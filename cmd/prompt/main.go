package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
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

	chatMemory := memory.NewConversationBuffer()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("GoRa started - Type 'quit' to exit")
	fmt.Println(strings.Repeat("-", 40))

	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" {
			break
		}

		response, err := generateFromSinglePrompt(ctx, store, chatMemory, input)
		if err != nil {
			log.Fatal(err)
		}

		//save input and response to memory
		chatMemory.ChatHistory.AddUserMessage(ctx, input)
		chatMemory.ChatHistory.AddAIMessage(ctx, response)

		fmt.Printf("GoRa: %v\n", response)
	}
}

func generateFromSinglePrompt(ctx context.Context, store *redisvector.Store, memory *memory.ConversationBuffer, query string) (string, error) {

	debugLog(fmt.Sprintf("Looking for: %s", query))

	results, err := store.SimilaritySearch(ctx, query, 3)
	if err != nil {
		debugLog(fmt.Sprintf("Search failed: %v", err))
	}

	var contextBuilder strings.Builder
	for i, doc := range results {
		fmt.Fprintf(&contextBuilder, "Document section %d:\n%s\n\n", i+1, doc.PageContent)
	}
	fullContext := contextBuilder.String()
	debugLog(fmt.Sprintf("Content passed to gpt: %s", fullContext))

	history, err := formatHistory(ctx, memory)
	if err != nil {
		debugLog(err.Error())
	}

	finalPrompt := fmt.Sprintf(`You are a professional technical support assistant.
Your goal is to provide accurate answers based EXCLUSIVELY on the provided documentation.

### STRICT OPERATING RULES:
1. USE the CHAT HISTORY only to resolve references (like "it", "this", or "that").
2. USE ONLY the facts found in the RELEVANT DOCUMENTATION.
3. DO NOT HALLUCINATE: Do not invent phone numbers, URLs, email addresses, or page numbers. If it's not in the text, it doesn't exist.
4. If the documentation does not contain the answer, explicitly state: "I'm sorry, but I don't have information about that in the current documentation."
5. DO NOT use your internal general knowledge to fill gaps.
6. LANGUAGE: Always respond in the SAME LANGUAGE as the user's current question.

### CHAT HISTORY:
%s

### RELEVANT DOCUMENTATION (ENGLISH):
%s

### CURRENT USER QUESTION:
%s

### YOUR RESPONSE:`, history, fullContext, query)

	llm, err := ollama.New(
		ollama.WithModel(cfg.Settings.OllamaModel),
		ollama.WithServerURL(cfg.Settings.OllamaURL),
	)
	if err != nil {
		log.Fatalf("Connection to ollama failed: %v", err)
	}

	response, err := llm.Call(ctx, finalPrompt)
	if err != nil {
		log.Fatalf("Unable to query model: %v", err)
	}

	return response, err

}

func formatHistory(ctx context.Context, memory *memory.ConversationBuffer) (string, error) {
	var converstation string
	messages, err := memory.ChatHistory.Messages(ctx)
	if err != nil {
		return "", err
	}

	if len(messages) > cfg.Settings.MaxHistoryMessages {
		messages = messages[len(messages)-cfg.Settings.MaxHistoryMessages:]
	}

	// format conversation history
	for _, msg := range messages {
		converstation += msg.GetContent() + "\n"
	}

	return converstation, nil
}

func debugLog(message string) {
	if cfg.Settings.Debug {
		log.Println(message)
	}
}
