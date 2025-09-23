package coder

import (
	"context"
	"fmt"
	"lukatincode/function"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ccIisIaIcat/GoAgent/agent/general"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message types for Bubble Tea
type (
	tickMsg       struct{}
	aiResponseMsg struct {
		sender  string
		message string
		isError bool
	}
	toolCallMsg struct {
		toolName string
	}
	statusMsg struct {
		status string
	}
	exportMsg struct {
		filename string
		success  bool
	}
)

// BubbleTeaTUI represents the new TUI using Bubble Tea
type BubbleTeaTUI struct {
	lukatinCode *LukatinCode
	program     *tea.Program // Store program reference for sending messages

	// UI components
	input    textinput.Model
	viewport viewport.Model
	spinner  spinner.Model

	// State
	messages       []string
	todos          []TodoItem
	isProcessing   bool
	status         string
	showTodos      bool
	todoUpdateTime time.Time

	// Styles
	inputStyle   lipgloss.Style
	messageStyle lipgloss.Style
	todoStyle    lipgloss.Style
	statusStyle  lipgloss.Style

	// Layout
	width  int
	height int
}

type TodoItem struct {
	ID        int
	Content   string
	Status    string // pending, in_progress, completed
	Timestamp string
}

// NewBubbleTeaTUI creates a new Bubble Tea TUI
func NewBubbleTeaTUI(lukatinCode *LukatinCode) *BubbleTeaTUI {
	ti := textinput.New()
	ti.Placeholder = "输入消息..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80

	vp := viewport.New(80, 20)
	vp.SetContent("")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &BubbleTeaTUI{
		lukatinCode: lukatinCode,
		input:       ti,
		viewport:    vp,
		spinner:     s,
		messages:    []string{},
		todos:       []TodoItem{},
		status:      "就绪",
		showTodos:   false, // 默认隐藏TodoList

		// Styles
		inputStyle: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),

		messageStyle: lipgloss.NewStyle().
			Padding(0, 1),

		todoStyle: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")),

		statusStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true),
	}
}

// Init initializes the Bubble Tea program
func (b *BubbleTeaTUI) Init() tea.Cmd {
	b.lukatinCode.Logger.Println("Bubble Tea TUI 初始化")

	// Load initial todos
	b.refreshTodos()

	// Add welcome message
	b.addMessage("🚀 欢迎使用 LukatinCode!", "system")
	b.addMessage("💡 输入消息开始对话，输入 'exit' 退出", "system")
	b.addMessage("🔧 快捷键: Ctrl+S=导出对话历史, Ctrl+L=清空历史, Ctrl+C=退出", "system")

	return tea.Batch(
		textinput.Blink,
		b.spinner.Tick,
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		}),
	)
}

// Update handles messages and updates the model
func (b *BubbleTeaTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height

		// Update viewport size
		b.viewport.Width = msg.Width - 4
		b.viewport.Height = msg.Height - 5

		// Update input width
		b.input.Width = msg.Width - 4

		b.lukatinCode.Logger.Printf("窗口大小变化: %dx%d", msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			b.lukatinCode.Logger.Println("用户请求退出")
			return b, tea.Quit

		case "ctrl+s":
			// 导出对话历史到文件
			b.lukatinCode.Logger.Println("用户请求导出对话历史")
			go b.exportHistory()
			return b, nil

		case "ctrl+l":
			// 清空对话历史
			b.lukatinCode.Logger.Println("用户请求清空对话历史")
			b.messages = []string{}
			b.addMessage("🗑️ 对话历史已清空", "system")
			b.addMessage("💡 按 Ctrl+S 导出对话历史到文件", "system")
			return b, nil

		case "enter":
			input := strings.TrimSpace(b.input.Value())
			if input == "" {
				break
			}

			if input == "exit" || input == "quit" {
				b.lukatinCode.Logger.Println("用户通过命令退出")
				return b, tea.Quit
			}

			b.lukatinCode.Logger.Printf("用户输入: %s", input)
			b.addMessage(fmt.Sprintf("👤 %s", input), "user")
			b.input.SetValue("")
			b.isProcessing = true
			b.status = "AI正在思考..."

			// Process input asynchronously
			go b.processInput(input)

			return b, b.spinner.Tick
		}

	case aiResponseMsg:
		b.lukatinCode.Logger.Printf("收到AI回复: %s", msg.message)

		if msg.isError {
			b.addMessage(fmt.Sprintf("❌ %s", msg.message), "error")
		} else if msg.sender == "explanation" {
			// AI的解释性文本，用灰色显示
			b.addMessage(msg.message, "explanation")
		} else {
			b.addMessage(fmt.Sprintf("%s", msg.message), "assistant")
		}

		b.isProcessing = false
		b.status = "就绪"

	case toolCallMsg:
		b.lukatinCode.Logger.Printf("收到工具调用: %s", msg.toolName)

		if msg.toolName == "TodoRead" || msg.toolName == "TodoWrite" {
			b.addMessage("🔧 TodoList 管理", "tool")
			// 直接在聊天流中显示格式化的TodoList
			todoData := function.ListTodosFormatted()
			b.addMessage(todoData, "todolist")
		} else {
			b.addMessage(fmt.Sprintf("🔧 %s", msg.toolName), "tool")
		}

	case statusMsg:
		b.lukatinCode.Logger.Printf("状态更新: %s", msg.status)
		b.status = msg.status

	case exportMsg:
		if msg.success {
			b.addMessage(fmt.Sprintf("✅ 对话历史已导出到: %s", msg.filename), "system")
		} else {
			b.addMessage("❌ 导出失败，请查看日志", "error")
		}

	case spinner.TickMsg:
		if b.isProcessing {
			var cmd tea.Cmd
			b.spinner, cmd = b.spinner.Update(msg)
			return b, cmd
		}

	case tickMsg:
		// 继续发送tick消息
		return b, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		})
	}

	// Update input
	var cmd tea.Cmd
	b.input, cmd = b.input.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport
	b.viewport, cmd = b.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return b, tea.Batch(cmds...)
}

