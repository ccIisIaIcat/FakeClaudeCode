package coder

import (
	"fmt"
	"log"
	"lukatincode/function"
	"os"
	"strings"
	"time"

	"runtime"

	"github.com/ccIisIaIcat/GoAgent/agent/ConversationManager"
	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

type LukatinCode struct {
	Lmmconfig       *general.LLMConfig
	CM              *ConversationManager.ConversationManager
	BubbleTUI       *BubbleTeaTUI    // New Bubble Tea TUI
	PersistentShell *PersistentShell // 持久化Shell
	Logger          *log.Logger
	LogFile         *os.File
	cancelChan      chan struct{} // 用于取消AI任务
	isProcessing    bool          // 标记是否正在处理AI任务
}

func GenLukatinCode(lmmconfig *general.LLMConfig, system_promote string) *LukatinCode {
	lc := &LukatinCode{
		Lmmconfig:    lmmconfig,
		cancelChan:   make(chan struct{}),
		isProcessing: false,
	}

	// 初始化日志文件（写入 log 目录）
	// 检查并创建log目录
	if _, err := os.Stat("log"); os.IsNotExist(err) {
		fmt.Println("创建log目录...")
		if err := os.MkdirAll("log", 0755); err != nil {
			log.Fatalf("无法创建log目录: %v", err)
		}
		fmt.Println("log目录创建成功")
	} else {
		fmt.Println("log目录已存在")
	}

	logFileName := fmt.Sprintf("log/lukatincode_%s.log", time.Now().Format("20060102_150405"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	lc.LogFile = logFile
	lc.Logger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	lc.Logger.Println("=================== LukatinCode 启动 ===================")
	lc.Logger.Printf("配置文件: %+v", lmmconfig)
	lc.Logger.Printf("系统提示: %s", system_promote)

	agentManager := general.NewAgentManager()
	for _, v := range lc.Lmmconfig.ToProviderConfigs() {
		agentManager.AddProvider(v)
		lc.Logger.Printf("添加Provider: %+v", v)
	}
	lc.CM = ConversationManager.NewConversationManager(agentManager)

	// 动态注入环境信息到系统提示
	wd, _ := os.Getwd()
	gitRepo := "No"
	if _, err := os.Stat(".git"); err == nil {
		gitRepo = "Yes"
	}
	platform := runtime.GOOS
	osVersion := runtime.GOARCH
	dateStr := time.Now().Format("2006-01-02")
	model := "unknown"

	envBlock := fmt.Sprintf("<env>\nWorking directory: %s\nIs directory a git repo: %s\nPlatform: %s\nOS Version: %s\nToday's date: %s\nModel: %s\n</env>",
		wd, gitRepo, platform, osVersion, dateStr, model,
	)

	// 替换占位符或在尾部追加
	if strings.Contains(system_promote, "{{RUNTIME_INJECT}}") {
		system_promote = strings.ReplaceAll(system_promote, "Working directory: {{RUNTIME_INJECT}}", "Working directory: "+wd)
		system_promote = strings.ReplaceAll(system_promote, "Is directory a git repo: {{RUNTIME_INJECT}}", "Is directory a git repo: "+gitRepo)
		system_promote = strings.ReplaceAll(system_promote, "Platform: {{RUNTIME_INJECT}}", "Platform: "+platform)
		system_promote = strings.ReplaceAll(system_promote, "OS Version: {{RUNTIME_INJECT}}", "OS Version: "+osVersion)
		system_promote = strings.ReplaceAll(system_promote, "Today's date: {{RUNTIME_INJECT}}", "Today's date: "+dateStr)
		system_promote = strings.ReplaceAll(system_promote, "Model: {{RUNTIME_INJECT}}", "Model: "+model)
	} else {
		system_promote = system_promote + "\n\n" + envBlock
	}

	lc.CM.SetSystemPrompt(system_promote)
	
	// 检测和安装 ripgrep
	lc.Logger.Println("检测 ripgrep 状态")
	function.LogRipgrepStatus()
	
	lc.RegisterAllFunction()

	// 初始化新的Bubble Tea TUI
	lc.Logger.Println("初始化Bubble Tea TUI组件")
	lc.BubbleTUI = NewBubbleTeaTUI(lc)
	
	// UI实例已经在LukatinCode结构中可用

	// 初始化持久化Shell
	lc.Logger.Println("初始化持久化Shell")
	lc.PersistentShell = NewPersistentShell()
	if err := lc.PersistentShell.Start(); err != nil {
		lc.Logger.Printf("启动持久化Shell失败: %v", err)
		fmt.Printf("警告: 启动持久化Shell失败: %v\n", err)
	} else {
		lc.Logger.Println("持久化Shell启动成功")
		fmt.Println("持久化Shell启动成功")
	}

	return lc
}

type FunctionParam struct {
	Description string `json:"description"`
	Type        string `json:"type"`
}

type FunctionParameters struct {
	Properties map[string]FunctionParam `json:"properties"`
	Required   []string                 `json:"required"`
}

type FunctionDescription struct {
	Description string             `json:"description"`
	Parameters  FunctionParameters `json:"parameters"`
}

// StartBubbleTUI 启动新的Bubble Tea TUI界面模式
func (lc *LukatinCode) StartBubbleTUI() error {
	lc.Logger.Println("启动Bubble Tea TUI界面模式")
	defer lc.Logger.Println("Bubble Tea TUI界面模式结束")
	defer lc.Cleanup()

	err := lc.BubbleTUI.Run()
	if err != nil {
		lc.Logger.Printf("Bubble Tea TUI运行错误: %v", err)
	}
	return err
}

// BashWithTimeout 使用持久化Shell执行bash命令（带超时）
func (lc *LukatinCode) BashWithTimeout(command string, timeout int) string {
	return lc.bashInternal(command, timeout)
}

// isExplanatoryText 判断文本是否为说明性文本（而非工具调用结果）
func (lc *LukatinCode) isExplanatoryText(text string) bool {
	// 简化版本：只要是文本就显示，但只显示前三行或最多100字符
	return true
}

// truncateText 截取文本的前三行或最大100字符
func (lc *LukatinCode) truncateText(text string) string {
	// 按行分割
	lines := strings.Split(text, "\n")

	// 取前三行
	var selectedLines []string
	for i, line := range lines {
		if i < 3 {
			selectedLines = append(selectedLines, line)
		} else {
			break
		}
	}

	result := strings.Join(selectedLines, "\n")

	// 如果结果超过100字符，截取前100字符并添加省略号
	if len(result) > 100 {
		result = result[:100] + "..."
	}

	return result
}

// CancelCurrentTask 取消当前AI任务
func (lc *LukatinCode) CancelCurrentTask() {
	if lc.isProcessing {
		lc.Logger.Println("用户请求取消当前AI任务")
		select {
		case lc.cancelChan <- struct{}{}:
			lc.Logger.Println("取消信号已发送")
		default:
			lc.Logger.Println("取消信号发送失败，通道可能已满")
		}
	} else {
		lc.Logger.Println("当前没有正在处理的AI任务")
	}
}

// Cleanup 清理资源
func (lc *LukatinCode) Cleanup() {
	lc.Logger.Println("开始清理资源")

	// 关闭持久化Shell
	if lc.PersistentShell != nil && lc.PersistentShell.IsRunning() {
		lc.Logger.Println("停止持久化Shell")
		if err := lc.PersistentShell.Stop(); err != nil {
			lc.Logger.Printf("停止持久化Shell失败: %v", err)
		} else {
			lc.Logger.Println("持久化Shell已停止")
		}
	}

	// 关闭日志文件
	if lc.LogFile != nil {
		lc.LogFile.Close()
	}

	lc.Logger.Println("资源清理完成")
}
