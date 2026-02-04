package commands

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/provider"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func NewChatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat [message]",
		Short: "Chat with Golem",
		RunE:  runChat,
	}
}

type (
	errMsg error
)

type model struct {
	viewport      viewport.Model
	textarea      textarea.Model
	senderStyle   lipgloss.Style
	aiStyle       lipgloss.Style
	thinkingStyle lipgloss.Style
	err           error
	loop          *agent.Loop
	ctx           context.Context
}

func initialModel(ctx context.Context, loop *agent.Loop) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to Golem Chat!
Type a message and press Enter to send.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:      ta,
		viewport:      vp,
		senderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		aiStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		thinkingStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true),
		loop:          loop,
		ctx:           ctx,
		err:           nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

type responseMsg string

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - 2 // Adjust for padding/border
		m.textarea.SetWidth(msg.Width)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textarea.Value() == "" {
				return m, nil
			}
			input := m.textarea.Value()
			m.textarea.Reset()

			m.viewport.SetContent(m.viewport.View() + "\n\n" + m.senderStyle.Render("You: ") + input)
			m.viewport.GotoBottom()

			return m, func() tea.Msg {
				resp, err := m.loop.ProcessDirect(m.ctx, input)
				if err != nil {
					return errMsg(err)
				}
				return responseMsg(resp)
			}
		}

	case responseMsg:
		content := string(msg)
		re := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
		matches := re.FindStringSubmatch(content)

		var viewContent string
		if len(matches) > 1 {
			thinkContent := strings.TrimSpace(matches[1])
			mainContent := strings.TrimSpace(re.ReplaceAllString(content, ""))

			// Indent thinking content slightly for better visual separation
			thinkContent = strings.ReplaceAll(thinkContent, "\n", "\n  ")

			viewContent = "\n\n" + m.thinkingStyle.Render("💭 Thinking:\n  "+thinkContent) +
				"\n\n" + m.aiStyle.Render("Golem: ") + mainContent
		} else {
			viewContent = "\n\n" + m.aiStyle.Render("Golem: ") + content
		}

		m.viewport.SetContent(m.viewport.View() + viewContent)
		m.viewport.GotoBottom()

	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	modelProvider, err := provider.NewChatModel(ctx, cfg)
	if err != nil {
		fmt.Printf("Warning: %v\nRunning without LLM (tools only mode)\n", err)
		modelProvider = nil
	}

	msgBus := bus.NewMessageBus(10)
	loop := agent.NewLoop(cfg, msgBus, modelProvider)

	if err := loop.RegisterDefaultTools(cfg); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	p := tea.NewProgram(initialModel(ctx, loop))

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