// View renders the UI
func (b *BubbleTeaTUI) View() string {
	// Main content area
	content := b.viewport.View()

	// Input area
	inputSection := b.inputStyle.Render(b.input.View())

	// Status line
	statusLine := b.renderStatus()

	// Simple vertical layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		inputSection,
		statusLine,
	)
}

// addMessage adds a message to the chat history
func (b *BubbleTeaTUI) addMessage(message, msgType string) {
	timestamp := time.Now().Format("15:04:05")

	// 自动换行处理，考虑viewport宽度
	maxWidth := b.viewport.Width - 20 // 留一些边距
	if maxWidth < 50 {
		maxWidth = 50 // 最小宽度
	}

	var styledMessage string
	switch msgType {
	case "user":
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // 蓝色
			Render(wrappedMsg)
	case "assistant":
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")). // 白色
			Render(wrappedMsg)
	case "tool":
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Render(wrappedMsg)
	case "explanation":
		wrappedMsg := b.wrapText(fmt.Sprintf("  %s", message), maxWidth) // 缩进显示
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // 灰色
			Render(wrappedMsg)
	case "todolist":
		// 处理TodoList的特殊格式化
		styledMessage = b.formatTodoListMessage(message)
	case "error":
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("31")).
			Render(wrappedMsg)
	case "system":
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render(wrappedMsg)
	default:
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = wrappedMsg
	}

	b.messages = append(b.messages, styledMessage, "") // 每条消息后添加空行

	// Update viewport content
	content := strings.Join(b.messages, "\n")
	b.viewport.SetContent(content)
	b.viewport.GotoBottom()
}

// formatTodoListMessage 格式化TodoList消息，对不同状态的任务应用不同样式
func (b *BubbleTeaTUI) formatTodoListMessage(message string) string {
	lines := strings.Split(message, "\n")
	var formattedLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			formattedLines = append(formattedLines, "")
			continue
		}

		// 检查是否包含正在执行的任务标记
		if strings.Contains(line, "▶") {
			// 正在执行的任务用绿色高亮
			styledLine := lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")). // 亮绿色
				Bold(true).
				Render(line)
			formattedLines = append(formattedLines, styledLine)
		} else if strings.HasPrefix(strings.TrimSpace(line), "☑") {
			// 已完成的任务保持正常白色
			styledLine := lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")). // 白色
				Render(line)
			formattedLines = append(formattedLines, styledLine)
		} else if strings.HasPrefix(strings.TrimSpace(line), "☐") {
			// 待处理的任务用白色
			styledLine := lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")). // 白色
				Render(line)
			formattedLines = append(formattedLines, styledLine)
		} else {
			// 统计信息等其他行用灰色
			styledLine := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")). // 灰色
				Render(line)
			formattedLines = append(formattedLines, styledLine)
		}
	}

	// 整体添加背景和边距
	finalMessage := strings.Join(formattedLines, "\n")
	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Padding(1).
		Render(finalMessage)
}

