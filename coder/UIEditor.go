package coder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Editor 带UI确认的编辑方法
func (lc *LukatinCode) Editor(file_path string, old_string string, new_string string, replace_all bool, expected_replacements int) string {
	start := time.Now()

	// 记录日志
	logFile, err := os.OpenFile("./log/edit.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err == nil {
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)
		logger.Printf("Editor函数调用 - file_path: %s, replace_all: %t, expected_replacements: %d", file_path, replace_all, expected_replacements)
		defer func() {
			duration := time.Since(start)
			logger.Printf("Editor函数执行完成 - 耗时: %v", duration)
		}()
	}

	// 1. 验证参数
	if old_string == new_string {
		return "Error: old_string and new_string are identical"
	}

	if old_string == "" {
		return "Error: old_string cannot be empty"
	}

	// 2. 验证文件路径
	absPath, err := filepath.Abs(file_path)
	if err != nil {
		return fmt.Sprintf("Error resolving absolute path: %v", err)
	}
	file_path = absPath

	fileInfo, err := os.Stat(file_path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: file does not exist: %s", file_path)
		}
		return fmt.Sprintf("Error checking file: %v", err)
	}

	// 3. 处理来自Read工具的行号前缀
	cleanedOldString := lc.cleanLineNumberPrefix(old_string)
	if cleanedOldString != old_string {
		if logger != nil {
			logger.Printf("检测并清理了行号前缀: %q -> %q", old_string, cleanedOldString)
		}
		old_string = cleanedOldString
	}

	// 4. 读取文件内容
	content, err := os.ReadFile(file_path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	originalSize := len(content)
	contentStr := string(content)

	// 5. 计算匹配次数
	count := strings.Count(contentStr, old_string)
	if count == 0 {
		return fmt.Sprintf("Error: old_string not found in file. Searched for: %q", old_string)
	}

	// 6. 验证期望的替换次数
	if expected_replacements <= 0 {
		if replace_all {
			expected_replacements = count // replace_all时，期望替换所有匹配
		} else {
			expected_replacements = 1 // 默认期望替换1次
		}
	}

	if !replace_all && count != expected_replacements {
		return fmt.Sprintf("Error: Expected %d replacements but found %d occurrences. Use replace_all=true to replace all %d occurrences", expected_replacements, count, count)
	}

	// 7. UI确认逻辑 - 生成预览内容
	var newContentStr string
	if replace_all {
		newContentStr = strings.ReplaceAll(contentStr, old_string, new_string)
	} else {
		newContentStr = strings.Replace(contentStr, old_string, new_string, expected_replacements)
	}

	// 8. 请求用户确认
	operation := "edit"
	if replace_all {
		operation = "edit (replace_all)"
	}

	if logger != nil {
		logger.Printf("请求用户确认修改 - 文件: %s", file_path)
	}

	// 使用内置的BubbleTUI显示确认对话框
	confirmed := lc.requestEditConfirmation(file_path, contentStr, newContentStr, operation)
	if !confirmed {
		if logger != nil {
			logger.Printf("用户取消修改操作")
		}
		return "Edit operation cancelled by user"
	}

	if logger != nil {
		logger.Printf("用户确认修改，继续执行")
	}

	// 9. 执行替换
	var newContent string
	var actualReplacements int

	if replace_all {
		newContent = strings.ReplaceAll(contentStr, old_string, new_string)
		actualReplacements = count
	} else {
		newContent = strings.Replace(contentStr, old_string, new_string, expected_replacements)
		actualReplacements = expected_replacements
	}

	// 10. 写入文件
	err = os.WriteFile(file_path, []byte(newContent), fileInfo.Mode())
	if err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	// 11. 返回结果
	newSize := len(newContent)
	sizeDelta := newSize - originalSize
	result := fmt.Sprintf("Successfully made %d replacement(s) in %s. Size changed by %+d bytes (%d -> %d)",
		actualReplacements, filepath.Base(file_path), sizeDelta, originalSize, newSize)

	if logger != nil {
		logger.Printf("Editor函数返回 - 成功编辑: %s", result)
	}
	return result
}

// requestEditConfirmation 请求用户确认编辑
func (lc *LukatinCode) requestEditConfirmation(filePath, oldContent, newContent, operation string) bool {
	if lc.BubbleTUI == nil {
		// 如果没有UI，默认确认
		return true
	}

	// 生成唯一的change ID
	changeId := fmt.Sprintf("change_%d", time.Now().UnixNano())

	// 发送代码修改消息到UI
	if lc.BubbleTUI.program != nil {
		lc.BubbleTUI.program.Send(codeChangeMsg{
			filePath:    filePath,
			oldContent:  oldContent,
			newContent:  newContent,
			operation:   operation,
			needConfirm: true,
			changeId:    changeId,
		})

		// 创建响应channel并等待确认
		responseChan := make(chan bool, 1)
		lc.BubbleTUI.responseChannels[changeId] = responseChan

		// 阻塞等待用户确认
		confirmed := <-responseChan

		// 清理
		delete(lc.BubbleTUI.responseChannels, changeId)
		delete(lc.BubbleTUI.pendingChanges, changeId)

		return confirmed
	}

	// 默认确认
	return true
}

// cleanLineNumberPrefix 清理从Read工具输出中复制的行号前缀
func (lc *LukatinCode) cleanLineNumberPrefix(text string) string {
	// 匹配格式: "  123\t内容" 或 " 123\t内容"
	lineNumPattern := regexp.MustCompile(`^\s*\d+\t`)
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		cleanedLine := lineNumPattern.ReplaceAllString(line, "")
		cleanedLines = append(cleanedLines, cleanedLine)
	}

	return strings.Join(cleanedLines, "\n")
}
