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
	"github.com/charmbracelet/glamour"
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

type markdownRenderer interface {
	Render(string) (string, error)
}

func renderMarkdown(r markdownRenderer, input string) string {
	if r == nil {
		return input
	}
	rendered, err := r.Render(input)
	if err != nil {
		return input + fmt.Sprintf("\n(Markdown render error: %v)", err)
	}
	return rendered
}

func splitThink(content string) (string, string, bool) {
	re := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		think := strings.TrimSpace(matches[1])
		main := strings.TrimSpace(re.ReplaceAllString(content, ""))
		return think, main, true
	}
	return "", content, false
}

func renderResponseParts(content string, r markdownRenderer) (string, string, bool) {
	think, main, hasThink := splitThink(content)
	if hasThink {
		return renderMarkdown(r, think), renderMarkdown(r, main), true
	}
	return "", renderMarkdown(r, main), false
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

type model struct {
	viewport      viewport.Model
	textarea      textarea.Model
	senderStyle   lipgloss.Style
	aiStyle       lipgloss.Style
	thinkingStyle lipgloss.Style
	renderer      markdownRenderer
	err           error
	loop          *agent.Loop
	ctx           context.Context
}

func initialModel(ctx context.Context, loop *agent.Loop) model {
	renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		renderer = nil
	}

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
		renderer:      renderer,
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
		var viewContent string
		thinkRendered, mainRendered, hasThink := renderResponseParts(content, m.renderer)
		if hasThink {
			thinkRendered = indentLines(thinkRendered, "  ")
			viewContent = "\n\n" + m.thinkingStyle.Render("💭 Thinking:\n"+thinkRendered) +
				"\n\n" + m.aiStyle.Render("Golem: ") + mainRendered
		} else {
			viewContent = "\n\n" + m.aiStyle.Render("Golem: ") + mainRendered
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
	loop, err := agent.NewLoop(cfg, msgBus, modelProvider)
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}
	if err := loop.RegisterDefaultTools(cfg); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	if len(args) > 0 {
		message := strings.Join(args, " ")
		resp, err := loop.ProcessDirect(ctx, message)
		if err != nil {
			return err
		}
		fmt.Println(resp)
		return nil
	}

	p := tea.NewProgram(initialModel(ctx, loop))

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
