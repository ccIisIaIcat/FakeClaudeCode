package function

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Grep 在给定 path（文件或目录）中按正则 pattern 搜索文件内容，
// 可用 include 进行文件名模式过滤（支持简单的 * ? 以及一层 {a,b} 展开）。
// 返回：按修改时间降序排序的匹配文件相对路径 JSON 数组字符串。
func Grep(pattern string, include string, path string) string {
	writeDebug := func(msg string) {
		if f, err := os.OpenFile("./log/grep.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			f.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), msg))
			f.Close()
		}
	}
	
	writeDebug(fmt.Sprintf("输入参数 - pattern: '%s', include: '%s', path: '%s'", pattern, include, path))
	
	if strings.TrimSpace(pattern) == "" {
		writeDebug("空模式，返回空数组")
		return "[]"
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		writeDebug(fmt.Sprintf("正则编译失败: %v", err))
		return "[]"
	}

	start := strings.TrimSpace(path)
	if start == "" {
		start = "."
	}
	writeDebug(fmt.Sprintf("搜索起始路径: '%s'", start))

	// 准备 include 模式集合
	includePatterns := expandBracePattern(strings.TrimSpace(include))
	writeDebug(fmt.Sprintf("include模式: %v", includePatterns))

	matchedFiles := make(map[string]fs.FileInfo)

	// 处理单文件
	info, statErr := os.Stat(start)
	if statErr == nil && !info.IsDir() {
		if shouldInclude(start, includePatterns) && fileMatches(start, compiled) {
			matchedFiles[start] = info
		}
		return toSortedJSON(matchedFiles)
	}

	// 处理目录递归
	fileCount := 0
	checkedCount := 0
	_ = filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		// 跳过常见大目录与隐藏目录
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		fileCount++
		if !shouldInclude(p, includePatterns) {
			return nil
		}
		checkedCount++
		if fileMatches(p, compiled) {
			if fi, e := os.Stat(p); e == nil {
				matchedFiles[p] = fi
				writeDebug(fmt.Sprintf("匹配文件: %s", p))
			}
		}
		return nil
	})
	writeDebug(fmt.Sprintf("遍历完成 - 总文件数: %d, 检查文件数: %d, 匹配文件数: %d", fileCount, checkedCount, len(matchedFiles)))

	result := toSortedJSON(matchedFiles)
	writeDebug(fmt.Sprintf("最终结果: %s (匹配文件数: %d)", result, len(matchedFiles)))
	return result
}

// shouldInclude 根据 include 模式判断文件是否应被检查
func shouldInclude(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	unix := filepath.ToSlash(path)
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		// 同时对全路径与基名尝试匹配
		if ok, _ := filepath.Match(pat, unix); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, filepath.Base(unix)); ok {
			return true
		}
	}
	return false
}

// fileMatches 判断文件内容是否匹配正则
func fileMatches(path string, re *regexp.Regexp) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// 简单跳过可能的二进制文件（包含 NUL）
	if idx := strings.IndexByte(string(b), 0); idx >= 0 {
		return false
	}
	return re.Match(b)
}

// toSortedJSON 将文件集合按修改时间降序排序并返回 JSON 数组字符串
func toSortedJSON(m map[string]fs.FileInfo) string {
	if len(m) == 0 {
		return "[]"
	}
	type item struct {
		path string
		mod  int64
	}
	arr := make([]item, 0, len(m))
	for p, fi := range m {
		arr = append(arr, item{path: normalizeRel(p), mod: fi.ModTime().UnixNano()})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].mod > arr[j].mod })
	out := make([]string, len(arr))
	for i, it := range arr {
		out[i] = it.path
	}
	js, _ := json.Marshal(out)
	return string(js)
}

// normalizeRel 将路径转换为相对工作目录的形式（若可能），并统一为正斜杠
func normalizeRel(p string) string {
	wd, err := os.Getwd()
	if err == nil {
		if rel, e := filepath.Rel(wd, p); e == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(p)
}

// expandBracePattern 将如 "*.{ts,tsx}" 展开为 ["*.ts","*.tsx"]；若无大括号则返回原样数组
func expandBracePattern(p string) []string {
	p = strings.TrimSpace(p)
	if p == "" {
		return nil
	}
	start := strings.IndexByte(p, '{')
	end := strings.IndexByte(p, '}')
	if start >= 0 && end > start {
		prefix := p[:start]
		suffix := p[end+1:]
		inner := p[start+1 : end]
		parts := strings.Split(inner, ",")
		res := make([]string, 0, len(parts))
		for _, part := range parts {
			res = append(res, prefix+strings.TrimSpace(part)+suffix)
		}
		return res
	}
	return []string{p}
}
