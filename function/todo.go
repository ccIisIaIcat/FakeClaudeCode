package function

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
)

// TodoItem 待办事项结构
type TodoItem struct {
	ID       string `json:"id"`       // 字符串ID
	Content  string `json:"content"`  // 任务内容
	Status   string `json:"status"`   // pending, in_progress, completed
	Priority string `json:"priority"` // high, medium, low
}

// TodoList 待办事项列表
type TodoList struct {
	Items []TodoItem `json:"items"`
	mu    sync.RWMutex
}

var globalTodoList = &TodoList{
	Items: make([]TodoItem, 0),
}

const todoFilePath = "./log/todolist.json"

var todoLogger *log.Logger
var todoLogFile *os.File

func init() {
	// 将日志写入log目录下的 todowrite.txt（追加）
	os.MkdirAll("./log", 0755)
	f, err := os.OpenFile("./log/todowrite.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// 打开失败则退回到丢弃，避免程序中断
		todoLogger = log.New(io.Discard, "[TODO] ", log.LstdFlags)
		return
	}
	todoLogFile = f
	todoLogger = log.New(f, "[TODO] ", log.LstdFlags)
}

// TodoWriteRequest 写入请求结构
type TodoWriteRequest struct {
	Todos []TodoItem `json:"todos"`
}

// TodoRead 读取当前所有待办事项
func TodoRead() string {
	// 记录日志
	logFile, err := os.OpenFile("./log/todoread.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("TodoRead函数调用")
	}

	globalTodoList.mu.RLock()
	defer globalTodoList.mu.RUnlock()

	if len(globalTodoList.Items) == 0 {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("TodoRead函数返回 - 暂无待办事项")
		}
		return "暂无待办事项"
	}

	data, err := json.Marshal(globalTodoList.Items)
	if err != nil {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("TodoRead函数返回 - 序列化错误: %v", err)
		}
		return "错误：无法序列化待办事项"
	}

	result := string(data)
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("TodoRead函数返回 - 成功读取%d项待办事项", len(globalTodoList.Items))
	}
	return result
}

