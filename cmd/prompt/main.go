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
	"github.com/xellio/gora/pkg/store"
)

var debugFlag = true
var indexName = "gora-doc"
var ollamaUserFacingModel = "gpt-oss:20b"
var ollamaURL = "http://127.0.0.1:11434"
var maxHistoryMessages = 10

func main() {
	ctx := context.Background()

	store, err := store.LoadStore(ctx, indexName)
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

	finalPrompt := fmt.Sprintf(`You are a highly skilled technical assistant for our API documentation.

### CONTEXTUAL GUIDELINES:
1. Use the CHAT HISTORY below to understand the context of the current question (e.g., resolving pronouns like "it", "this", or "that").
2. Use the PROVIDED DOCUMENTATION to find the factual answer.
3. If the answer is not contained within the documentation, state that you do not have enough information.
4. IMPORTANT: Always respond in the SAME LANGUAGE as the user's current question (e.g., if the user asks in German, answer in German).

### CHAT HISTORY:
%s

### RELEVANT DOCUMENTATION:
%s

### CURRENT USER QUESTION:
%s

### YOUR RESPONSE:`, history, fullContext, query)

	llm, err := ollama.New(
		ollama.WithModel(ollamaUserFacingModel),
		ollama.WithServerURL(ollamaURL),
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

	if len(messages) > maxHistoryMessages {
		messages = messages[len(messages)-maxHistoryMessages:]
	}

	// format conversation history
	for _, msg := range messages {
		converstation += msg.GetContent() + "\n"
	}

	return converstation, nil
}

func debugLog(message string) {
	if debugFlag {
		log.Println(message)
	}
}
