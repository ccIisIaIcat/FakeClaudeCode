package coder

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Bash 使用持久化Shell执行bash命令
func (lc *LukatinCode) Bash(command string, description string, timeout int) string {
	// 记录函数开始时间
	functionStart := time.Now()

	// 记录到bash专用日志文件
	lc.logToBashFile(fmt.Sprintf("Bash函数调用 - command: %s, description: %s, timeout: %d", command, description, timeout))
	lc.Logger.Printf("Bash函数调用开始 - command: %s, timeout: %d", command, timeout)

	// 如果没有提供timeout，使用默认值
	if timeout == 0 {
		timeout = 120000 // 默认2分钟超时
	}

	// 在UI中显示description（如果提供了）
	if description != "" {
		lc.logToBashFile(">>" + description)
		if lc.BubbleTUI != nil && lc.BubbleTUI.program != nil {
			// 发送description作为工具描述消息显示
			lc.BubbleTUI.program.Send(toolDescriptionMsg{
				description: description,
			})
		}
	}

	// 记录内部执行开始时间
	internalStart := time.Now()
	if lc.Logger != nil {
		lc.Logger.Printf("准备调用bashInternal - command: %s", command)
	}
	lc.logToBashFile(fmt.Sprintf("准备调用bashInternal - command: %s", command))

	result := lc.bashInternal(command, timeout)

	lc.Logger.Printf("bashInternal调用完成")
	lc.logToBashFile("bashInternal调用完成")
	internalDuration := time.Since(internalStart)

	// 计算总函数执行时间
	functionDuration := time.Since(functionStart)

	// 记录执行时间
	lc.Logger.Printf("Bash函数执行完成 - 总耗时: %v, 内部执行耗时: %v", functionDuration, internalDuration)
	lc.logToBashFile(fmt.Sprintf("Bash函数返回 - 总耗时: %v, 内部执行耗时: %v, 输出长度: %d", functionDuration, internalDuration, len(result)))

	return result
}

// bashInternal 内部实现
func (lc *LukatinCode) bashInternal(command string, timeout int) string {
	lc.Logger.Printf("bashInternal开始执行 - 命令: %s", command)
	lc.logToBashFile(fmt.Sprintf("bashInternal开始执行 - 命令: %s", command))

	if command == "" {
		lc.Logger.Println("bashInternal - 命令为空")
		return `{"error": "Command is required", "exit_code": -1}`
	}

	// 检查PersistentShell是否正在运行
	lc.Logger.Printf("检查PersistentShell状态 - isNil: %v", lc.PersistentShell == nil)
	if lc.PersistentShell != nil {
		lc.Logger.Printf("PersistentShell.IsRunning(): %v", lc.PersistentShell.IsRunning())
	}

	if lc.PersistentShell == nil || !lc.PersistentShell.IsRunning() {
		lc.Logger.Println("PersistentShell未运行，尝试重新启动")
		lc.logToBashFile("PersistentShell未运行，尝试重新启动")

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
	lc.Logger.Printf("准备调用PersistentShell.ExecuteCommand")
	lc.logToBashFile("准备调用PersistentShell.ExecuteCommand")

	output, err := lc.PersistentShell.ExecuteCommand(command)

	lc.Logger.Printf("PersistentShell.ExecuteCommand返回")
	lc.logToBashFile("PersistentShell.ExecuteCommand返回")

	if err != nil {
		lc.Logger.Printf("命令执行失败: %v", err)
		lc.logToBashFile(fmt.Sprintf("命令执行失败: %v", err))
		return fmt.Sprintf(`{"error": "%s", "exit_code": 1, "output": ""}`, err.Error())
	}

	lc.Logger.Printf("命令执行成功，输出长度: %d", len(output))
	lc.logToBashFile(fmt.Sprintf("命令执行成功，输出长度: %d", len(output)))

	// 格式化为JSON响应
	response := map[string]interface{}{
		"output":    output,
		"error":     "",
		"exit_code": 0,
	}

	responseJSON, _ := json.Marshal(response)
	return string(responseJSON)
}

// logToBashFile 记录bash函数专用日志
func (lc *LukatinCode) logToBashFile(message string) {
	// 确保log目录存在
	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.MkdirAll("log", 0755)
	}

	// 打开或创建bash.txt文件
	file, err := os.OpenFile("log/bash.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		if lc.Logger != nil {
			lc.Logger.Printf("无法打开bash.txt文件: %v", err)
		}
		return
	}
	defer file.Close()

	// 写入时间戳和消息
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	fmt.Fprintf(file, "%s %s\n", timestamp, message)
}
