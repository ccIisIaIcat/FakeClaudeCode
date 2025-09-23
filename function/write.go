package function

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func Write(file_path string, content string) string {
	// 记录日志
	logFile, err := os.OpenFile("./log/write.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Write函数调用 - file_path: %s, content_length: %d", file_path, len(content))
	}
	if !filepath.IsAbs(file_path) {
		return "Error: file_path must be absolute"
	}

	dir := filepath.Dir(file_path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err)
	}

	err = os.WriteFile(file_path, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	result := fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), filepath.Base(file_path))
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Write函数返回 - 成功写入: %s", result)
	}
	return result
}