// renderTodos renders the todo list with checkboxes
func (b *BubbleTeaTUI) renderTodos() string {
	if len(b.todos) == 0 {
		return b.todoStyle.Render("📝 TodoList\n\n暂无任务")
	}

	// 直接显示TodoList原始数据
	for _, todo := range b.todos {
		if todo.Status == "display" {
			return b.todoStyle.Render(todo.Content)
		}
	}

	return b.todoStyle.Render("📝 TodoList\n\n数据加载中...")
}

// renderStatus renders the status line
func (b *BubbleTeaTUI) renderStatus() string {
	if b.isProcessing {
		return b.statusStyle.Render(
			fmt.Sprintf("%s %s", b.spinner.View(), b.status),
		)
	}

	return b.statusStyle.Render(fmt.Sprintf("⚡ %s", b.status))
}

// refreshTodos updates the todo list from the function
func (b *BubbleTeaTUI) refreshTodos() {
	b.lukatinCode.Logger.Println("刷新TodoList")

	// 获取最新的格式化TodoList数据
	todoData := function.ListTodosFormatted()
	b.lukatinCode.Logger.Printf("TodoList数据: %s", todoData)

	// 简化解析：直接将原始数据存储显示
	b.todos = []TodoItem{
		{
			ID:      1,
			Content: todoData,
			Status:  "display",
		},
	}
	b.lukatinCode.Logger.Println("TodoList已更新")
}

// processInput handles user input asynchronously
func (b *BubbleTeaTUI) processInput(input string) {
	b.lukatinCode.Logger.Printf("开始处理输入: %s", input)

	// Create a modified version of ChatWithTUI that sends results back to Bubble Tea
	b.chatWithBubbleTea(input)
}

// chatWithBubbleTea is a modified version of ChatWithTUI for Bubble Tea
func (b *BubbleTeaTUI) chatWithBubbleTea(input string) {
	b.lukatinCode.Logger.Printf("=================== 开始处理用户输入 ===================")
	b.lukatinCode.Logger.Printf("用户输入: %s", input)

	if b.program != nil {
		b.program.Send(statusMsg{status: "AI正在思考..."})
	}

	info_chan := make(chan general.Message, 10)
	var messages []string
	var toolCalls []string
	var wg sync.WaitGroup
	wg.Add(1)

	b.lukatinCode.Logger.Println("创建消息处理goroutine")
	// 处理消息
	go func() {
		defer wg.Done()
		b.lukatinCode.Logger.Println("开始处理来自AI的消息")
		messageCount := 0
		for msg := range info_chan {
			messageCount++
			if msg.Role == general.RoleAssistant {
				// 处理工具调用
				if len(msg.ToolCalls) > 0 {
					for i, toolCall := range msg.ToolCalls {
						b.lukatinCode.Logger.Printf("处理工具调用%d: %s", i+1, toolCall.Function.Name)
						toolCalls = append(toolCalls, toolCall.Function.Name)
						b.lukatinCode.Logger.Println("工具调用时的其他细节：", toolCall)
						// Send tool call message to UI
						if b.program != nil {
							b.program.Send(toolCallMsg{toolName: toolCall.Function.Name})
						}
					}
				}

				// 处理文本内容 - 这些是AI的解释性文本
				for i, content := range msg.Content {
					b.lukatinCode.Logger.Printf("处理内容%d: Type=%s, Text长度=%d",
						i+1, content.Type, len(content.Text))
					if content.Type == general.ContentTypeText && content.Text != "" {
						// 如果有工具调用，这些文本是解释性的，用灰色显示
						if len(msg.ToolCalls) > 0 {
							if b.program != nil {
								b.program.Send(aiResponseMsg{
									sender:  "explanation",
									message: content.Text,
									isError: false,
								})
							}
						} else {
							// 普通AI回复
							messages = append(messages, content.Text)
							b.lukatinCode.Logger.Printf("添加文本消息: %s", content.Text)
						}
					}
				}
			}
		}
		b.lukatinCode.Logger.Printf("消息处理goroutine结束, 共处理%d条消息", messageCount)
	}()

	// 调用AI
	b.lukatinCode.Logger.Println("开始调用AI Chat方法")
	start := time.Now()
	model := b.lukatinCode.Lmmconfig.AgentAPIKey.OpenAI.Model
	_, _, err, _ := b.lukatinCode.CM.Chat(context.Background(), general.ProviderOpenAI, model, input, []string{}, info_chan)
	duration := time.Since(start)
	b.lukatinCode.Logger.Printf("AI Chat调用完成, 耗时: %v", duration)

	close(info_chan)
	b.lukatinCode.Logger.Println("关闭info_chan")

	// 等待消息处理完成
	b.lukatinCode.Logger.Println("等待消息处理goroutine完成")
	wg.Wait()
	b.lukatinCode.Logger.Println("消息处理完成")

	if err != nil {
		b.lukatinCode.Logger.Printf("AI调用出错: %v", err)
		if b.program != nil {
			b.program.Send(aiResponseMsg{
				sender:  "assistant",
				message: fmt.Sprintf("处理失败: %v", err),
				isError: true,
			})
		}
	} else {
		// 显示AI回复
		b.lukatinCode.Logger.Printf("准备显示AI回复, 消息数量: %d, 工具调用数量: %d", len(messages), len(toolCalls))
		if len(messages) > 0 {
			response := strings.Join(messages, "\n")
			b.lukatinCode.Logger.Printf("显示AI回复: %s", response)
			if b.program != nil {
				b.program.Send(aiResponseMsg{
					sender:  "assistant",
					message: response,
					isError: false,
				})
			}
		} else {
			b.lukatinCode.Logger.Println("没有收到AI文本回复，显示默认消息")
			if b.program != nil {
				b.program.Send(aiResponseMsg{
					sender:  "assistant",
					message: "已处理完成",
					isError: false,
				})
			}
		}
	}

	b.lukatinCode.Logger.Println("更新状态为: 就绪")
	if b.program != nil {
		b.program.Send(statusMsg{status: "就绪"})
	}
	b.lukatinCode.Logger.Printf("=================== 用户输入处理完成 ===================")
}