// TodoWrite 写入完整的待办事项列表
func TodoWrite(requestJSON string) string {
	todoLogger.Printf("[DEBUG] TodoWrite开始执行，参数长度: %d", len(requestJSON))

	// 首先尝试解析为包含request字段的格式
	type WrappedRequest struct {
		Request string `json:"request"`
	}

	var wrapped WrappedRequest
	var req TodoWriteRequest

	todoLogger.Printf("[DEBUG] 开始解析JSON参数")
	todoLogger.Printf("[DEBUG] 原始JSON: %s", requestJSON)

	// 先尝试解析为包装格式
	if err := json.Unmarshal([]byte(requestJSON), &wrapped); err == nil && wrapped.Request != "" {
		todoLogger.Printf("[DEBUG] 检测到包装格式，内部request长度: %d", len(wrapped.Request))
		// 如果是包装格式，解析内部的request字符串
		if err := json.Unmarshal([]byte(wrapped.Request), &req); err != nil {
			todoLogger.Printf("[DEBUG] 解析内部request失败: %v", err)
			return "❌ 错误: 无效的内部请求格式\n\n示例: {\"request\":\"{\\\"todos\\\":[{\\\"id\\\":\\\"1\\\",\\\"content\\\":\\\"任务描述\\\",\\\"status\\\":\\\"pending\\\",\\\"priority\\\":\\\"medium\\\"}]}\"}"
		}
		todoLogger.Printf("[DEBUG] 成功解析包装格式，todos数量: %d", len(req.Todos))
	} else {
		todoLogger.Printf("[DEBUG] 尝试直接解析为TodoWriteRequest格式")
		// 尝试直接解析为TodoWriteRequest格式
		if err := json.Unmarshal([]byte(requestJSON), &req); err != nil {
			todoLogger.Printf("[DEBUG] 直接解析TodoWriteRequest失败: %v", err)

			// 尝试解析为直接的TodoItem数组
			todoLogger.Printf("[DEBUG] 尝试解析为TodoItem数组格式")
			var directItems []TodoItem
			if err2 := json.Unmarshal([]byte(requestJSON), &directItems); err2 == nil {
				todoLogger.Printf("[DEBUG] 成功解析为TodoItem数组，items数量: %d", len(directItems))
				req.Todos = directItems
			} else {
				todoLogger.Printf("[DEBUG] 解析为TodoItem数组也失败: %v", err2)

				// 尝试修复常见的JSON格式问题（数字ID转换为字符串）
				todoLogger.Printf("[DEBUG] 尝试修复JSON格式问题")
				fixedJSON := fixJSONFormat(requestJSON)
				todoLogger.Printf("[DEBUG] 修复后的JSON: %s", fixedJSON)

				if err3 := json.Unmarshal([]byte(fixedJSON), &directItems); err3 == nil {
					todoLogger.Printf("[DEBUG] 修复后成功解析为TodoItem数组，items数量: %d", len(directItems))
					req.Todos = directItems
				} else {
					todoLogger.Printf("[DEBUG] 修复后仍然解析失败: %v", err3)

					// 作为纯文本行进行预处理：每行一个任务，去掉 1. / 1) 等编号前缀
					todoLogger.Printf("[DEBUG] 尝试按纯文本行解析")
					lines := splitNonEmptyLines(requestJSON)
					if len(lines) == 0 {
						return "❌ 错误: 无效的请求格式\n\n支持格式:\n1. {\"todos\":[{\"id\":\"1\",\"content\":\"任务描述\",\"status\":\"pending\",\"priority\":\"medium\"}]}\n2. [{\"id\":\"1\",\"content\":\"任务描述\",\"status\":\"pending\",\"priority\":\"medium\"}]\n3. 纯文本每行一个任务（支持行首 1. / 1) / - / * 等前缀）"
					}
					var items []TodoItem
					for i, raw := range lines {
						content := stripListPrefix(raw)
						if content == "" {
							continue
						}
						items = append(items, TodoItem{
							ID:       fmt.Sprintf("%d", i+1),
							Content:  content,
							Status:   "pending",
							Priority: "medium",
						})
					}
					todoLogger.Printf("[DEBUG] 按纯文本行解析得到 items: %d", len(items))
					req.Todos = items
				}
			}
		} else {
			todoLogger.Printf("[DEBUG] 成功直接解析，todos数量: %d", len(req.Todos))
		}
	}

	todoLogger.Printf("[DEBUG] 开始获取锁")
	globalTodoList.mu.Lock()
	defer globalTodoList.mu.Unlock()
	todoLogger.Printf("[DEBUG] 成功获取锁")

	// 验证数据
	todoLogger.Printf("[DEBUG] 开始验证todos数据")
	if err := validateTodos(req.Todos); err != nil {
		todoLogger.Printf("[DEBUG] 验证失败: %v", err)
		return fmt.Sprintf("❌ 错误: %s", err.Error())
	}
	todoLogger.Printf("[DEBUG] 数据验证通过")

	// 更新全局列表
	todoLogger.Printf("[DEBUG] 开始更新全局列表")
	globalTodoList.Items = req.Todos
	todoLogger.Printf("[DEBUG] 全局列表更新完成，当前items数量: %d", len(globalTodoList.Items))

	// 直接格式化结果，避免死锁
	todoLogger.Printf("[DEBUG] 开始生成返回结果")
	result := formatTodoList(globalTodoList.Items)
	todoLogger.Printf("[DEBUG] TodoWrite执行完成，返回结果长度: %d", len(result))
	return result
}

// validateTodos 验证待办事项列表
func validateTodos(todos []TodoItem) error {
	inProgressCount := 0
	idMap := make(map[string]bool)

	for _, item := range todos {
		// 检查ID唯一性
		if item.ID == "" {
			return fmt.Errorf("待办事项ID不能为空")
		}
		if idMap[item.ID] {
			return fmt.Errorf("发现重复的ID: %s", item.ID)
		}
		idMap[item.ID] = true

		// 检查内容
		if item.Content == "" {
			return fmt.Errorf("待办事项内容不能为空")
		}

		// 检查状态
		switch item.Status {
		case "pending", "completed":
			// 正常状态
		case "in_progress":
			inProgressCount++
			if inProgressCount > 1 {
				return fmt.Errorf("只能有一个任务处于进行中状态")
			}
		default:
			return fmt.Errorf("无效的状态: %s，只支持 pending, in_progress, completed", item.Status)
		}

		// 检查优先级
		switch item.Priority {
		case "high", "medium", "low":
			// 正常优先级
		default:
			return fmt.Errorf("无效的优先级: %s，只支持 high, medium, low", item.Priority)
		}
	}

	return nil
}

