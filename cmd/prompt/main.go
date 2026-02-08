package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
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
	s, err := store.LoadStore(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	chatMemory := memory.NewConversationBuffer()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\033[35mGoRa started - Type 'quit' to exit\033[0m")
	fmt.Println(strings.Repeat("-", 40))

	for {
		fmt.Print("\n\033[34mYou:\033[0m ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" {
			break
		}

		if input == "" {
			continue
		}

		response, err := generateAndStream(ctx, s, chatMemory, input)
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			continue
		}
		fmt.Println()

		chatMemory.ChatHistory.AddUserMessage(ctx, input)
		chatMemory.ChatHistory.AddAIMessage(ctx, response)
	}
}

func generateAndStream(ctx context.Context, store *redisvector.Store, mem *memory.ConversationBuffer, query string) (string, error) {
	prompt := preparePrompt(ctx, store, mem, query)

	llm, err := ollama.New(
		ollama.WithModel(cfg.Settings.OllamaModel),
		ollama.WithServerURL(cfg.Settings.OllamaURL),
	)
	if err != nil {
		return "", err
	}

	fmt.Print("\033[32mGoRa:\033[0m ")
	var fullResponse strings.Builder
	_, err = llm.Call(ctx, prompt,
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			content := string(chunk)
			fmt.Print(content)
			fullResponse.WriteString(content)
			return nil
		}),
	)
	return fullResponse.String(), err
}

func preparePrompt(ctx context.Context, store *redisvector.Store, memory *memory.ConversationBuffer, query string) string {

	debugLog(fmt.Sprintf("Looking for: %s", query))

	results, err := store.SimilaritySearch(ctx, query, 3)
	if err != nil {
		debugLog(fmt.Sprintf("Search failed: %v", err))
	}

	var contextBuilder strings.Builder
	for i, doc := range results {
		debugLog(fmt.Sprintf("Score: %f", doc.Score))
		fmt.Fprintf(&contextBuilder, "Document section %d:\n%s\n\n", i+1, doc.PageContent)
	}
	fullContext := contextBuilder.String()
	//debugLog(fmt.Sprintf("Content passed to LLM: %s", fullContext))

	history, err := formatHistory(ctx, memory)
	if err != nil {
		debugLog(err.Error())
	}

	return fmt.Sprintf(`You are a professional technical support assistant.
Your goal is to provide accurate answers based EXCLUSIVELY on the provided documentation.

### STRICT OPERATING RULES:
1. USE the CHAT HISTORY only to resolve references (like "it", "this", or "that").
2. USE ONLY the facts found in the "Content" sections of the RELEVANT DOCUMENTATION.
3. DO NOT HALLUCINATE: Do not invent phone numbers, URLs, email addresses, or page numbers. If it's not in the text, it doesn't exist.
4. If the documentation does not contain the answer, explicitly state: "I'm sorry, but I don't have information about that in the current documentation."
5. DO NOT use your internal general knowledge to fill gaps.
6. LANGUAGE: Always respond in the SAME LANGUAGE as the user's current question.
7. STRUCTURE NOTE: The documentation chunks may contain a "Questions" section followed by a "Content" section. Use the questions as context to understand the intent, but derive your answer ONLY from the "Content".

### CHAT HISTORY:
%s

### RELEVANT DOCUMENTATION (ENGLISH):
%s

### CURRENT USER QUESTION:
%s

### YOUR RESPONSE:`, history, fullContext, query)

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
		log.Println(strings.Repeat("+", 40))
		log.Println(message)
		log.Println(strings.Repeat("-", 40))
	}
}
