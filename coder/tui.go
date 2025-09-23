package coder

import (
	"fmt"
	"lukatincode/function"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TUIComponent struct {
	app          *tview.Application
	chatView     *tview.TextView
	todoView     *tview.TextView
	inputField   *tview.InputField
	statusBar    *tview.TextView
	mainFlex     *tview.Flex
	rightPanel   *tview.Flex
	mu           sync.Mutex
	isProcessing bool
	lukatinCode  *LukatinCode
}

func NewTUIComponent(lukatinCode *LukatinCode) *TUIComponent {
	tui := &TUIComponent{
		app:         tview.NewApplication(),
		lukatinCode: lukatinCode,
	}
	tui.setupUI()
	return tui
}

func (tui *TUIComponent) setupUI() {
	// 创建聊天显示区域
	tui.chatView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)
	tui.chatView.SetBorder(true).
		SetTitle(" 🤖 LukatinCode 对话 ").
		SetTitleAlign(tview.AlignLeft)

	// 创建TodoList显示区域
	tui.todoView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)
	tui.todoView.SetBorder(true).
		SetTitle(" 📝 TodoList ").
		SetTitleAlign(tview.AlignLeft)

	// 创建输入框
	tui.inputField = tview.NewInputField().
		SetLabel("💬 输入: ").
		SetFieldWidth(0)

	// 设置输入框的按键处理
	tui.inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			input := strings.TrimSpace(tui.inputField.GetText())
			if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
				tui.lukatinCode.Logger.Printf("TUI收到输入: '%s'", input)
			}
			if input != "" {
				if input == "exit" || input == "quit" {
					if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
						tui.lukatinCode.Logger.Println("用户请求退出")
					}
					tui.app.Stop()
					return nil
				}
				tui.inputField.SetText("")
				tui.addChatMessage("👤 用户", input, "cyan")
				if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
					tui.lukatinCode.Logger.Printf("启动ChatWithTUI处理输入: %s", input)
				}
				go tui.lukatinCode.ChatWithTUI(input)
			}
			return nil
		}
		return event
	})

	tui.inputField.SetBorder(true).
		SetTitle(" 消息输入 ").
		SetTitleAlign(tview.AlignLeft)

	// 创建状态栏
	tui.statusBar = tview.NewTextView().
		SetDynamicColors(true)
	tui.statusBar.SetBorder(true).
		SetTitle(" 状态 ").
		SetTitleAlign(tview.AlignLeft)
	// 直接设置初始状态文本，不使用QueueUpdateDraw
	fmt.Fprint(tui.statusBar, "[green]✅ 就绪[white]")

	// 创建右侧面板（TodoList + 状态栏）
	tui.rightPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tui.todoView, 0, 3, false).
		AddItem(tui.statusBar, 3, 0, false)

	// 创建主布局
	leftPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tui.chatView, 0, 1, false).
		AddItem(tui.inputField, 3, 0, true)

	tui.mainFlex = tview.NewFlex().
		AddItem(leftPanel, 0, 2, true).
		AddItem(tui.rightPanel, 40, 0, false)

	// 设置应用根视图
	tui.app.SetRoot(tui.mainFlex, true).SetFocus(tui.inputField)

	// 设置全局键盘快捷键
	tui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// 只处理Ctrl组合键，让其他按键正常传递
		if event.Modifiers()&tcell.ModCtrl != 0 {
			switch event.Key() {
			case tcell.KeyCtrlC:
				tui.app.Stop()
				return nil
			case tcell.KeyCtrlT:
				tui.updateTodoList()
				return nil
			case tcell.KeyCtrlL:
				tui.chatView.Clear()
				tui.displayWelcome()
				return nil
			}
		}
		return event
	})
}

