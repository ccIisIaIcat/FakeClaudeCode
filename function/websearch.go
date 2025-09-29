package function

import (
	"log"
	"os"
	"strings"
	"time"
)

func WebSearch(query string, allowed_domains []string, blocked_domains []string) string {
	// 立即输出到标准输出，确保能看到调用
	log.Printf("*** WebSearch函数被调用 - query: %s ***", query)
	
	// 立即尝试记录日志，不管后续是否成功
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WebSearch函数发生panic: %v", r)
		}
	}()
	
	start := time.Now()
	
	// 确保log目录存在
	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.MkdirAll("log", 0755)
	}
	
	// 记录日志 - 立即创建并写入第一条日志
	logFile, err := os.OpenFile("./log/websearch.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err == nil {
		logger = log.New(logFile, "", log.LstdFlags)
		logger.Printf("=== WebSearch函数开始调用 ===")
		logger.Printf("WebSearch函数调用 - query: %s, allowed_domains: %v, blocked_domains: %v", 
			query, allowed_domains, blocked_domains)
		
		// 立即刷新日志文件
		logFile.Sync()
		logFile.Close()
		
		// 重新打开日志文件用于后续记录
		logFile, err = os.OpenFile("./log/websearch.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			defer logFile.Close()
			logger = log.New(logFile, "", log.LstdFlags)
			defer func() {
				duration := time.Since(start)
				logger.Printf("WebSearch函数执行完成 - 耗时: %v", duration)
				logger.Printf("=== WebSearch函数调用结束 ===")
				logFile.Sync()
			}()
		}
	} else {
		// 如果无法创建日志文件，输出到标准输出
		log.Printf("WebSearch: 无法创建日志文件: %v", err)
	}

	if strings.TrimSpace(query) == "" {
		if logger != nil {
			logger.Printf("WebSearch函数返回 - 错误: query is required")
		}
		return "Error: query is required"
	}

	if len(query) < 2 {
		if logger != nil {
			logger.Printf("WebSearch函数返回 - 错误: query must be at least 2 characters long")
		}
		return "Error: query must be at least 2 characters long"
	}

	// 调用TavilySearch实现真正的网络搜索
	if logger != nil {
		logger.Printf("开始调用TavilySearch进行网络搜索")
	}
	
	// 添加调用前的诊断日志
	if logger != nil {
		logger.Printf("检查TavilySearch函数是否可用...")
	}
	
	tavilyStart := time.Now()
	result := TavilySearch(query, allowed_domains, blocked_domains)
	tavilyDuration := time.Since(tavilyStart)
	
	if logger != nil {
		logger.Printf("TavilySearch调用完成 - 耗时: %v", tavilyDuration)
		logger.Printf("TavilySearch返回结果 - 长度: %d字符", len(result))
		if len(result) > 0 {
			// 输出结果的前100个字符用于调试
			preview := result
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			logger.Printf("TavilySearch结果预览: %s", preview)
		} else {
			logger.Printf("警告: TavilySearch返回空结果")
		}
	}

	// 在返回前记录返回值
	if logger != nil {
		logger.Printf("WebSearch最终返回值长度: %d字符", len(result))
		if len(result) > 0 {
			// 记录返回值的前200个字符用于调试
			preview := result
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			logger.Printf("WebSearch返回值预览: %s", preview)
		} else {
			logger.Printf("警告: WebSearch返回空结果")
		}
		logger.Printf("即将返回WebSearch结果给AI agent")
	}

	return result
}
