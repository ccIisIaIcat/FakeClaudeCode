package coder

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"lukatincode/function"
	"os"
	"strings"
	"sync"
	"time"

	"runtime"

	"github.com/ccIisIaIcat/GoAgent/agent/ConversationManager"
	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

type LukatinCode struct {
	Lmmconfig       *general.LLMConfig
	CM              *ConversationManager.ConversationManager
	TUI             *TUIComponent    // Old tview TUI
	BubbleTUI       *BubbleTeaTUI    // New Bubble Tea TUI
	PersistentShell *PersistentShell // 持久化Shell
	Logger          *log.Logger
	LogFile         *os.File
}

func GenLukatinCode(lmmconfig *general.LLMConfig, system_promote string) *LukatinCode {
	lc := &LukatinCode{
		Lmmconfig: lmmconfig,
	}

	// 初始化日志文件（写入 log 目录）
	_ = os.MkdirAll("log", 0755)
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
	lc.RegisterAllFunction()

	// 初始化TUI组件
	lc.Logger.Println("初始化TUI组件")
	lc.TUI = NewTUIComponent(lc)

	// 初始化新的Bubble Tea TUI
	lc.Logger.Println("初始化Bubble Tea TUI组件")
	lc.BubbleTUI = NewBubbleTeaTUI(lc)

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

func (lc *LukatinCode) RegisterAllFunction() {
	lc.Logger.Println("开始注册函数")

	// 读取函数描述文件
	functionDescFile := "./function/function_description.json"
	data, err := ioutil.ReadFile(functionDescFile)
	if err != nil {
		lc.Logger.Printf("读取函数描述文件失败: %v", err)
		fmt.Printf("读取函数描述文件失败: %v\n", err)
		return
	}

	var functionDescs map[string]FunctionDescription
	err = json.Unmarshal(data, &functionDescs)
	if err != nil {
		lc.Logger.Printf("解析函数描述文件失败: %v", err)
		fmt.Printf("解析函数描述文件失败: %v\n", err)
		return
	}

	// 注册 Bash 函数 (使用持久化Shell)
	if desc, ok := functionDescs["Bash"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Bash", desc.Description, lc.Bash, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Bash函数失败: %v", err)
			fmt.Printf("注册Bash函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Bash函数 (持久化Shell)")
			fmt.Printf("成功注册Bash函数 (持久化Shell)\n")
		}
	}

	// 注册 TodoRead 函数
	if desc, ok := functionDescs["TodoRead"]; ok {
		err := lc.CM.RegisterFunction("TodoRead", desc.Description, function.TodoRead, []string{}, []string{})
		if err != nil {
			lc.Logger.Printf("注册TodoRead函数失败: %v", err)
			fmt.Printf("注册TodoRead函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册TodoRead函数")
			fmt.Printf("成功注册TodoRead函数\n")
		}
	}

	// 注册 TodoWrite 函数
	if desc, ok := functionDescs["TodoWrite"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("TodoWrite", desc.Description, function.TodoWrite, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册TodoWrite函数失败: %v", err)
			fmt.Printf("注册TodoWrite函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册TodoWrite函数")
			fmt.Printf("成功注册TodoWrite函数\n")
		}
	}

	// 注册 Grep 函数
	if desc, ok := functionDescs["Grep"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Grep", desc.Description, function.Grep, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Grep函数失败: %v", err)
			fmt.Printf("注册Grep函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Grep函数")
			fmt.Printf("成功注册Grep函数\n")
		}
	}

	// 注册 Glob 函数
	if desc, ok := functionDescs["Glob"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Glob", desc.Description, function.Glob, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Glob函数失败: %v", err)
			fmt.Printf("注册Glob函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Glob函数")
			fmt.Printf("成功注册Glob函数\n")
		}
	}

	// 注册 Task 函数
	if desc, ok := functionDescs["Task"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Task", desc.Description, function.Task, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Task函数失败: %v", err)
			fmt.Printf("注册Task函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Task函数")
			fmt.Printf("成功注册Task函数\n")
		}
	}

	// 注册 LS 函数
	if desc, ok := functionDescs["LS"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("LS", desc.Description, function.LS, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册LS函数失败: %v", err)
			fmt.Printf("注册LS函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册LS函数")
			fmt.Printf("成功注册LS函数\n")
		}
	}

	// 注册 Read 函数
	if desc, ok := functionDescs["Read"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Read", desc.Description, function.Read, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Read函数失败: %v", err)
			fmt.Printf("注册Read函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Read函数")
			fmt.Printf("成功注册Read函数\n")
		}
	}

	// 注册 Edit 函数
	if desc, ok := functionDescs["Edit"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Edit", desc.Description, function.Edit, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Edit函数失败: %v", err)
			fmt.Printf("注册Edit函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Edit函数")
			fmt.Printf("成功注册Edit函数\n")
		}
	}

	// 注册 MultiEdit 函数
	if desc, ok := functionDescs["MultiEdit"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("MultiEdit", desc.Description, function.MultiEdit, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册MultiEdit函数失败: %v", err)
			fmt.Printf("注册MultiEdit函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册MultiEdit函数")
			fmt.Printf("成功注册MultiEdit函数\n")
		}
	}

	// 注册 Write 函数
	if desc, ok := functionDescs["Write"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("Write", desc.Description, function.Write, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册Write函数失败: %v", err)
			fmt.Printf("注册Write函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册Write函数")
			fmt.Printf("成功注册Write函数\n")
		}
	}

	// 注册 WebFetch 函数
	if desc, ok := functionDescs["WebFetch"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("WebFetch", desc.Description, function.WebFetch, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册WebFetch函数失败: %v", err)
			fmt.Printf("注册WebFetch函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册WebFetch函数")
			fmt.Printf("成功注册WebFetch函数\n")
		}
	}

	// 注册 WebSearch 函数
	if desc, ok := functionDescs["WebSearch"]; ok {
		var paramNames []string
		var paramDescs []string
		for _, param := range desc.Parameters.Required {
			if paramInfo, exists := desc.Parameters.Properties[param]; exists {
				paramNames = append(paramNames, param)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		// 添加可选参数
		for paramName, paramInfo := range desc.Parameters.Properties {
			isRequired := false
			for _, req := range desc.Parameters.Required {
				if req == paramName {
					isRequired = true
					break
				}
			}
			if !isRequired {
				paramNames = append(paramNames, paramName)
				paramDescs = append(paramDescs, paramInfo.Description)
			}
		}
		err := lc.CM.RegisterFunction("WebSearch", desc.Description, function.WebSearch, paramNames, paramDescs)
		if err != nil {
			lc.Logger.Printf("注册WebSearch函数失败: %v", err)
			fmt.Printf("注册WebSearch函数失败: %v\n", err)
		} else {
			lc.Logger.Println("成功注册WebSearch函数")
			fmt.Printf("成功注册WebSearch函数\n")
		}
	}

	lc.Logger.Println("函数注册完成")
}

// StartTUI 启动TUI界面模式 (旧版tview)
func (lc *LukatinCode) StartTUI() error {
	lc.Logger.Println("启动TUI界面模式")
	defer lc.Logger.Println("TUI界面模式结束")
	defer lc.LogFile.Close()

	err := lc.TUI.Run()
	if err != nil {
		lc.Logger.Printf("TUI运行错误: %v", err)
	}
	return err
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

// ChatWithTUI 带TUI显示的对话处理
func (lc *LukatinCode) ChatWithTUI(input string) {
	lc.Logger.Printf("=================== 开始处理用户输入 ===================")
	lc.Logger.Printf("用户输入: %s", input)
	lc.Logger.Println("更新状态为: AI正在思考...")
	lc.TUI.UpdateStatus("[yellow]🤔 AI正在思考...[white]")

	info_chan := make(chan general.Message, 10)
	var messages []string
	var toolCalls []string
	var wg sync.WaitGroup
	wg.Add(1)

	lc.Logger.Println("创建消息处理goroutine")
	// 处理消息
	go func() {
		defer wg.Done()
		lc.Logger.Println("开始处理来自AI的消息")
		messageCount := 0
		for msg := range info_chan {
			messageCount++
			lc.Logger.Printf("收到第%d条消息, Role: %s, ToolCalls数量: %d, Content数量: %d",
				messageCount, msg.Role, len(msg.ToolCalls), len(msg.Content))

			if msg.Role == general.RoleAssistant {
				// 处理工具调用
				if len(msg.ToolCalls) > 0 {
					for i, toolCall := range msg.ToolCalls {
						lc.Logger.Printf("处理工具调用%d: %s", i+1, toolCall.Function.Name)
						toolCalls = append(toolCalls, toolCall.Function.Name)
						lc.TUI.DisplayToolCall(toolCall.Function.Name)
					}
				}

				// 处理文本内容
				for i, content := range msg.Content {
					lc.Logger.Printf("处理内容%d: Type=%s, Text长度=%d",
						i+1, content.Type, len(content.Text))
					if content.Type == general.ContentTypeText && content.Text != "" {
						messages = append(messages, content.Text)
						lc.Logger.Printf("添加文本消息: %s", content.Text)
					}
				}
			}
		}
		lc.Logger.Printf("消息处理goroutine结束, 共处理%d条消息", messageCount)
	}()

	// 调用AI
	lc.Logger.Println("开始调用AI Chat方法")
	start := time.Now()
	// 从配置选择模型（若未配置则留空交由底层处理）
	model := lc.Lmmconfig.AgentAPIKey.OpenAI.Model
	_, _, err, _ := lc.CM.Chat(context.Background(), general.ProviderOpenAI, model, input, []string{}, info_chan)
	duration := time.Since(start)
	lc.Logger.Printf("AI Chat调用完成, 耗时: %v", duration)

	close(info_chan)
	lc.Logger.Println("关闭info_chan")

	// 等待消息处理完成
	lc.Logger.Println("等待消息处理goroutine完成")
	wg.Wait()
	lc.Logger.Println("消息处理完成")

	if err != nil {
		lc.Logger.Printf("AI调用出错: %v", err)
		lc.TUI.DisplayMessage("❌ 错误", fmt.Sprintf("处理失败: %v", err), "red")
	} else {
		// 显示AI回复
		lc.Logger.Printf("准备显示AI回复, 消息数量: %d, 工具调用数量: %d", len(messages), len(toolCalls))
		if len(messages) > 0 {
			response := strings.Join(messages, "\n")
			lc.Logger.Printf("显示AI回复: %s", response)
			lc.TUI.DisplayMessage("🤖 Assistant", response, "green")
		} else {
			lc.Logger.Println("没有收到AI文本回复，显示默认消息")
			lc.TUI.DisplayMessage("🤖 Assistant", "已处理完成", "green")
		}
	}

	lc.Logger.Println("更新状态为: 就绪")
	lc.TUI.UpdateStatus("[green]✅ 就绪[white]")
	lc.Logger.Printf("=================== 用户输入处理完成 ===================")
}

// Bash 使用持久化Shell执行bash命令
func (lc *LukatinCode) Bash(command string, timeout int) string {
	lc.Logger.Printf("执行Bash命令: %s", command)

	if command == "" {
		return `{"error": "Command is required", "exit_code": -1}`
	}

	// 检查PersistentShell是否正在运行
	if lc.PersistentShell == nil || !lc.PersistentShell.IsRunning() {
		lc.Logger.Println("PersistentShell未运行，尝试重新启动")

		// 尝试重新启动PersistentShell
		if lc.PersistentShell == nil {
			lc.PersistentShell = NewPersistentShell()
		}

		if err := lc.PersistentShell.Start(); err != nil {
			lc.Logger.Printf("重新启动PersistentShell失败: %v", err)
			return fmt.Sprintf(`{"error": "Failed to start persistent shell: %v", "exit_code": -1}`, err)
		}
		lc.Logger.Println("PersistentShell重新启动成功")
	}

	// 执行命令
	output, err := lc.PersistentShell.ExecuteCommand(command)

	if err != nil {
		lc.Logger.Printf("命令执行失败: %v", err)
		return fmt.Sprintf(`{"error": "%s", "exit_code": 1, "output": ""}`, err.Error())
	}

	lc.Logger.Printf("命令执行成功，输出长度: %d", len(output))

	// 格式化为JSON响应
	response := map[string]interface{}{
		"output":    output,
		"error":     "",
		"exit_code": 0,
	}

	responseJSON, _ := json.Marshal(response)
	return string(responseJSON)
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
