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
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 确认选择项
type confirmOption struct {
	title string
	desc  string
	value bool
}

func (c confirmOption) Title() string       { return c.title }
func (c confirmOption) Description() string { return c.desc }
func (c confirmOption) FilterValue() string { return c.title }

// Message types for Bubble Tea
type (
	tickMsg       struct{}
	aiResponseMsg struct {
		sender  string
		message string
		isError bool
	}
	toolCallMsg struct {
		toolCalls []toolCallInfo
	}
	toolCallInfo struct {
		toolName string
		params   string
	}
	statusMsg struct {
		status string
	}
	exportMsg struct {
		filename string
		success  bool
	}
	toolDescriptionMsg struct {
		description string
	}
	codeChangeMsg struct {
		filePath    string
		oldContent  string
		newContent  string
		operation   string // "edit", "multiedit", "write"
		needConfirm bool
		changeId    string
	}
	userConfirmMsg struct {
		changeId string
		approved bool
	}
)

// BubbleTeaTUI represents the new TUI using Bubble Tea
type BubbleTeaTUI struct {
	lukatinCode *LukatinCode
	program     *tea.Program // Store program reference for sending messages

	// UI components
	input         textinput.Model
	viewport      viewport.Model
	spinner       spinner.Model
	confirmList   list.Model

	// State
	messages       []string
	todos          []TodoItem
	isProcessing   bool
	status         string
	showTodos      bool
	todoUpdateTime time.Time
	
	// Code change confirmation
	pendingChanges   map[string]codeChangeMsg
	responseChannels map[string]chan bool
	waitingForConfirm bool
	currentChangeId   string
	uiMode           string // "normal" 或 "confirm"

	// Styles
	inputStyle     lipgloss.Style
	messageStyle   lipgloss.Style
	todoStyle      lipgloss.Style
	statusStyle    lipgloss.Style
	diffAddedStyle lipgloss.Style
	diffRemovedStyle lipgloss.Style
	diffHeaderStyle lipgloss.Style

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

	b := &BubbleTeaTUI{
		lukatinCode: lukatinCode,
		input:       ti,
		viewport:    vp,
		spinner:     s,
		messages:    []string{},
		todos:       []TodoItem{},
		status:      "就绪",
		showTodos:   false, // 默认隐藏TodoList
		
		// Code change confirmation
		pendingChanges:    make(map[string]codeChangeMsg),
		responseChannels:  make(map[string]chan bool),
		waitingForConfirm: false,
		currentChangeId:   "",
		uiMode:           "normal",

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

		diffAddedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Background(lipgloss.Color("22")),

		diffRemovedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Background(lipgloss.Color("52")),

		diffHeaderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true),
	}

	// 初始化确认列表
	options := []list.Item{
		confirmOption{
			title: "✅ 确认执行修改",
			desc:  "继续执行代码修改操作",
			value: true,
		},
		confirmOption{
			title: "❌ 取消修改",
			desc:  "停止AI任务并等待新指令",
			value: false,
		},
	}
	
	b.confirmList = list.New(options, list.NewDefaultDelegate(), 50, 10)
	b.confirmList.Title = "请选择操作"
	b.confirmList.SetShowStatusBar(false)
	b.confirmList.SetFilteringEnabled(false)
	b.confirmList.SetShowHelp(false)

	return b
}

