package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/vectorstores/redisvector"
	"github.com/xellio/gora/pkg/config"
	"github.com/xellio/gora/pkg/store"
)

var cfg *config.Config

// UI styles
var (
	roleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	aiStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Margin(1)
)

type model struct {
	store      *redisvector.Store
	memory     *memory.ConversationBuffer
	cfg        *config.Config
	textInput  textinput.Model
	spinner    spinner.Model
	loading    bool
	lastResult string
	err        error
}

type responseMsg string
type errMsg error

func initialModel(s *redisvector.Store, m *memory.ConversationBuffer, c *config.Config) model {
	ti := textinput.New()
	ti.Placeholder = "Ask me something about the documentation..."
	ti.Focus()

	smp := spinner.New()
	smp.Spinner = spinner.Dot
	smp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		store:     s,
		memory:    m,
		cfg:       c,
		textInput: ti,
		spinner:   smp,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) askLLM(query string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := generateFromSinglePrompt(ctx, m.store, m.memory, query)
		if err != nil {
			return errMsg(err)
		}
		m.memory.ChatHistory.AddUserMessage(ctx, query)
		m.memory.ChatHistory.AddAIMessage(ctx, resp)

		return responseMsg(resp)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "quit":
			return m, tea.Quit
		case "enter":
			if m.textInput.Value() != "" && !m.loading {
				query := m.textInput.Value()
				m.loading = true
				m.textInput.Reset()
				return m, tea.Batch(m.spinner.Tick, m.askLLM(query))
			}
		}

	case responseMsg:
		m.loading = false
		m.lastResult = string(msg)
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("GoRa - Technical Documentation Assistant"))
	s.WriteString("\n\n")

	if m.lastResult != "" {
		s.WriteString(roleStyle.Render("GoRa: "))
		s.WriteString(aiStyle.Render(m.lastResult))
		s.WriteString("\n\n")
	}

	if m.loading {
		s.WriteString(m.spinner.View() + " Thinking...")
	} else {
		s.WriteString(m.textInput.View())
	}

	s.WriteString("\n\n(esc to quit)\n")

	return s.String()
}

func main() {
	// ... dein Config Load & Store Load wie gehabt ...
	cfg, _ = config.LoadConfig("config.yml")
	ctx := context.Background()
	s, _ := store.LoadStore(ctx, cfg)
	mem := memory.NewConversationBuffer()

	p := tea.NewProgram(initialModel(s, mem, cfg))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

/*
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
*/

func generateFromSinglePrompt(ctx context.Context, store *redisvector.Store, memory *memory.ConversationBuffer, query string) (string, error) {

	debugLog(fmt.Sprintf("Looking for: %s", query))

	results, err := store.SimilaritySearch(ctx, query, 3)
	if err != nil {
		debugLog(fmt.Sprintf("Search failed: %v", err))
	}

	var contextBuilder strings.Builder
	for i, doc := range results {
		//fmt.Println(doc.Score)
		fmt.Fprintf(&contextBuilder, "Document section %d:\n%s\n\n", i+1, doc.PageContent)
	}
	fullContext := contextBuilder.String()
	debugLog(fmt.Sprintf("Content passed to LLM: %s", fullContext))

	history, err := formatHistory(ctx, memory)
	if err != nil {
		debugLog(err.Error())
	}

	finalPrompt := fmt.Sprintf(`You are a professional technical support assistant.
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
		log.Printf("+++++++++++\n%s\n===========", message)
	}
}
