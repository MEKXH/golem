package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/metrics"
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
// 无固定宽度常量，依赖于实时更新的窗口宽度
)

var (
	// 样式定义
	userHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#8E4EC6")). // 紫色
			Bold(true).
			Padding(0, 1)

	userBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8E4EC6")).
			Padding(0, 1)

	golemHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#2E8B57")). // 海洋绿
				Bold(true).
				Padding(0, 1)

	golemBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2E8B57")).
			Padding(0, 1)

	toolLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("240")).
			Padding(0, 1).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(1).
			PaddingRight(2)
)

// Golem ASCII 艺术字
const golemArt = `
   ______      __
  / ____/___  / /__  ____ ___
 / / __/ __ \/ / _ \/ __  __ \
/ /_/ / /_/ / /  __/ / / / / /
\____/\____/_/\___/_/ /_/ /_/
`

// NewChatCmd 创建交互式聊天命令。
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
		// 同时使用 Markdown 格式渲染思考过程和主内容
		return renderMarkdown(r, think), renderMarkdown(r, main), true
	}
	return "", renderMarkdown(r, main), false
}

// 结构化历史记录的数据结构
type ToolLog struct {
	Name   string
	Result string
	Err    error
}

type ChatMessage struct {
	Role     string // "user", "golem", "system"
	Content  string
	Thinking string
	Tools    []ToolLog
	IsError  bool

	// 缓存已渲染的内容
	renderedContent      string
	renderedWidth        int
	lastRenderedContent  string
	lastRenderedThinking string
	lastRenderedToolsLen int
}

type model struct {
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	thinking bool

	// 样式
	senderStyle   lipgloss.Style
	aiStyle       lipgloss.Style
	thinkingStyle lipgloss.Style

	renderer markdownRenderer

	// 消息历史
	messages      []ChatMessage
	currentHelper *ChatMessage // 追踪当前正在生成的消息

	loop *agent.Loop
	ctx  context.Context
	err  error

	currentTool string // 追踪当前正在运行的工具

	width int // 窗口宽度，用于重新渲染
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

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")). // 偏紫色
		Padding(0, 1)

	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	vp := viewport.New(30, 5)

	// 初始欢迎消息
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
		width:         30,
	}

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

// renderAll 根据当前宽度重新渲染整个聊天历史
func (m model) renderAll() string {
	var sb strings.Builder

	for i := range m.messages {
		sb.WriteString(m.renderMessage(&m.messages[i]))
		sb.WriteString("\n")
	}

	if m.currentHelper != nil {
		sb.WriteString(m.renderMessage(m.currentHelper))
	}

	return sb.String()
}

