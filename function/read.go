package function

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Read(file_path string, offset int, limit int) string {
	// 记录日志
	logFile, err := os.OpenFile("./log/read.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Read函数调用 - file_path: %s, offset: %d, limit: %d", file_path, offset, limit)
	}

	if !filepath.IsAbs(file_path) {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("Read函数返回 - 错误: file_path must be absolute")
		}
		return "Error: file_path must be absolute"
	}

	file, err := os.Open(file_path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 1

	if offset <= 0 {
		offset = 1
	}
	if limit <= 0 {
		limit = 2000
	}

	for scanner.Scan() {
		if lineNum >= offset {
			line := scanner.Text()
			if len(line) > 2000 {
				line = line[:2000] + "... [truncated]"
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

	if len(lines) == 0 {
		return "File is empty or no lines in specified range"
	}

	result := strings.Join(lines, "\n")
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Read函数返回 - 读取行数: %d, 结果长度: %d", len(lines), len(result))
	}
	return result
}