// formatTodoList 格式化待办事项列表（无锁版本，供内部使用）
func formatTodoList(items []TodoItem) string {
	todoLogger.Printf("[DEBUG] formatTodoList开始执行，items数量: %d", len(items))

	if len(items) == 0 {
		todoLogger.Printf("[DEBUG] 没有items，返回空消息")
		return "暂无待办事项"
	}

	todoLogger.Printf("[DEBUG] 开始构建结果字符串")
	var result strings.Builder

	// 按创建顺序显示所有任务
	todoLogger.Printf("[DEBUG] 开始遍历items")
	for i, item := range items {
		todoLogger.Printf("[DEBUG] 处理item %d: ID=%s, Status=%s", i, item.ID, item.Status)
		var statusIcon, taskLine string

		switch item.Status {
		case "pending":
			statusIcon = "☐"
			taskLine = fmt.Sprintf("%s %s", statusIcon, item.Content)
		case "in_progress":
			statusIcon = "☐"
			// 正在执行的任务用特殊标记
			taskLine = fmt.Sprintf("%s ▶ %s", statusIcon, item.Content)
		case "completed":
			statusIcon = "☑"
			// 已完成的任务
			taskLine = fmt.Sprintf("%s %s", statusIcon, item.Content)
		}

		result.WriteString(taskLine)
		result.WriteString("\n")
		todoLogger.Printf("[DEBUG] 完成item %d处理", i)
	}

	// 显示统计信息
	todoLogger.Printf("[DEBUG] 开始计算统计信息")
	pending := 0
	inProgress := 0
	completed := 0
	for _, item := range items {
		switch item.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		}
	}

	todoLogger.Printf("[DEBUG] 统计完成: pending=%d, inProgress=%d, completed=%d", pending, inProgress, completed)
	result.WriteString(fmt.Sprintf("\n进度: %d/%d 已完成 | %d 进行中 | %d 待处理",
		completed, len(items), inProgress, pending))

	finalResult := result.String()
	todoLogger.Printf("[DEBUG] formatTodoList执行完成，结果长度: %d", len(finalResult))
	return finalResult
}

// listTodos 列出所有待办事项
func listTodos() string {
	todoLogger.Printf("[DEBUG] listTodos开始执行")

	todoLogger.Printf("[DEBUG] 尝试获取读锁")
	globalTodoList.mu.RLock()
	items := make([]TodoItem, len(globalTodoList.Items))
	copy(items, globalTodoList.Items)
	globalTodoList.mu.RUnlock()
	todoLogger.Printf("[DEBUG] 成功获取读锁并复制数据，items数量: %d", len(items))

	// 使用无锁版本的格式化函数
	return formatTodoList(items)
}

// ListTodosFormatted 返回格式化的TodoList用于UI显示
func ListTodosFormatted() string {
	return listTodos()
}

// fixJSONFormat 修复常见的JSON格式问题
func fixJSONFormat(jsonStr string) string {
	// 修复数字ID：将 "id": 123 转换为 "id": "123"
	re := regexp.MustCompile(`"id"\s*:\s*(\d+(\.\d+)?)`)
	fixed := re.ReplaceAllString(jsonStr, `"id": "$1"`)

	// 确保priority字段有默认值
	if !strings.Contains(fixed, "priority") {
		// 如果没有priority字段，添加默认值
		re2 := regexp.MustCompile(`("status"\s*:\s*"[^"]*")`)
		fixed = re2.ReplaceAllString(fixed, `$1, "priority": "medium"`)
	}

	return fixed
}

// splitNonEmptyLines 将输入按行拆分并去除空行与围栏```json等
func splitNonEmptyLines(s string) []string {
	// 去除可能的```json/```围栏
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")

	var lines []string
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			lines = append(lines, t)
		}
	}
	return lines
}

// stripListPrefix 去掉行首编号或项目符号，例如: "1. ", "1) ", "- ", "* "
func stripListPrefix(s string) string {
	re := regexp.MustCompile(`^\s*(?:[-*+]\s+|\d+[\.)]\s+)`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}
