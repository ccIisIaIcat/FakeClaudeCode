package function

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Read(file_path string, offset int, limit int) string {
	start := time.Now()
	
	// 记录日志
	logFile, err := os.OpenFile("./log/read.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err == nil {
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)
		logger.Printf("Read函数调用 - file_path: %s, offset: %d, limit: %d", file_path, offset, limit)
		defer func() {
			duration := time.Since(start)
			logger.Printf("Read函数执行完成 - 耗时: %v", duration)
		}()
	}

	// 1. 基础验证
	if !filepath.IsAbs(file_path) {
		if logger != nil {
			logger.Printf("Read函数返回 - 错误: file_path must be absolute")
		}
		return "Error: file_path must be absolute"
	}

	// 2. 检查文件信息
	fileInfo, err := os.Stat(file_path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: File does not exist: %s", file_path)
		}
		return fmt.Sprintf("Error checking file: %v", err)
	}

	// 3. 检查文件类型和大小
	ext := strings.ToLower(filepath.Ext(file_path))
	fileSize := fileInfo.Size()
	
	// 记录读取历史（供Edit函数使用）
	RecordReadHistory(file_path)
	
	// 4. 处理Jupyter Notebook文件
	if ext == ".ipynb" {
		if logger != nil {
			logger.Printf("检测到Jupyter Notebook文件: %s", file_path)
		}
		return fmt.Sprintf("Note: This is a Jupyter Notebook file (.ipynb). Consider using NotebookRead tool instead of Read for better handling of notebook structure.\n\nFile: %s\nSize: %d bytes\nLast modified: %s", 
			file_path, fileSize, fileInfo.ModTime().Format("2006-01-02 15:04:05"))
	}

	// 5. 处理多媒体文件（图片、截图等）
	if isMultimediaFile(ext) {
		if logger != nil {
			logger.Printf("检测到多媒体文件: %s", file_path)
		}
		return handleMultimediaFile(file_path, fileSize, fileInfo.ModTime(), logger)
	}

	// 6. 处理二进制文件
	if isBinaryFile(file_path, ext) {
		if logger != nil {
			logger.Printf("检测到二进制文件: %s", file_path)
		}
		return fmt.Sprintf("Warning: This appears to be a binary file (.%s).\n\nFile: %s\nSize: %d bytes\nLast modified: %s\n\nBinary files cannot be displayed as text. File extension suggests this is a binary format.",
			strings.TrimPrefix(ext, "."), file_path, fileSize, fileInfo.ModTime().Format("2006-01-02 15:04:05"))
	}

	// 7. 处理大文件警告
	if fileSize > 1024*1024 { // 1MB
		if logger != nil {
			logger.Printf("警告: 大文件 %s (%d bytes)", file_path, fileSize)
		}
	}

	// 8. 读取文本文件
	file, err := os.Open(file_path)
	if err != nil {
		return fmt.Sprintf("Error opening file: %v", err)
	}
	defer file.Close()

	// 9. 设置默认参数
	if offset <= 0 {
		offset = 1
	}
	if limit <= 0 {
		limit = 2000
	}

	// 10. 逐行读取
	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 1
	totalLines := 0

	// 计算总行数（用于显示进度）
	tempFile, _ := os.Open(file_path)
	if tempFile != nil {
		tempScanner := bufio.NewScanner(tempFile)
		for tempScanner.Scan() {
			totalLines++
		}
		tempFile.Close()
	}

	for scanner.Scan() {
		if lineNum >= offset {
			line := scanner.Text()
			
			// 行长度截断
			if len(line) > 2000 {
				line = line[:2000] + "... [truncated - line too long]"
			}
			
			lines = append(lines, fmt.Sprintf("%6d\t%s", lineNum, line))
			
			if len(lines) >= limit {
				break
			}
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	// 11. 处理空文件情况
	if len(lines) == 0 {
		if totalLines == 0 {
			// 空文件警告（模拟system reminder）
			emptyWarning := fmt.Sprintf("SYSTEM REMINDER: File %s exists but has empty contents (0 bytes, 0 lines).", filepath.Base(file_path))
			if logger != nil {
				logger.Printf("空文件警告: %s", file_path)
			}
			return emptyWarning
		} else {
			return fmt.Sprintf("No lines in specified range (offset: %d, limit: %d). File has %d total lines.", offset, limit, totalLines)
		}
	}

	// 12. 构建结果
	result := strings.Join(lines, "\n")
	
	// 13. 添加文件信息头部
	header := fmt.Sprintf("File: %s | Size: %d bytes | Lines: %d | Showing lines %d-%d",
		filepath.Base(file_path), fileSize, totalLines, offset, offset+len(lines)-1)
	
	if len(lines) >= limit {
		header += fmt.Sprintf(" | Truncated (use offset/limit for more)")
	}
	
	result = header + "\n" + strings.Repeat("=", len(header)) + "\n" + result

	if logger != nil {
		logger.Printf("Read函数返回 - 读取行数: %d/%d, 结果长度: %d", len(lines), totalLines, len(result))
	}
	
	return result
}

// isMultimediaFile 检查是否为多媒体文件
func isMultimediaFile(ext string) bool {
	multimediaExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg",
		".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv",
		".mp3", ".wav", ".flac", ".aac", ".ogg",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
	}
	
	for _, mediaExt := range multimediaExts {
		if ext == mediaExt {
			return true
		}
	}
	return false
}

