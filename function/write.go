package function

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Write(file_path string, content string) string {
	start := time.Now()
	
	// 记录日志
	logFile, err := os.OpenFile("./log/write.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err == nil {
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)
		logger.Printf("Write函数调用 - file_path: %s, content_length: %d", file_path, len(content))
		defer func() {
			duration := time.Since(start)
			logger.Printf("Write函数执行完成 - 耗时: %v", duration)
		}()
	}

	// 1. 基础验证
	if !filepath.IsAbs(file_path) {
		if logger != nil {
			logger.Printf("Write函数返回 - 错误: file_path must be absolute")
		}
		return "Error: file_path must be absolute"
	}

	if content == "" {
		if logger != nil {
			logger.Printf("Write函数返回 - 错误: content cannot be empty")
		}
		return "Error: content cannot be empty"
	}

	// 2. 检查是否为现有文件，需要先使用Read工具
	fileExists := false
	var existingSize int64 = 0
	if fileInfo, err := os.Stat(file_path); err == nil {
		fileExists = true
		existingSize = fileInfo.Size()
		
		// 检查是否在最近5分钟内使用过Read工具读取此文件
		if lastRead, readExists := readHistory[file_path]; readExists {
			if time.Since(lastRead) > 5*time.Minute {
				if logger != nil {
					logger.Printf("警告: 文件 %s 上次读取时间超过5分钟，建议先使用Read工具", file_path)
				}
			}
		} else {
			if logger != nil {
				logger.Printf("警告: 未检测到对现有文件 %s 的Read操作历史，建议先使用Read工具", file_path)
			}
		}
	}

	// 3. 禁止创建文档文件（除非显式请求）
	fileName := filepath.Base(file_path)
	ext := strings.ToLower(filepath.Ext(file_path))
	
	if !fileExists && isDocumentationFile(fileName, ext) {
		errorMsg := fmt.Sprintf("Error: Creation of documentation files is restricted. File: %s. Use Edit tool for existing files or explicitly request documentation creation.", fileName)
		if logger != nil {
			logger.Printf("Write函数返回 - 文档文件创建限制: %s", errorMsg)
		}
		return errorMsg
	}

	// 4. 安全检查 - 敏感路径
	if isSensitivePath(file_path) {
		errorMsg := fmt.Sprintf("Error: Writing to sensitive system paths is not allowed: %s", file_path)
		if logger != nil {
			logger.Printf("Write函数返回 - 敏感路径限制: %s", errorMsg)
		}
		return errorMsg
	}

	// 5. 内容安全检查
	if containsSensitiveContent(content) {
		if logger != nil {
			logger.Printf("警告: 检测到可能的敏感内容")
		}
	}

	// 6. 创建目录
	dir := filepath.Dir(file_path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		errorMsg := fmt.Sprintf("Error creating directory: %v", err)
		if logger != nil {
			logger.Printf("Write函数返回 - 目录创建错误: %s", errorMsg)
		}
		return errorMsg
	}

	// 7. 权限检查
	if err := checkWritePermissions(dir); err != nil {
		errorMsg := fmt.Sprintf("Error: No write permission to directory: %v", err)
		if logger != nil {
			logger.Printf("Write函数返回 - 权限错误: %s", errorMsg)
		}
		return errorMsg
	}

	// 8. 大文件警告和备份
	contentSize := len(content)
	if fileExists && (existingSize > 50*1024 || int64(contentSize) > 50*1024) { // 50KB
		// 创建备份
		backupPath := file_path + ".backup." + time.Now().Format("20060102_150405")
		if existingContent, err := os.ReadFile(file_path); err == nil {
			if backupErr := os.WriteFile(backupPath, existingContent, 0644); backupErr == nil {
				if logger != nil {
					logger.Printf("已创建备份文件: %s", backupPath)
				}
			}
		}
	}

	// 9. 写入文件
	err = os.WriteFile(file_path, []byte(content), 0644)
	if err != nil {
		errorMsg := fmt.Sprintf("Error writing file: %v", err)
		if logger != nil {
			logger.Printf("Write函数返回 - 写入错误: %s", errorMsg)
		}
		return errorMsg
	}

	// 10. 构建结果
	operation := "created"
	if fileExists {
		operation = "overwritten"
	}

	result := fmt.Sprintf("Successfully %s %s (%d bytes)", operation, filepath.Base(file_path), contentSize)
	
	if fileExists {
		sizeDelta := int64(contentSize) - existingSize
		result += fmt.Sprintf(". Size changed by %+d bytes (%d -> %d)", sizeDelta, existingSize, contentSize)
	}

	if logger != nil {
		logger.Printf("Write函数返回 - 成功写入: %s", result)
	}
	
	return result
}

// isDocumentationFile 检查是否为文档文件
func isDocumentationFile(fileName, ext string) bool {
	// 检查文件扩展名
	docExts := []string{".md", ".markdown", ".rst", ".txt", ".doc", ".docx"}
	for _, docExt := range docExts {
		if ext == docExt {
			return true
		}
	}
	
	// 检查常见文档文件名（不区分大小写）
	lowerName := strings.ToLower(fileName)
	docNames := []string{"readme", "changelog", "license", "contributing", "authors", "todo", "roadmap"}
	for _, docName := range docNames {
		if lowerName == docName || strings.HasPrefix(lowerName, docName+".") {
			return true
		}
	}
	
	return false
}

// isSensitivePath 检查是否为敏感系统路径
func isSensitivePath(filePath string) bool {
	lowerPath := strings.ToLower(filePath)
	
	// Windows系统路径
	winSensitivePaths := []string{
		"c:\\windows\\", "c:\\program files\\", "c:\\program files (x86)\\",
		"\\system32\\", "\\syswow64\\",
	}
	
	// Unix系统路径
	unixSensitivePaths := []string{
		"/etc/", "/bin/", "/sbin/", "/usr/bin/", "/usr/sbin/",
		"/sys/", "/proc/", "/dev/", "/boot/",
	}
	
	for _, path := range winSensitivePaths {
		if strings.Contains(lowerPath, path) {
			return true
		}
	}
	
	for _, path := range unixSensitivePaths {
		if strings.HasPrefix(lowerPath, path) {
			return true
		}
	}
	
	return false
}

// containsSensitiveContent 检查内容是否包含敏感信息
func containsSensitiveContent(content string) bool {
	lowerContent := strings.ToLower(content)
	
	// 检查常见敏感信息模式
	sensitivePatterns := []string{
		"password", "secret", "token", "api_key", "private_key",
		"ssh_key", "credential", "auth_token",
	}
	
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}
	
	return false
}

// checkWritePermissions 检查目录写入权限
func checkWritePermissions(dir string) error {
	// 尝试在目录中创建临时文件来测试权限
	tempFile := filepath.Join(dir, ".write_test_"+time.Now().Format("20060102150405"))
	
	file, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	file.Close()
	
	// 清理临时文件
	os.Remove(tempFile)
	return nil
}