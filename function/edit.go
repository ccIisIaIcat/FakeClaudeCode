package function

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// 用于跟踪Read工具的使用情况
var readHistory = make(map[string]time.Time)

func Edit(file_path string, old_string string, new_string string, replace_all bool, expected_replacements int) string {
	start := time.Now()
	
	// 记录日志
	logFile, err := os.OpenFile("./log/edit.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err == nil {
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)
		logger.Printf("Edit函数调用 - file_path: %s, replace_all: %t, expected_replacements: %d", file_path, replace_all, expected_replacements)
		defer func() {
			duration := time.Since(start)
			logger.Printf("Edit函数执行完成 - 耗时: %v", duration)
		}()
	}

	// 1. 基础验证
	if !filepath.IsAbs(file_path) {
		return "Error: file_path must be absolute"
	}

	if old_string == new_string {
		return "Error: old_string and new_string must be different"
	}

	if old_string == "" {
		return "Error: old_string cannot be empty"
	}

	// 2. 检查文件是否存在以及是否需要Read工具前置检查
	fileInfo, err := os.Stat(file_path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: File does not exist: %s", file_path)
		}
		return fmt.Sprintf("Error checking file: %v", err)
	}

	// 检查是否在最近5分钟内使用过Read工具读取此文件
	if lastRead, exists := readHistory[file_path]; exists {
		if time.Since(lastRead) > 5*time.Minute {
			if logger != nil {
				logger.Printf("警告: 文件 %s 上次读取时间超过5分钟，建议先使用Read工具", file_path)
			}
		}
	} else {
		if logger != nil {
			logger.Printf("警告: 未检测到对文件 %s 的Read操作历史，建议先使用Read工具", file_path)
		}
	}

	// 3. 处理来自Read工具的行号前缀
	cleanedOldString := cleanLineNumberPrefix(old_string)
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

	// 7. 执行替换
	var newContent string
	var actualReplacements int

	if replace_all {
		newContent = strings.ReplaceAll(contentStr, old_string, new_string)
		actualReplacements = count
	} else {
		// 只替换第一个匹配
		newContent = strings.Replace(contentStr, old_string, new_string, 1)
		actualReplacements = 1
	}

	// 8. 创建备份（如果文件较大或替换较多）
	if originalSize > 10*1024 || actualReplacements > 10 { // 文件>10KB或替换>10次时备份
		backupPath := file_path + ".backup." + time.Now().Format("20060102_150405")
		if backupErr := os.WriteFile(backupPath, content, fileInfo.Mode()); backupErr == nil {
			if logger != nil {
				logger.Printf("已创建备份文件: %s", backupPath)
			}
		}
	}

	// 9. 写入新内容
	err = os.WriteFile(file_path, []byte(newContent), fileInfo.Mode())
	if err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	// 10. 返回结果
	newSize := len(newContent)
	sizeDelta := newSize - originalSize
	result := fmt.Sprintf("Successfully made %d replacement(s) in %s. Size changed by %+d bytes (%d -> %d)", 
		actualReplacements, filepath.Base(file_path), sizeDelta, originalSize, newSize)
	
	if logger != nil {
		logger.Printf("Edit函数返回 - 成功编辑: %s", result)
	}
	return result
}

// cleanLineNumberPrefix 清理从Read工具输出中复制的行号前缀
func cleanLineNumberPrefix(text string) string {
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

// 供Read函数调用，记录文件读取历史
func RecordReadHistory(file_path string) {
	readHistory[file_path] = time.Now()
}