// Run starts the Bubble Tea TUI
func (b *BubbleTeaTUI) Run() error {
	b.lukatinCode.Logger.Println("启动 Bubble Tea TUI")

	p := tea.NewProgram(
		b,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Store program reference for sending messages
	b.program = p

	err := p.Start()
	if b.lukatinCode.Logger != nil {
		if err != nil {
			b.lukatinCode.Logger.Printf("Bubble Tea TUI运行结束，错误: %v", err)
		} else {
			b.lukatinCode.Logger.Println("Bubble Tea TUI正常结束")
		}
	}
	return err
}

// wrapText wraps text to fit within specified width
func (b *BubbleTeaTUI) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			lines = append(lines, line)
			continue
		}

		// 按字符数换行，优先在空格处断开
		words := strings.Fields(line)
		if len(words) == 0 {
			lines = append(lines, line)
			continue
		}

		currentLine := ""
		for _, word := range words {
			// 如果单个词太长，强制截断
			if len(word) > width {
				if currentLine != "" {
					lines = append(lines, currentLine)
					currentLine = ""
				}
				// 分割长词
				for len(word) > width {
					lines = append(lines, word[:width])
					word = word[width:]
				}
				if word != "" {
					currentLine = word
				}
				continue
			}

			// 检查添加这个词是否会超过宽度
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word

			if len(testLine) <= width {
				currentLine = testLine
			} else {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return strings.Join(lines, "\n")
}

// exportHistory exports the conversation history to a file
func (b *BubbleTeaTUI) exportHistory() {
	filename := fmt.Sprintf("conversation_%s.txt", time.Now().Format("20060102_150405"))

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		b.lukatinCode.Logger.Printf("获取工作目录失败: %v", err)
		if b.program != nil {
			b.program.Send(exportMsg{filename: filename, success: false})
		}
		return
	}

	absolutePath := filepath.Join(currentDir, "log", filename)

	b.lukatinCode.Logger.Printf("开始导出对话历史到: %s", absolutePath)

	content := "=== LukatinCode 对话历史 ===\n"
	content += fmt.Sprintf("导出时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("总消息数: %d\n", len(b.messages))
	content += "========================\n\n"

	for i, msg := range b.messages {
		content += fmt.Sprintf("[%03d] %s\n", i+1, msg)
	}

	content += "\n========================\n"
	content += "💡 提示: 这个文件包含了完整的对话历史，你可以复制其中的任何内容\n"
	content += "🔧 快捷键: Ctrl+S=导出历史, Ctrl+L=清空历史, Ctrl+C=退出\n"

	result := function.Write(absolutePath, content)
	if result != "" && !strings.Contains(result, "Successfully") {
		b.lukatinCode.Logger.Printf("导出失败: %s", result)
		if b.program != nil {
			b.program.Send(exportMsg{filename: filename, success: false})
		}
	} else {
		b.lukatinCode.Logger.Printf("导出成功: %s", absolutePath)
		if b.program != nil {
			b.program.Send(exportMsg{filename: filename, success: true})
		}
	}
}
