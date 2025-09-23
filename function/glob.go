package function

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Glob(pattern string, path string) string {
	// 记录日志
	logFile, err := os.OpenFile("./log/glob.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Glob函数调用 - pattern: %s, path: %s", pattern, path)
	}

	if strings.TrimSpace(pattern) == "" {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("Glob函数返回 - 空pattern: []")
		}
		return "[]"
	}

	start := strings.TrimSpace(path)
	if start == "" {
		start = "."
	}

	var matches []string
	err = filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(start, p)
		if err != nil {
			relPath = p
		}
		relPath = filepath.ToSlash(relPath)

		matched, err := filepath.Match(pattern, filepath.Base(p))
		if err != nil {
			return nil
		}

		if matched {
			matches = append(matches, relPath)
		}

		if strings.Contains(pattern, "**/") || strings.Contains(pattern, "/") {
			fullMatched, err := doubleStarMatch(pattern, relPath)
			if err == nil && fullMatched && !matched {
				matches = append(matches, relPath)
			}
		}

		return nil
	})

	if err != nil {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("Glob函数返回 - 遍历错误: %v", err)
		}
		return "[]"
	}

	if len(matches) == 0 {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("Glob函数返回 - 无匹配文件: []")
		}
		return "[]"
	}

	type fileInfo struct {
		path string
		time int64
	}

	var fileInfos []fileInfo
	for _, match := range matches {
		fullPath := filepath.Join(start, match)
		if info, err := os.Stat(fullPath); err == nil {
			fileInfos = append(fileInfos, fileInfo{
				path: match,
				time: info.ModTime().UnixNano(),
			})
		}
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].time > fileInfos[j].time
	})

	sortedPaths := make([]string, len(fileInfos))
	for i, fi := range fileInfos {
		sortedPaths[i] = fi.path
	}

	result, _ := json.Marshal(sortedPaths)
	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Glob函数返回 - 匹配文件数: %d, 结果: %s", len(sortedPaths), string(result))
	}
	return string(result)
}

func doubleStarMatch(pattern, path string) (bool, error) {
	if !strings.Contains(pattern, "**/") {
		return filepath.Match(pattern, path)
	}

	parts := strings.Split(pattern, "**/")
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]

		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false, nil
		}

		if suffix == "" {
			return true, nil
		}

		if suffix != "" {
			pathParts := strings.Split(path, "/")
			for i := 0; i < len(pathParts); i++ {
				remainingPath := strings.Join(pathParts[i:], "/")
				if matched, err := filepath.Match(suffix, remainingPath); err == nil && matched {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