func (m model) renderMessage(msg *ChatMessage) string {
	contentWidth := m.width - 6
	if contentWidth < 10 {
		contentWidth = 10
	}

	// 检查缓存
	if msg.renderedContent != "" &&
		msg.renderedWidth == m.width &&
		msg.Content == msg.lastRenderedContent &&
		msg.Thinking == msg.lastRenderedThinking &&
		len(msg.Tools) == msg.lastRenderedToolsLen {
		return msg.renderedContent
	}

	var result string

	switch msg.Role {
	case "system":
		style := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("240"))
		result = style.Render(msg.Content) + "\n"

	case "user":
		header := userHeaderStyle.Render("USER")
		fullContent := header + "\n" + msg.Content
		body := userBodyStyle.
			Width(contentWidth).
			Render(fullContent)
		result = fmt.Sprintf("\n%s", body)

	case "golem":
		headerStyle := golemHeaderStyle
		bodyStyle := golemBodyStyle
		headerLabel := "GOLEM"

		if msg.IsError {
			headerStyle = headerStyle.Copy().Background(lipgloss.Color("#FF0000"))
			bodyStyle = bodyStyle.Copy().BorderForeground(lipgloss.Color("#FF0000"))
			headerLabel = "ERROR"
		}

		header := headerStyle.Render(headerLabel)

		var contentBuilder strings.Builder
		contentBuilder.WriteString(header + "\n")

		// 渲染工具执行日志
		if len(msg.Tools) > 0 {
			contentBuilder.WriteString("\n")
			for _, t := range msg.Tools {
				contentBuilder.WriteString(toolLogStyle.Render(fmt.Sprintf("➢ %s", t.Name)))

				if t.Err != nil {
					errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
					contentBuilder.WriteString(errStyle.Render(" ✖ "))
					contentBuilder.WriteString(errStyle.Italic(true).Render(fmt.Sprintf("%v", t.Err)))
				} else {
					res := t.Result
					if len(res) > 200 {
						res = res[:200] + "..."
					}
					successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2E8B57"))
					contentBuilder.WriteString(successStyle.Render(" ✔ "))
					contentBuilder.WriteString(successStyle.Italic(true).Render(res))
				}
				contentBuilder.WriteString("\n")
			}
		}

		// 渲染思考过程
		if msg.Thinking != "" {
			if len(msg.Tools) > 0 {
				contentBuilder.WriteString("\n")
			}

			thinkStyle := m.thinkingStyle.Width(contentWidth - 2)
			thinkTxt := "◉ Thinking:\n" + msg.Thinking
			contentBuilder.WriteString(thinkStyle.Render(thinkTxt) + "\n")

			// 添加分隔线
			sep := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(strings.Repeat("─", contentWidth-2))
			contentBuilder.WriteString(sep + "\n")
		} else if len(msg.Tools) > 0 {
			contentBuilder.WriteString("\n")
		}

		// 渲染主回答内容
		contentBuilder.WriteString(msg.Content)

		finalContent := strings.TrimRight(contentBuilder.String(), "\n")
		body := bodyStyle.
			Width(contentWidth).
			Render(finalContent)

		result = fmt.Sprintf("\n%s", body)
	}

	msg.renderedContent = result
	msg.renderedWidth = m.width
	msg.lastRenderedContent = msg.Content
	msg.lastRenderedThinking = msg.Thinking
	msg.lastRenderedToolsLen = len(msg.Tools)
	return result
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)

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

	textareaHeight := 3
	processingHeight := 0
	if m.thinking {
		processingHeight = 1
	}
	helpHeight := 1

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		availableWidth := msg.Width - 4
		if availableWidth < 20 {
			availableWidth = 20
		}

		m.textarea.SetWidth(availableWidth)
		m.viewport.Width = availableWidth

		availableHeight := msg.Height - textareaHeight - processingHeight - helpHeight
		if availableHeight < 5 {
			availableHeight = 5
		}
		m.viewport.Height = availableHeight

		if m.renderer != nil {
			newRenderer, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle("dark"),
				glamour.WithWordWrap(availableWidth-6),
			)
			if err == nil {
				m.renderer = newRenderer
			}
		}

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

			// 处理内置斜杠命令：/new
			if strings.TrimSpace(input) == "/new" {
				m.messages = nil
				m.currentHelper = &ChatMessage{Role: "golem"}
				m.thinking = true
				m.viewport.SetContent(m.renderAll())
				m.viewport.GotoBottom()
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

			// 添加用户消息
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})

			// 初始化 Golem 助手生成状态
			m.currentHelper = &ChatMessage{Role: "golem"}
			m.thinking = true

			m.viewport.SetContent(m.renderAll())
			m.viewport.GotoBottom()

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
		m.currentTool = ""
		if m.thinking {
			m.viewport.Height += 1
		}
		m.thinking = false

		if m.currentHelper != nil {
			content := string(msg)
			think, main, hasThink := renderResponseParts(content, m.renderer)

			if hasThink {
				m.currentHelper.Thinking = think
			}
			m.currentHelper.Content = main

			m.messages = append(m.messages, *m.currentHelper)
			m.currentHelper = nil
		}

		m.viewport.SetContent(m.renderAll())
		m.viewport.GotoBottom()

	case toolStartMsg:
		m.currentTool = msg.name
		if m.currentHelper == nil {
			m.currentHelper = &ChatMessage{Role: "golem"}
		}
		m.viewport.SetContent(m.renderAll())
		m.viewport.GotoBottom()

	case toolFinishMsg:
		m.currentTool = ""
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
		padding := strings.Repeat(" ", 2)
		label := "Thinking..."
		if m.currentTool != "" {
			label = fmt.Sprintf("Running tool: %s...", m.currentTool)
		}
		processingView = fmt.Sprintf("%s%s %s", padding, m.spinner.View(), label)
		processingView = m.thinkingStyle.Render(processingView)
	}

	separator := descStyle.Render(" • ")
	helpView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		keyStyle.Render("Enter"),
		descStyle.Render("Send"),
		separator,
		keyStyle.Render("/new"),
		descStyle.Render("Reset"),
		separator,
		keyStyle.Render("PgUp/Dn"),
		descStyle.Render("Scroll"),
		separator,
		keyStyle.Render("Esc/Ctrl+C"),
		descStyle.Render("Quit"),
	)

	if !m.viewport.AtBottom() {
		scrollIndicator := keyStyle.Copy().Background(lipgloss.Color("172")).Render("↓ Scrolled Up")
		helpView = lipgloss.JoinHorizontal(lipgloss.Top, helpView, scrollIndicator)
	}

	helpView = helpStyle.Render(helpView)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		processingView,
		m.textarea.View(),
		helpView,
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
	workspacePath, err := cfg.WorkspacePathChecked()
	if err != nil {
		return fmt.Errorf("invalid workspace: %w", err)
	}
	runtimeMetrics := metrics.NewRuntimeMetrics(workspacePath)
	defer runtimeMetrics.Close()
	loop.SetRuntimeMetrics(runtimeMetrics)
	logAndAuditRuntimePolicyStartup(ctx, loop, cfg)

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

	// 设置回调，将工具执行状态同步到 TUI
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