// isBinaryFile 检查是否为二进制文件
func isBinaryFile(file_path, ext string) bool {
	// 已知二进制扩展名
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib", ".bin", ".dat",
		".zip", ".rar", ".7z", ".tar", ".gz",
		".class", ".jar", ".war",
	}
	
	for _, binExt := range binaryExts {
		if ext == binExt {
			return true
		}
	}
	
	// 读取文件开头检查是否包含大量二进制字符
	file, err := os.Open(file_path)
	if err != nil {
		return false
	}
	defer file.Close()
	
	buffer := make([]byte, 512) // 读取前512字节
	n, err := file.Read(buffer)
	if err != nil || n == 0 {
		return false
	}
	
	// 如果包含过多非文本字符，认为是二进制文件
	nonTextCount := 0
	for i := 0; i < n; i++ {
		b := buffer[i]
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			nonTextCount++
		}
	}
	
	return float64(nonTextCount)/float64(n) > 0.1 // 超过10%非文本字符
}

// handleMultimediaFile 处理多媒体文件
func handleMultimediaFile(file_path string, fileSize int64, modTime time.Time, logger *log.Logger) string {
	ext := strings.ToLower(filepath.Ext(file_path))
	
	// 对于图片文件，尝试提供base64编码（小文件）
	if isImageFile(ext) && fileSize < 1024*1024 { // 1MB以下的图片
		if data, err := os.ReadFile(file_path); err == nil {
			b64 := base64.StdEncoding.EncodeToString(data)
			mimeType := getMimeType(ext)
			
			result := fmt.Sprintf("Image File Detected: %s\n", filepath.Base(file_path))
			result += fmt.Sprintf("Size: %d bytes | Last modified: %s\n", fileSize, modTime.Format("2006-01-02 15:04:05"))
			result += fmt.Sprintf("MIME Type: %s\n", mimeType)
			result += "Note: This tool can display images. The image content is encoded below:\n\n"
			result += fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
			
			if logger != nil {
				logger.Printf("返回图片文件base64编码: %s (%d bytes)", file_path, fileSize)
			}
			return result
		}
	}
	
	// 其他多媒体文件的通用处理
	return fmt.Sprintf("Multimedia File Detected: %s\n\nFile: %s\nType: %s\nSize: %d bytes\nLast modified: %s\n\nThis file type cannot be displayed as text content. Use appropriate tools for this file type.",
		filepath.Base(file_path), file_path, strings.TrimPrefix(ext, "."), fileSize, modTime.Format("2006-01-02 15:04:05"))
}

// isImageFile 检查是否为图片文件
func isImageFile(ext string) bool {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// getMimeType 获取MIME类型
func getMimeType(ext string) string {
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg", 
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
	}
	
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}