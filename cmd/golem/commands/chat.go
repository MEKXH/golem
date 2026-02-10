package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/provider"
	"github.com/MEKXH/golem/internal/render"
	"github.com/MEKXH/golem/internal/version"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const (
// No fixed width constant, we rely on updated width
)

var (
	// Styles
	userHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#8E4EC6")). // Purple
			Bold(true).
			Padding(0, 1)

	userBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8E4EC6")).
			Padding(0, 1)

	golemHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#2E8B57")). // SeaGreen
				Bold(true).
				Padding(0, 1)

	golemBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2E8B57")).
			Padding(0, 1)

	toolLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)
)

// Golem ASCII Art
const golemArt = `
   ______      __
  / ____/___  / /__  ____ ___
 / / __/ __ \/ / _ \/ __  __ \
/ /_/ / /_/ / /  __/ / / / / /
\____/\____/_/\___/_/ /_/ /_/
`

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

func renderResponseParts(content string, r markdownRenderer) (string, string, bool) {
	think, main, hasThink := render.SplitThink(content)
	if hasThink {
		// Render both thinking and main content with markdown formatting.
		return renderMarkdown(r, think), renderMarkdown(r, main), true
	}
	return "", renderMarkdown(r, main), false
}

// Data Structures for Structured History
type ToolLog struct {
	Name   string
	Result string
	Err    error
}

type ChatMessage struct {
	Role     string // "user", "golem"
	Content  string
	Thinking string
	Tools    []ToolLog
	IsError  bool
}

type model struct {
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	thinking bool

	// Styles
	senderStyle   lipgloss.Style
	aiStyle       lipgloss.Style
	thinkingStyle lipgloss.Style

	renderer markdownRenderer

	// Structured History
	messages      []ChatMessage
	currentHelper *ChatMessage // Tracks the message currently being generated

	loop *agent.Loop
	ctx  context.Context
	err  error

	width int // Track window width for re-rendering
}

func initialModel(ctx context.Context, loop *agent.Loop) model {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(30),
	)
	if err != nil {
		renderer = nil
	}

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 2000

	ta.SetWidth(30)
	ta.SetHeight(1)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	// Style the textarea
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")). // Purple-ish
		Padding(0, 1)

	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	vp := viewport.New(30, 5)

	// Initial Welcome Message

	welcomeMsg := ChatMessage{
		Role:    "system",
		Content: golemArt + fmt.Sprintf("\nWelcome to Golem Chat %s\nType a message and press Enter to send.", version.Version),
	}

	messages := []ChatMessage{welcomeMsg}

	ta.KeyMap.InsertNewline.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		textarea:      ta,
		viewport:      vp,
		spinner:       s,
		thinking:      false,
		senderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
		aiStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
		thinkingStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true),
		renderer:      renderer,
		messages:      messages,
		loop:          loop,
		ctx:           ctx,
		err:           nil,
		width:         30, // Default initial width
	}

	// Pre-render initial state
	m.viewport.SetContent(m.renderAll())

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick, tea.EnableMouseCellMotion)
}

type responseMsg string

type toolStartMsg struct {
	name string
	args string
}

type toolFinishMsg struct {
	name   string
	result string
	err    error
}

// renderAll re-renders the entire chat history based on current width
func (m model) renderAll() string {
	var sb strings.Builder

	for _, msg := range m.messages {
		sb.WriteString(m.renderMessage(msg))
		sb.WriteString("\n")
	}

	if m.currentHelper != nil {
		sb.WriteString(m.renderMessage(*m.currentHelper))
	}

	return sb.String()
}

