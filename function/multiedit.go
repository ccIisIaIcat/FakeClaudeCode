package function

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type EditOperation struct {
	OldString            string `json:"old_string"`
	NewString            string `json:"new_string"`
	ExpectedReplacements int    `json:"expected_replacements"`
}

type MultiEditRequest struct {
	FilePath string          `json:"file_path"`
	Edits    []EditOperation `json:"edits"`
}

func MultiEdit(file_path string, edits []EditOperation) string {
	// 记录日志
	logFile, err := os.OpenFile("./log/multiedit.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("MultiEdit函数调用 - file_path: %s, edits_count: %d", file_path, len(edits))
	}
	if !filepath.IsAbs(file_path) {
		return "Error: file_path must be absolute"
	}

	if len(edits) == 0 {
		return "Error: no edits provided"
	}

	content, err := os.ReadFile(file_path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	currentContent := string(content)
	totalReplacements := 0

	for i, edit := range edits {
		if edit.OldString == edit.NewString {
			return fmt.Sprintf("Error in edit %d: old_string and new_string must be different", i+1)
		}

		count := strings.Count(currentContent, edit.OldString)
		expectedReplacements := edit.ExpectedReplacements
		if expectedReplacements <= 0 {
			expectedReplacements = 1
		}

		if count != expectedReplacements {
			return fmt.Sprintf("Error in edit %d: expected %d replacements but found %d occurrences", i+1, expectedReplacements, count)
		}

		if count == 0 {
			return fmt.Sprintf("Error in edit %d: old_string not found in file", i+1)
		}

		currentContent = strings.ReplaceAll(currentContent, edit.OldString, edit.NewString)
		totalReplacements += count
	}

	err = os.WriteFile(file_path, []byte(currentContent), 0644)
	if err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	result := fmt.Sprintf("Successfully made %d total replacement(s) across %d edit(s) in %s", totalReplacements, len(edits), filepath.Base(file_path))
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("MultiEdit函数返回 - 成功编辑: %s", result)
	}
	return result
}