// Init initializes the Bubble Tea program
func (b *BubbleTeaTUI) Init() tea.Cmd {
	b.lukatinCode.Logger.Println("Bubble Tea TUI 初始化")

	// Load initial todos
	b.refreshTodos()

	// Add welcome message
	b.addMessage("🚀 欢迎使用 LukatinCode!", "system")
	b.addMessage("💡 输入消息开始对话，输入 'exit' 退出", "system")
	b.addMessage("🔧 快捷键: ESC=取消AI任务, Ctrl+S=导出历史, Ctrl+L=清空历史, Ctrl+C=退出", "system")

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
		case "ctrl+c":
			b.lukatinCode.Logger.Println("用户请求退出")
			return b, tea.Quit

		case "esc":
			b.lukatinCode.Logger.Println("用户按下ESC键")
			if b.lukatinCode != nil {
				b.lukatinCode.CancelCurrentTask()
			}
			return b, nil

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
			// 处理代码修改确认
			if b.uiMode == "confirm" && b.currentChangeId != "" {
				selectedItem := b.confirmList.SelectedItem().(confirmOption)
				return b, func() tea.Msg {
					return userConfirmMsg{
						changeId: b.currentChangeId,
						approved: selectedItem.value,
					}
				}
			}

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
		// 构建合并的工具调用显示
		var toolCallsDisplay []string
		var allParams []string
		hasTodoCall := false

		for _, toolCall := range msg.toolCalls {
			b.lukatinCode.Logger.Printf("收到工具调用: %s", toolCall.toolName)
			
			if toolCall.toolName == "TodoRead" || toolCall.toolName == "TodoWrite" {
				hasTodoCall = true
			}
			
			toolCallsDisplay = append(toolCallsDisplay, toolCall.toolName)
			if toolCall.params != "" {
				allParams = append(allParams, fmt.Sprintf("%s(%s)", toolCall.toolName, toolCall.params))
			}
		}

		// 如果包含TodoList操作，单独处理
		if hasTodoCall {
			b.addMessage("🔧 TodoList 管理", "tool")
			todoData := function.ListTodosFormatted()
			b.addMessage(todoData, "todolist")
		}

		// 构建工具调用的合并显示
		if len(toolCallsDisplay) > 0 {
			toolCallText := strings.Join(toolCallsDisplay, " + ")
			
			if len(allParams) > 0 {
				paramsText := strings.Join(allParams, " | ")
				b.addMessageWithParams(fmt.Sprintf("🔧 %s", toolCallText), paramsText, "tool_with_params")
			} else {
				b.addMessage(fmt.Sprintf("🔧 %s", toolCallText), "tool")
			}
		}

	case statusMsg:
		b.lukatinCode.Logger.Printf("状态更新: %s", msg.status)
		b.status = msg.status

	case toolDescriptionMsg:
		b.lukatinCode.Logger.Printf("收到工具描述: %s", msg.description)
		b.addMessage(msg.description, "tool_description")

	case exportMsg:
		if msg.success {
			b.addMessage(fmt.Sprintf("✅ 对话历史已导出到: %s", msg.filename), "system")
		} else {
			b.addMessage("❌ 导出失败，请查看日志", "error")
		}

	case codeChangeMsg:
		if msg.needConfirm {
			// 存储待确认的修改
			b.pendingChanges[msg.changeId] = msg
			b.waitingForConfirm = true
			b.currentChangeId = msg.changeId
			b.uiMode = "confirm" // 切换到确认模式
			
			// 显示diff
			b.showCodeChangeDiff(msg)
		} else {
			// 直接显示修改结果
			b.showCodeChangeResult(msg)
		}

	case userConfirmMsg:
		// 向响应channel发送确认结果
		if responseChan, exists := b.responseChannels[msg.changeId]; exists {
			if msg.approved {
				b.addMessage("✅ 用户确认修改，正在执行...", "system")
			} else {
				b.addMessage("❌ 用户取消修改操作", "system")
				b.addMessage("💡 修改已停止，请输入进一步的指令或问题继续对话", "system")
			}
			
			// 发送确认结果到channel
			responseChan <- msg.approved
			
			// 更新状态
			b.waitingForConfirm = false
			b.currentChangeId = ""
			b.uiMode = "normal" // 切回正常模式
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

	// Update input 或 confirmList
	var cmd tea.Cmd
	if b.uiMode == "confirm" {
		// 在确认模式下更新确认列表
		b.confirmList, cmd = b.confirmList.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		// 正常模式下更新输入框
		b.input, cmd = b.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	b.viewport, cmd = b.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return b, tea.Batch(cmds...)
}

// View renders the UI
func (b *BubbleTeaTUI) View() string {
	// Main content area
	content := b.viewport.View()

	// Input area 或 确认界面
	var bottomSection string
	if b.uiMode == "confirm" {
		// 在确认模式下显示选择列表
		bottomSection = b.confirmList.View()
	} else {
		// 正常模式下显示输入框
		bottomSection = b.inputStyle.Render(b.input.View())
	}

	// Status line
	statusLine := b.renderStatus()

	// Simple vertical layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		bottomSection,
		statusLine,
	)
}

// addMessage adds a message to the chat history
func (b *BubbleTeaTUI) addMessage(message, msgType string) {
	b.addMessageWithParams(message, "", msgType)
}

// addMessageWithParams adds a message with optional parameters to the chat history
func (b *BubbleTeaTUI) addMessageWithParams(message, params, msgType string) {
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
	case "tool_with_params":
		// 分别处理函数名和参数的样式
		toolNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
		paramsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // 灰色

		toolName := toolNameStyle.Render(fmt.Sprintf("[%s] %s", timestamp, message))
		paramsText := ""
		if params != "" {
			paramsText = " " + paramsStyle.Render(params)
		}
		styledMessage = toolName + paramsText
	case "explanation":
		wrappedMsg := b.wrapText(fmt.Sprintf("  %s", message), maxWidth) // 缩进显示
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // 灰色
			Render(wrappedMsg)
	case "tool_description":
		wrappedMsg := b.wrapText(message, maxWidth) // 不添加时间戳和缩进
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
	case "diff_added":
		wrappedMsg := b.wrapText(message, maxWidth)
		styledMessage = b.diffAddedStyle.Render(wrappedMsg)
	case "diff_removed":
		wrappedMsg := b.wrapText(message, maxWidth)
		styledMessage = b.diffRemovedStyle.Render(wrappedMsg)
	case "diff_header":
		wrappedMsg := b.wrapText(message, maxWidth)
		styledMessage = b.diffHeaderStyle.Render(wrappedMsg)
	case "confirm":
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true).
			Render(wrappedMsg)
	case "success":
		wrappedMsg := b.wrapText(fmt.Sprintf("[%s] %s", timestamp, message), maxWidth)
		styledMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
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

// // renderTodos renders the todo list with checkboxes
// func (b *BubbleTeaTUI) renderTodos() string {
// 	if len(b.todos) == 0 {
// 		return b.todoStyle.Render("📝 TodoList\n\n暂无任务")
// 	}

// 	// 直接显示TodoList原始数据
// 	for _, todo := range b.todos {
// 		if todo.Status == "display" {
// 			return b.todoStyle.Render(todo.Content)
// 		}
// 	}

// 	return b.todoStyle.Render("📝 TodoList\n\n数据加载中...")
// }

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

	// 设置处理状态
	b.lukatinCode.isProcessing = true
	defer func() {
		b.lukatinCode.isProcessing = false
		b.lukatinCode.Logger.Println("AI任务处理状态已重置")
	}()

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
					var toolCallInfos []toolCallInfo
					for i, toolCall := range msg.ToolCalls {
						b.lukatinCode.Logger.Printf("处理工具调用%d: %s", i+1, toolCall.Function.Name)
						toolCalls = append(toolCalls, toolCall.Function.Name)
						b.lukatinCode.Logger.Println("工具调用时的其他细节：", toolCall)

						// 格式化参数
						formattedParams := b.formatToolParams(toolCall.Function.Arguments)

						// 收集工具调用信息
						toolCallInfos = append(toolCallInfos, toolCallInfo{
							toolName: toolCall.Function.Name,
							params:   formattedParams,
						})
					}

					// 一次性发送所有工具调用到UI
					if b.program != nil && len(toolCallInfos) > 0 {
						b.program.Send(toolCallMsg{
							toolCalls: toolCallInfos,
						})
					}
				}

				// 处理文本内容
				for i, content := range msg.Content {
					b.lukatinCode.Logger.Printf("处理内容%d: Type=%s, Text长度=%d",
						i+1, content.Type, len(content.Text))
					if content.Type == general.ContentTypeText && content.Text != "" {
						// 如果有工具调用，这些可能是说明文本，需要判断是否显示
						if len(msg.ToolCalls) > 0 {
							// 判断是否为说明性文本
							if b.lukatinCode.isExplanatoryText(content.Text) {
								// 截取前三行或最大100字符
								truncatedText := b.lukatinCode.truncateText(content.Text)
								if b.program != nil {
									b.program.Send(aiResponseMsg{
										sender:  "explanation",
										message: truncatedText,
										isError: false,
									})
								}
								b.lukatinCode.Logger.Printf("添加说明文本: %s", truncatedText)
							} else {
								b.lukatinCode.Logger.Printf("跳过结果文本: %s", content.Text)
							}
						} else {
							// 没有工具调用的普通AI回复
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

	// 创建可取消的context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动取消监听goroutine
	go func() {
		select {
		case <-b.lukatinCode.cancelChan:
			b.lukatinCode.Logger.Println("收到取消信号，正在中断AI任务")
			cancel()
			if b.program != nil {
				b.program.Send(statusMsg{status: "任务已取消"})
			}
		case <-ctx.Done():
			// AI任务正常完成或其他原因取消
		}
	}()

	model := b.lukatinCode.Lmmconfig.AgentAPIKey.OpenAI.Model
	b.lukatinCode.Logger.Printf("================== 开始网络请求 ==================")
	b.lukatinCode.Logger.Printf("请求模型: %s", model)
	b.lukatinCode.Logger.Printf("请求提供商: %s", general.ProviderOpenAI)
	b.lukatinCode.Logger.Printf("输入文本长度: %d 字符", len(input))

	// 记录网络请求开始时间
	networkStart := time.Now()

	// 构建已注册的工具列表
	b.lukatinCode.CM.SetMaxFunctionCallingNums(10000000)
	_, _, err, usage := b.lukatinCode.CM.Chat(ctx, general.ProviderOpenAI, model, input, []string{}, info_chan)
	networkDuration := time.Since(networkStart)

	// 总体耗时
	totalDuration := time.Since(start)

	// 详细的时间记录
	b.lukatinCode.Logger.Printf("================== 网络请求完成 ==================")
	b.lukatinCode.Logger.Printf("网络请求耗时: %v", networkDuration)
	b.lukatinCode.Logger.Printf("总体处理耗时: %v", totalDuration)
	b.lukatinCode.Logger.Printf("本地处理耗时: %v (总时间 - 网络时间)", totalDuration-networkDuration)

	// Token使用统计
	if usage != nil {
		b.lukatinCode.Logger.Printf("Token使用情况: Prompt=%d, Completion=%d, Total=%d",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	} else {
		b.lukatinCode.Logger.Printf("Token使用情况: 未获取到usage数据")
	}

	// 记录到专门的网络性能日志文件
	b.logNetworkPerformance(input, model, networkDuration, totalDuration, usage, err)

	close(info_chan)
	b.lukatinCode.Logger.Println("关闭info_chan")

	// 等待消息处理完成
	b.lukatinCode.Logger.Println("等待消息处理goroutine完成")
	wg.Wait()
	b.lukatinCode.Logger.Println("消息处理完成")

	if err != nil {
		b.lukatinCode.Logger.Printf("AI调用出错: %v", err)
		if b.program != nil {
			// 检查是否因为取消而出错
			if ctx.Err() == context.Canceled {
				b.program.Send(aiResponseMsg{
					sender:  "assistant",
					message: "AI任务已被用户取消",
					isError: false,
				})
			} else {
				b.program.Send(aiResponseMsg{
					sender:  "assistant",
					message: fmt.Sprintf("处理失败: %v", err),
					isError: true,
				})
			}
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

// RequestEditConfirmation 实现UIInteractor接口，请求编辑确认
func (b *BubbleTeaTUI) RequestEditConfirmation(filePath, oldContent, newContent, operation string) bool {
	// 生成唯一的change ID
	changeId := b.generateChangeId()
	
	// 创建响应channel
	responseChan := make(chan bool, 1)
	
	// 存储等待确认的修改和响应channel
	b.pendingChanges[changeId] = codeChangeMsg{
		filePath:    filePath,
		oldContent:  oldContent,
		newContent:  newContent,
		operation:   operation,
		needConfirm: true,
		changeId:    changeId,
	}
	
	// 存储响应channel（需要一个map来管理）
	if b.responseChannels == nil {
		b.responseChannels = make(map[string]chan bool)
	}
	b.responseChannels[changeId] = responseChan
	
	// 发送代码修改消息到UI
	if b.program != nil {
		b.program.Send(codeChangeMsg{
			filePath:    filePath,
			oldContent:  oldContent,
			newContent:  newContent,
			operation:   operation,
			needConfirm: true,
			changeId:    changeId,
		})
	}
	
	// 阻塞等待用户确认
	confirmed := <-responseChan
	
	// 清理
	delete(b.pendingChanges, changeId)
	delete(b.responseChannels, changeId)
	
	return confirmed
}

// generateChangeId 生成唯一的change ID
func (b *BubbleTeaTUI) generateChangeId() string {
	// 简单的时间戳ID
	return fmt.Sprintf("change_%d", time.Now().UnixNano())
}

// truncateParams 截断参数字符串到指定长度
func (b *BubbleTeaTUI) truncateParams(params string, maxLen int) string {
	if len(params) <= maxLen {
		return params
	}
	return params[:maxLen] + "..."
}

// formatToolParams 格式化工具调用参数
func (b *BubbleTeaTUI) formatToolParams(arguments []byte) string {
	if len(arguments) == 0 {
		return ""
	}

	// 转换为字符串
	argsStr := string(arguments)

	// 移除换行符和多余空格
	cleaned := strings.ReplaceAll(argsStr, "\n", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	// 截断到50字符
	return b.truncateParams(cleaned, 50)
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

// showCodeChangeDiff 显示代码修改的彩色diff
func (b *BubbleTeaTUI) showCodeChangeDiff(change codeChangeMsg) {
	b.addMessage(fmt.Sprintf("📝 准备修改文件: %s", change.filePath), "system")
	b.addMessage(fmt.Sprintf("🔧 操作类型: %s", change.operation), "system")
	
	// 显示diff
	if change.oldContent != "" && change.newContent != "" {
		b.addMessage("═══════════ DIFF ═══════════", "diff_header")
		
		// 更好的diff算法：查找实际变化的内容
		oldLines := strings.Split(change.oldContent, "\n")
		newLines := strings.Split(change.newContent, "\n")
		
		// 显示上下文和变化
		b.showLineDiff(oldLines, newLines)
		
		b.addMessage("═══════════════════════════", "diff_header")
	} else if change.newContent != "" {
		// 新文件
		b.addMessage("📄 新文件内容:", "system")
		lines := strings.Split(change.newContent, "\n")
		for _, line := range lines {
			b.addMessage(fmt.Sprintf("+ %s", line), "diff_added")
		}
	}
}

// showLineDiff 显示行级diff
func (b *BubbleTeaTUI) showLineDiff(oldLines, newLines []string) {
	// 找到第一个不同的行
	firstDiff := -1
	minLen := len(oldLines)
	if len(newLines) < minLen {
		minLen = len(newLines)
	}
	
	for i := 0; i < minLen; i++ {
		if oldLines[i] != newLines[i] {
			firstDiff = i
			break
		}
	}
	
	// 如果没有找到不同的行，检查长度是否不同
	if firstDiff == -1 && len(oldLines) != len(newLines) {
		firstDiff = minLen
	}
	
	if firstDiff == -1 {
		b.addMessage("(内容相同，无变化)", "system")
		return
	}
	
	// 显示上下文（变化前的几行）
	contextStart := firstDiff - 2
	if contextStart < 0 {
		contextStart = 0
	}
	
	for i := contextStart; i < firstDiff; i++ {
		if i < len(oldLines) {
			b.addMessage(fmt.Sprintf("  %s", oldLines[i]), "system")
		}
	}
	
	// 显示删除的行
	changeEnd := len(oldLines)
	for i := firstDiff; i < changeEnd; i++ {
		if i < len(oldLines) {
			b.addMessage(fmt.Sprintf("- %s", oldLines[i]), "diff_removed")
		}
	}
	
	// 显示添加的行
	changeEnd = len(newLines)
	for i := firstDiff; i < changeEnd; i++ {
		if i < len(newLines) {
			b.addMessage(fmt.Sprintf("+ %s", newLines[i]), "diff_added")
		}
	}
	
	// 显示上下文（变化后的几行）
	contextEnd := firstDiff + 3
	if len(newLines) > firstDiff {
		maxContext := len(newLines)
		if maxContext > contextEnd {
			maxContext = contextEnd
		}
		for i := len(newLines); i < maxContext; i++ {
			if i < len(newLines) {
				b.addMessage(fmt.Sprintf("  %s", newLines[i]), "system")
			}
		}
	}
}

// showCodeChangeResult 显示代码修改结果
func (b *BubbleTeaTUI) showCodeChangeResult(change codeChangeMsg) {
	switch change.operation {
	case "edit":
		b.addMessage(fmt.Sprintf("✅ 文件 %s 修改完成", change.filePath), "success")
	case "multiedit":
		b.addMessage(fmt.Sprintf("✅ 文件 %s 批量修改完成", change.filePath), "success")
	case "write":
		b.addMessage(fmt.Sprintf("✅ 文件 %s 写入完成", change.filePath), "success")
	}
}

// executeCodeChange 执行代码修改
func (b *BubbleTeaTUI) executeCodeChange(change codeChangeMsg) {
	// 这里应该调用实际的文件操作函数
	// 为了简化，现在只显示执行结果
	go func() {
		time.Sleep(500 * time.Millisecond) // 模拟执行时间
		if b.program != nil {
			b.program.Send(codeChangeMsg{
				filePath:    change.filePath,
				operation:   change.operation,
				needConfirm: false,
			})
		}
	}()
}

// logNetworkPerformance 记录网络性能数据到专门的日志文件
func (b *BubbleTeaTUI) logNetworkPerformance(input, model string, networkDuration, totalDuration time.Duration, usage *general.Usage, err error) {
	// 确保log目录存在
	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.MkdirAll("log", 0755)
	}

	// 打开或创建network_performance.txt文件
	file, err := os.OpenFile("log/network_performance.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	// 写入详细的网络性能数据
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	status := "SUCCESS"
	if err != nil {
		status = fmt.Sprintf("ERROR: %v", err)
	}

	// 计算网络占比
	networkRatio := float64(networkDuration) / float64(totalDuration) * 100

	// 处理Token信息
	var tokenInfo string
	if usage != nil {
		tokenInfo = fmt.Sprintf("Prompt:%d|Completion:%d|Total:%d",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	} else {
		tokenInfo = "Token:N/A"
	}

	fmt.Fprintf(file, "%s | %s | %s | Input:%d chars | Network:%v | Total:%v | Ratio:%.1f%% | %s | %s\n",
		timestamp,
		general.ProviderOpenAI,
		model,
		len(input),
		networkDuration,
		totalDuration,
		networkRatio,
		tokenInfo,
		status,
	)
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

