package function

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func LS(path string, ignore []string) string {
	// 记录日志
	logFile, err := os.OpenFile("./log/ls.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("LS函数调用 - path: %s, ignore: %v", path, ignore)
	}

	if !filepath.IsAbs(path) {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("LS函数返回 - 错误: Path must be absolute")
		}
		return `{"error": "Path must be absolute"}`
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("LS函数返回 - 读取目录错误: %s", err.Error())
		}
		return `{"error": "` + err.Error() + `"}`
	}

	var result []string
	for _, entry := range entries {
		name := entry.Name()

		shouldIgnore := false
		for _, pattern := range ignore {
			if matched, _ := filepath.Match(pattern, name); matched {
				shouldIgnore = true
				break
			}
		}

		if !shouldIgnore {
			if entry.IsDir() {
				result = append(result, name+"/")
			} else {
				result = append(result, name)
			}
		}
	}

	jsonResult, _ := json.Marshal(result)
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("LS函数返回 - 文件数: %d, 结果: %s", len(result), string(jsonResult))
	}
	return string(jsonResult)
}