func (m model) renderMessage(msg ChatMessage) string {
	// Adjust max width for content (Total Width - Padding - Border)
	// Approximate padding/border is 4 chars
	contentWidth := m.width - 6
	if contentWidth < 10 {
		contentWidth = 10
	}

	switch msg.Role {
	case "system":
		// Left aligned system message to preserve ASCII art alignment
		style := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("240"))
		return style.Render(msg.Content) + "\n"

	case "user":
		// Render content with a header inside
		header := userHeaderStyle.Render("USER")

		fullContent := header + "\n" + msg.Content

		body := userBodyStyle.
			Width(contentWidth).
			Render(fullContent)
		return fmt.Sprintf("\n%s", body)

	case "golem":
		// Golem Header
		header := golemHeaderStyle.Render("GOLEM")

		var contentBuilder strings.Builder
		contentBuilder.WriteString(header + "\n")

		// Render Tools
		if len(msg.Tools) > 0 {
			contentBuilder.WriteString("\n")
			for _, t := range msg.Tools {
				toolTxt := fmt.Sprintf("➢ %s", t.Name)
				if t.Err != nil {
					toolTxt += fmt.Sprintf(" ✖ %v", t.Err)
				} else {
					res := t.Result
					if len(res) > 50 {
						res = res[:50] + "..."
					}
					toolTxt += fmt.Sprintf(" ✔ %s", res)
				}
				contentBuilder.WriteString(toolLogStyle.Render(toolTxt) + "\n")
			}
		}

		// Render Thinking
		if msg.Thinking != "" {
			if len(msg.Tools) > 0 {
				contentBuilder.WriteString("\n")
			}

			// We render thinking as a distinct block
			// Use layout width - 2 (left/right padding of parent) for wrap
			thinkStyle := m.thinkingStyle.Width(contentWidth - 2)

			thinkTxt := "◉ Thinking:\n" + msg.Thinking
			contentBuilder.WriteString(thinkStyle.Render(thinkTxt) + "\n")

			// Add separator
			// Separator width should match content width minus a bit of safe margin to prevent wrapping
			sep := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(strings.Repeat("─", contentWidth-2))
			contentBuilder.WriteString(sep + "\n")
		} else if len(msg.Tools) > 0 {
			contentBuilder.WriteString("\n")
		}

		// Render Content (main)
		contentBuilder.WriteString(msg.Content)

		// Trim trailing newlines to prevent extra bottom padding in the bubble
		finalContent := strings.TrimRight(contentBuilder.String(), "\n")

		body := golemBodyStyle.
			Width(contentWidth).
			Render(finalContent)

		return fmt.Sprintf("\n%s", body)
	}
	return ""
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)

	// Handle mouse events before viewport update to override default scroll speed
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		switch mouseMsg.Type {
		case tea.MouseWheelUp:
			m.viewport.LineUp(3)
			return m, nil
		case tea.MouseWheelDown:
			m.viewport.LineDown(3)
			return m, nil
		}
	}

	m.viewport, vpCmd = m.viewport.Update(msg)

	// Calculate the available height and width
	// WindowHeight = Viewport + Processing + Textarea
	textareaHeight := 3

	// Processing indicator height
	processingHeight := 0
	if m.thinking {
		processingHeight = 1
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

		// Width calculation:
		// Subtract safety margin to avoid edge artifacts with CJK characters
		// WindowWidth - 4 gives buffer.
		availableWidth := msg.Width - 4
		if availableWidth < 20 {
			availableWidth = 20 // Minimum width
		}

		m.textarea.SetWidth(availableWidth)

		// Update viewport width
		m.viewport.Width = availableWidth

		// Calculate available height for viewport
		// WindowHeight - TextareaHeight - ProcessingHeight
		availableHeight := msg.Height - textareaHeight - processingHeight
		if availableHeight < 5 {
			availableHeight = 5 // Minimum height
		}
		m.viewport.Height = availableHeight

		// Update renderer width
		if m.renderer != nil {
			newRenderer, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle("dark"),
				glamour.WithWordWrap(availableWidth-6),
			)
			if err == nil {
				m.renderer = newRenderer
			}
		}

		// Re-render all messages with new width
		m.viewport.SetContent(m.renderAll())

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

			// Add User Message
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})

			// Initialize Golem Helper
			m.currentHelper = &ChatMessage{Role: "golem"}
			m.thinking = true

			// Render
			m.viewport.SetContent(m.renderAll())
			m.viewport.GotoBottom()

			// Reduce viewport height to make room for thinking indicator
			if m.viewport.Height > 5 {
				m.viewport.Height -= 1
			}

			return m, tea.Batch(
				m.spinner.Tick,
				func() tea.Msg {
					resp, err := m.loop.ProcessDirect(m.ctx, input)
					if err != nil {
						return errMsg(err)
					}
					return responseMsg(resp)
				},
			)

		}

	case responseMsg:
		// Restore viewport height
		if m.thinking {
			m.viewport.Height += 1
		}
		m.thinking = false

		// Finalize Golem Message
		if m.currentHelper != nil {
			content := string(msg)
			think, main, hasThink := renderResponseParts(content, m.renderer)

			if hasThink {
				// indentation handled by renderAll
				m.currentHelper.Thinking = think
			}
			m.currentHelper.Content = main

			m.messages = append(m.messages, *m.currentHelper)
			m.currentHelper = nil
		}

		m.viewport.SetContent(m.renderAll())
		m.viewport.GotoBottom()

	case toolStartMsg:
		if m.currentHelper == nil {
			m.currentHelper = &ChatMessage{Role: "golem"}
		}
		// Wait for finish to append log.
		m.viewport.SetContent(m.renderAll())
		m.viewport.GotoBottom()

	case toolFinishMsg:
		if m.currentHelper == nil {
			m.currentHelper = &ChatMessage{Role: "golem"}
		}
		m.currentHelper.Tools = append(m.currentHelper.Tools, ToolLog{
			Name:   msg.name,
			Result: msg.result,
			Err:    msg.err,
		})

		m.viewport.SetContent(m.renderAll())
		m.viewport.GotoBottom()

	case errMsg:
		if m.thinking {
			m.viewport.Height += 1
		}
		m.thinking = false

		if m.currentHelper != nil {
			m.currentHelper.Content = fmt.Sprintf("Error: %v", msg)
			m.currentHelper.IsError = true
			m.messages = append(m.messages, *m.currentHelper)
			m.currentHelper = nil
		} else {
			// Standalone error
			m.messages = append(m.messages, ChatMessage{Role: "golem", Content: fmt.Sprintf("System Error: %v", msg), IsError: true})
		}

		m.viewport.SetContent(m.renderAll())
		m.viewport.GotoBottom()

		m.err = msg
		return m, nil
	}

	if m.thinking {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m model) View() string {
	var processingView string
	if m.thinking {
		// When thinking, we show the spinner
		padding := strings.Repeat(" ", 2)
		processingView = fmt.Sprintf("%s%s Thinking...", padding, m.spinner.View())
		processingView = m.thinkingStyle.Render(processingView)
	}

	// Use JoinVertical for cleaner stacking
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		processingView,
		m.textarea.View(),
	)

	return content
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := configureLogger(cfg, logLevelOverride, true); err != nil {
		return fmt.Errorf("failed to configure logger: %w", err)
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

	p := tea.NewProgram(initialModel(ctx, loop), tea.WithAltScreen())

	// Set callbacks
	loop.OnToolStart = func(name, args string) {
		p.Send(toolStartMsg{name: name, args: args})
	}
	loop.OnToolFinish = func(name, result string, err error) {
		p.Send(toolFinishMsg{name: name, result: result, err: err})
	}

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
