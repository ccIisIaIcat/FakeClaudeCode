package function

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Edit(file_path string, old_string string, new_string string, expected_replacements int) string {
	// 记录日志
	logFile, err := os.OpenFile("./log/edit.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Edit函数调用 - file_path: %s, expected_replacements: %d", file_path, expected_replacements)
	}
	if !filepath.IsAbs(file_path) {
		return "Error: file_path must be absolute"
	}

	if old_string == new_string {
		return "Error: old_string and new_string must be different"
	}

	content, err := os.ReadFile(file_path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	contentStr := string(content)
	count := strings.Count(contentStr, old_string)

	if expected_replacements <= 0 {
		expected_replacements = 1
	}

	if count != expected_replacements {
		return fmt.Sprintf("Error: expected %d replacements but found %d occurrences", expected_replacements, count)
	}

	if count == 0 {
		return "Error: old_string not found in file"
	}

	newContent := strings.ReplaceAll(contentStr, old_string, new_string)

	err = os.WriteFile(file_path, []byte(newContent), 0644)
	if err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	result := fmt.Sprintf("Successfully made %d replacement(s) in %s", count, filepath.Base(file_path))
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Edit函数返回 - 成功编辑: %s", result)
	}
	return result
}