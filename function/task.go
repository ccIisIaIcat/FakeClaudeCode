package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ccIisIaIcat/GoAgent/agent/ConversationManager"
	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

type TaskRequest struct {
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
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

// logToTaskFile 记录Task函数专用日志
func logToTaskFile(message string) {
	// 确保log目录存在
	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.MkdirAll("log", 0755)
	}

	// 打开或创建task.txt文件
	file, err := os.OpenFile("log/task.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	// 写入时间戳和消息
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	fmt.Fprintf(file, "%s %s\n", timestamp, message)
}

func Task(description string, prompt string) string {
	// 添加调试日志
	logToTaskFile(fmt.Sprintf("Task函数开始执行 - description: %s", description))
	
	req := TaskRequest{
		Description: description,
		Prompt:      prompt,
	}

	logToTaskFile("Task函数：正在加载配置文件")
	config, err := general.LoadConfig("./LLMConfig.yaml")
	if err != nil {
		return fmt.Sprintf("Failed to load config: %v", err)
	}
	logToTaskFile("Task函数：配置文件加载成功")

	logToTaskFile("Task函数：正在创建AgentManager和ConversationManager")
	agentManager := general.NewAgentManager()
	agentManager.AddProvider(config.ToProviderConfigs()[0])

	cm := ConversationManager.NewConversationManager(agentManager)
	cm.SetSystemPrompt("You are a helpful AI assistant that can perform various tasks using available tools.")
	
	logToTaskFile("Task函数：正在注册函数")
	// 注册函数
	registerTaskFunctions(cm)
	logToTaskFile("Task函数：函数注册完成")

	logToTaskFile("Task函数：准备调用Chat方法")
	logToTaskFile(fmt.Sprintf("请求模型: %s", config.AgentAPIKey.OpenAI.Model))
	logToTaskFile(fmt.Sprintf("请求提供商: %s", general.ProviderOpenAI))
	logToTaskFile(fmt.Sprintf("输入文本长度: %d 字符", len(req.Prompt)))
	
	// 记录网络请求时间
	networkStart := time.Now()
	ctx := context.Background()
	messages, _, err, usage := cm.Chat(ctx, general.ProviderOpenAI, config.AgentAPIKey.OpenAI.Model, req.Prompt, []string{}, nil)
	networkDuration := time.Since(networkStart)
	
	logToTaskFile(fmt.Sprintf("Task函数：Chat方法调用完成，网络耗时: %v", networkDuration))
	
	// Token使用统计
	if usage != nil {
		logToTaskFile(fmt.Sprintf("Task Token使用情况: Prompt=%d, Completion=%d, Total=%d", 
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
	} else {
		logToTaskFile("Task Token使用情况: 未获取到usage数据")
	}
	
	// 性能分析
	if networkDuration > 10*time.Second {
		logToTaskFile(fmt.Sprintf("⚠️ 警告: Task网络请求耗时较长 (%v > 10s)，可能存在网络问题", networkDuration))
	} else if networkDuration > 5*time.Second {
		logToTaskFile(fmt.Sprintf("💡 提示: Task网络请求耗时 %v，属于正常范围但偏慢", networkDuration))
	} else {
		logToTaskFile(fmt.Sprintf("✅ Task网络请求响应良好: %v", networkDuration))
	}
	
	// 记录到网络性能日志
	logTaskNetworkPerformance(req.Prompt, config.AgentAPIKey.OpenAI.Model, networkDuration, usage, err)

	if err != nil {
		logToTaskFile(fmt.Sprintf("Task函数：Chat方法出错: %v", err))
		return fmt.Sprintf("Error: %v", err)
	}

	logToTaskFile(fmt.Sprintf("Task函数：正在处理返回的消息，消息数量: %d", len(messages)))
	// 直接从返回的messages中提取文本内容
	var response []string
	for i, msg := range messages {
		logToTaskFile(fmt.Sprintf("Task函数：处理消息%d，角色: %s，内容数量: %d", i+1, msg.Role, len(msg.Content)))
		if msg.Role == general.RoleAssistant {
			for j, content := range msg.Content {
				logToTaskFile(fmt.Sprintf("Task函数：处理内容%d，类型: %s，文本长度: %d", j+1, content.Type, len(content.Text)))
				if content.Type == general.ContentTypeText && content.Text != "" {
					response = append(response, content.Text)
				}
			}
		}
	}

	logToTaskFile(fmt.Sprintf("Task函数：提取到的响应数量: %d", len(response)))
	if len(response) == 0 {
		logToTaskFile("Task函数：返回默认完成消息")
		return "Task completed successfully"
	}

	result := strings.Join(response, "\n")
	logToTaskFile(fmt.Sprintf("Task函数：返回结果，长度: %d", len(result)))
	return result
}

// registerTaskFunctions 注册Task子代理需要的所有函数
func registerTaskFunctions(cm *ConversationManager.ConversationManager) {
	logToTaskFile("registerTaskFunctions：开始注册函数")
	// 读取函数描述文件
	functionDescFile := "./function/function_description.json"
	logToTaskFile(fmt.Sprintf("registerTaskFunctions：读取函数描述文件: %s", functionDescFile))
	data, err := ioutil.ReadFile(functionDescFile)
	if err != nil {
		logToTaskFile(fmt.Sprintf("registerTaskFunctions：读取文件失败: %v", err))
		return // 如果读取失败，跳过注册
	}

	logToTaskFile("registerTaskFunctions：解析JSON数据")
	var functionDescs map[string]FunctionDescription
	err = json.Unmarshal(data, &functionDescs)
	if err != nil {
		logToTaskFile(fmt.Sprintf("registerTaskFunctions：JSON解析失败: %v", err))
		return // 如果解析失败，跳过注册
	}

	// 注册指定的函数列表
	functionList := []string{"Bash", "Glob", "Grep", "LS", "Read", "Edit", "MultiEdit", "Write", "WebFetch", "TodoRead", "TodoWrite", "WebSearch"}
	logToTaskFile(fmt.Sprintf("registerTaskFunctions：准备注册%d个函数", len(functionList)))
	
	for _, funcName := range functionList {
		if desc, ok := functionDescs[funcName]; ok {
			var paramNames []string
			var paramDescs []string
			
			// 添加必需参数
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
			
			// 根据函数名注册对应的函数
			switch funcName {
			case "Bash":
				// 注意：Task子代理无法访问主程序的LukatinCode实例，所以使用简化的Bash函数
				cm.RegisterFunction("Bash", desc.Description, SimpleBash, paramNames, paramDescs)
			case "Glob":
				cm.RegisterFunction("Glob", desc.Description, Glob, paramNames, paramDescs)
			case "Grep":
				cm.RegisterFunction("Grep", desc.Description, Grep, paramNames, paramDescs)
			case "LS":
				cm.RegisterFunction("LS", desc.Description, LS, paramNames, paramDescs)
			case "Read":
				cm.RegisterFunction("Read", desc.Description, Read, paramNames, paramDescs)
			case "Edit":
				cm.RegisterFunction("Edit", desc.Description, Edit, paramNames, paramDescs)
			case "MultiEdit":
				cm.RegisterFunction("MultiEdit", desc.Description, MultiEdit, paramNames, paramDescs)
			case "Write":
				cm.RegisterFunction("Write", desc.Description, Write, paramNames, paramDescs)
			case "WebFetch":
				cm.RegisterFunction("WebFetch", desc.Description, WebFetch, paramNames, paramDescs)
			case "TodoRead":
				cm.RegisterFunction("TodoRead", desc.Description, TodoRead, paramNames, paramDescs)
			case "TodoWrite":
				cm.RegisterFunction("TodoWrite", desc.Description, TodoWrite, paramNames, paramDescs)
			case "WebSearch":
				cm.RegisterFunction("WebSearch", desc.Description, WebSearch, paramNames, paramDescs)
			}
		}
	}
}

// SimpleBash 简化的Bash函数，用于Task子代理
func SimpleBash(command string, description string, timeout int) string {
	logToTaskFile(fmt.Sprintf("SimpleBash：开始执行 - command: %s, timeout: %d", command, timeout))
	
	// 如果是grep命令，自动替换为最优搜索命令
	if strings.HasPrefix(strings.TrimSpace(command), "grep ") {
		searchCmd := GetOptimalSearchCommand()
		if searchCmd == "rg" {
			// 将grep命令转换为rg命令
			command = strings.Replace(command, "grep ", "rg ", 1)
			logToTaskFile(fmt.Sprintf("SimpleBash：检测到grep命令，自动替换为ripgrep: %s", command))
		}
	}
	
	if timeout == 0 {
		timeout = 120000 // 默认2分钟超时
	}
	
	// 转换超时时间（毫秒到秒）
	timeoutDuration := time.Duration(timeout) * time.Millisecond
	logToTaskFile(fmt.Sprintf("SimpleBash：设置超时时间: %v", timeoutDuration))
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
		logToTaskFile("SimpleBash：使用Windows cmd执行命令")
	} else {
		cmd = exec.CommandContext(ctx, "/bin/bash", "-c", command)
		logToTaskFile("SimpleBash：使用bash执行命令")
	}
	
	logToTaskFile("SimpleBash：开始执行命令")
	output, err := cmd.CombinedOutput()
	logToTaskFile(fmt.Sprintf("SimpleBash：命令执行完成，输出长度: %d", len(output)))
	
	// 格式化为JSON响应
	response := map[string]interface{}{
		"output":    string(output),
		"exit_code": 0,
		"error":     "",
	}
	
	if err != nil {
		logToTaskFile(fmt.Sprintf("SimpleBash：命令执行出错: %v", err))
		if exitError, ok := err.(*exec.ExitError); ok {
			response["exit_code"] = exitError.ExitCode()
		} else {
			response["exit_code"] = 1
		}
		response["error"] = err.Error()
	}
	
	responseJSON, _ := json.Marshal(response)
	logToTaskFile(fmt.Sprintf("SimpleBash：返回结果，JSON长度: %d", len(string(responseJSON))))
	return string(responseJSON)
}

// logTaskNetworkPerformance 记录Task网络性能数据到专门的日志文件
func logTaskNetworkPerformance(input, model string, networkDuration time.Duration, usage *general.Usage, err error) {
	// 确保log目录存在
	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.MkdirAll("log", 0755)
	}

	// 打开或创建network_performance.txt文件（与主进程共享同一个文件）
	file, err := os.OpenFile("log/network_performance.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	// 写入Task的网络性能数据
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	status := "SUCCESS"
	if err != nil {
		status = fmt.Sprintf("ERROR: %v", err)
	}
	
	// 处理Token信息
	var tokenInfo string
	if usage != nil {
		tokenInfo = fmt.Sprintf("Prompt:%d|Completion:%d|Total:%d", 
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	} else {
		tokenInfo = "Token:N/A"
	}
	
	fmt.Fprintf(file, "%s | %s | %s | Input:%d chars | Network:%v | %s | Source:TASK | %s\n",
		timestamp,
		general.ProviderOpenAI,
		model,
		len(input),
		networkDuration,
		tokenInfo,
		status,
	)
}