func (tui *TUIComponent) displayWelcome() {
	welcome := `[blue]═══════════════════════════════════════[white]
[green]🚀 欢迎使用 LukatinCode![white]
[yellow]✨ 您的智能编程助手[white]

[cyan]快捷键:[white]
• [yellow]Enter[white] - 发送消息
• [yellow]Ctrl+C[white] - 退出程序
• [yellow]Ctrl+T[white] - 刷新TodoList
• [yellow]Ctrl+L[white] - 清空对话

[blue]═══════════════════════════════════════[white]

`
	fmt.Fprint(tui.chatView, welcome)
}

func (tui *TUIComponent) addChatMessage(sender, message, color string) {
	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("\n[%s][%s] %s[white]\n%s\n[gray]────────────────────────────────────────[white]\n",
		color, timestamp, sender, message)
	fmt.Fprint(tui.chatView, formatted)
	tui.chatView.ScrollToEnd()
}

func (tui *TUIComponent) updateTodoList() {
	todoData := function.TodoRead()
	tui.app.QueueUpdateDraw(func() {
		tui.todoView.Clear()
		fmt.Fprint(tui.todoView, todoData)
	})
}

func (tui *TUIComponent) updateTodoListDirect() {
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		tui.lukatinCode.Logger.Println("开始直接更新TodoList")
	}
	todoData := function.TodoRead()
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		tui.lukatinCode.Logger.Printf("获取TodoList数据: %d字符", len(todoData))
	}
	tui.todoView.Clear()
	fmt.Fprint(tui.todoView, todoData)
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		tui.lukatinCode.Logger.Println("TodoList更新完成")
	}
}

func (tui *TUIComponent) updateStatus(status string) {
	tui.app.QueueUpdateDraw(func() {
		tui.statusBar.Clear()
		timestamp := time.Now().Format("15:04:05")
		fmt.Fprintf(tui.statusBar, "[gray]%s[white] %s", timestamp, status)
	})
}

func (tui *TUIComponent) Run() error {
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		tui.lukatinCode.Logger.Println("TUI组件开始运行")
	}

	// 在应用开始运行后初始化显示内容
	go func() {
		// 稍微延迟以确保应用已启动
		time.Sleep(100 * time.Millisecond)
		if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
			tui.lukatinCode.Logger.Println("初始化TUI显示内容")
		}
		tui.app.QueueUpdateDraw(func() {
			tui.displayWelcome()
			tui.updateTodoListDirect()
		})
	}()

	err := tui.app.Run()
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		if err != nil {
			tui.lukatinCode.Logger.Printf("TUI应用运行结束，错误: %v", err)
		} else {
			tui.lukatinCode.Logger.Println("TUI应用正常结束")
		}
	}
	return err
}

func (tui *TUIComponent) Stop() {
	tui.app.Stop()
}

// DisplayMessage 显示消息到聊天界面
func (tui *TUIComponent) DisplayMessage(sender, message, color string) {
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		tui.lukatinCode.Logger.Printf("TUI显示消息 - 发送者: %s, 消息: %s, 颜色: %s", sender, message, color)
	}
	tui.app.QueueUpdateDraw(func() {
		tui.addChatMessage(sender, message, color)
	})
}

// DisplayToolCall 显示工具调用
func (tui *TUIComponent) DisplayToolCall(toolName string) {
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		tui.lukatinCode.Logger.Printf("TUI显示工具调用: %s", toolName)
	}
	tui.app.QueueUpdateDraw(func() {
		if toolName == "TodoRead" || toolName == "TodoWrite" {
			tui.addChatMessage("🔧 工具调用", "📝 TodoList 管理", "yellow")
			// 直接调用updateTodoListDirect避免嵌套QueueUpdateDraw
			tui.updateTodoListDirect()
		} else {
			tui.addChatMessage("🔧 工具调用", fmt.Sprintf("🔧 %s", toolName), "yellow")
		}
	})
}

// UpdateStatus 更新状态栏
func (tui *TUIComponent) UpdateStatus(status string) {
	if tui.lukatinCode != nil && tui.lukatinCode.Logger != nil {
		tui.lukatinCode.Logger.Printf("TUI更新状态: %s", status)
	}
	tui.updateStatus(status)
}
