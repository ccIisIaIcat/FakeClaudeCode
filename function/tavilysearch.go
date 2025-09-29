package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// TavilyConfig Tavily配置结构体
type TavilyConfig struct {
	BaseUrl string `yaml:"BaseUrl"`
	APIKEY  string `yaml:"APIKEY"`
}

// ExtendedConfig 扩展配置结构体
type ExtendedConfig struct {
	Tavily TavilyConfig `yaml:"Tavily"`
}

// TavilySearchRequest 请求结构体
type TavilySearchRequest struct {
	APIKey         string   `json:"api_key"`
	Query          string   `json:"query"`
	SearchDepth    string   `json:"search_depth,omitempty"`    // basic or advanced
	IncludeDomains []string `json:"include_domains,omitempty"` // allowed_domains
	ExcludeDomains []string `json:"exclude_domains,omitempty"` // blocked_domains
	MaxResults     int      `json:"max_results,omitempty"`     // 默认5
	IncludeAnswer  bool     `json:"include_answer,omitempty"`  // 是否包含AI摘要
	IncludeRawContent bool  `json:"include_raw_content,omitempty"` // 是否包含原始内容
}

// TavilySearchResponse 响应结构体
type TavilySearchResponse struct {
	Answer  string `json:"answer"`
	Query   string `json:"query"`
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

// loadTavilyConfig 加载Tavily配置
func loadTavilyConfig() (*TavilyConfig, error) {
	data, err := os.ReadFile("./LLMConfig.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config ExtendedConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config.Tavily, nil
}

func TavilySearch(query string, allowed_domains []string, blocked_domains []string) string {
	start := time.Now()
	
	// 记录日志
	logFile, err := os.OpenFile("./log/tavilysearch.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err == nil {
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)
		logger.Printf("TavilySearch函数调用 - query: %s, allowed_domains: %v, blocked_domains: %v", 
			query, allowed_domains, blocked_domains)
		defer func() {
			duration := time.Since(start)
			logger.Printf("TavilySearch函数执行完成 - 耗时: %v", duration)
		}()
	}

	if strings.TrimSpace(query) == "" {
		if logger != nil {
			logger.Printf("TavilySearch函数返回 - 错误: query is required")
		}
		return "Error: query is required"
	}

	if len(query) < 2 {
		if logger != nil {
			logger.Printf("TavilySearch函数返回 - 错误: query must be at least 2 characters long")
		}
		return "Error: query must be at least 2 characters long"
	}

	// 加载Tavily配置
	tavilyConfig, err := loadTavilyConfig()
	if err != nil {
		if logger != nil {
			logger.Printf("配置加载失败: %v", err)
		}
		return fmt.Sprintf("Failed to load Tavily config: %v", err)
	}

	// 检查Tavily配置
	if tavilyConfig.APIKEY == "" {
		if logger != nil {
			logger.Printf("Tavily API密钥未配置")
		}
		return "Error: Tavily API key not configured"
	}

	// 构建请求
	request := TavilySearchRequest{
		APIKey:         tavilyConfig.APIKEY,
		Query:          query,
		SearchDepth:    "basic", // 使用基础搜索
		IncludeDomains: allowed_domains,
		ExcludeDomains: blocked_domains,
		MaxResults:     10,
		IncludeAnswer:  true,
		IncludeRawContent: false,
	}

	if logger != nil {
		logger.Printf("构建搜索请求 - MaxResults: %d", request.MaxResults)
		if len(allowed_domains) > 0 {
			logger.Printf("限制搜索域名: %v", allowed_domains)
		}
		if len(blocked_domains) > 0 {
			logger.Printf("排除搜索域名: %v", blocked_domains)
		}
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		if logger != nil {
			logger.Printf("请求序列化失败: %v", err)
		}
		return fmt.Sprintf("Error marshaling request: %v", err)
	}

	// 发送HTTP请求
	if logger != nil {
		logger.Printf("开始HTTP请求 - URL: %s", tavilyConfig.BaseUrl)
	}
	
	httpStart := time.Now()
	resp, err := http.Post(tavilyConfig.BaseUrl, "application/json", bytes.NewBuffer(requestBody))
	httpDuration := time.Since(httpStart)
	
	if logger != nil {
		logger.Printf("HTTP请求完成 - 耗时: %v", httpDuration)
	}

	if err != nil {
		if logger != nil {
			logger.Printf("HTTP请求失败: %v", err)
		}
		return fmt.Sprintf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if logger != nil {
		logger.Printf("HTTP响应状态: %d %s", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		if logger != nil {
			logger.Printf("HTTP状态码错误: %d %s", resp.StatusCode, resp.Status)
		}
		return fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.Printf("读取响应失败: %v", err)
		}
		return fmt.Sprintf("Error reading response: %v", err)
	}

	if logger != nil {
		logger.Printf("获取响应内容 - 长度: %d bytes", len(responseBody))
	}

	// 解析响应
	var searchResponse TavilySearchResponse
	err = json.Unmarshal(responseBody, &searchResponse)
	if err != nil {
		if logger != nil {
			logger.Printf("响应解析失败: %v", err)
		}
		return fmt.Sprintf("Error parsing response: %v", err)
	}

	if logger != nil {
		logger.Printf("解析搜索结果 - 结果数量: %d", len(searchResponse.Results))
	}

	// 格式化结果
	var result strings.Builder
	
	// 添加AI摘要（如果有）
	if searchResponse.Answer != "" {
		result.WriteString("## 搜索摘要\n")
		result.WriteString(searchResponse.Answer)
		result.WriteString("\n\n")
	}

	// 添加搜索结果
	result.WriteString("## 搜索结果\n\n")
	for i, item := range searchResponse.Results {
		result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, item.Title))
		result.WriteString(fmt.Sprintf("**URL:** %s\n", item.URL))
		result.WriteString(fmt.Sprintf("**评分:** %.2f\n", item.Score))
		result.WriteString(fmt.Sprintf("**内容:** %s\n\n", item.Content))
	}

	finalResult := result.String()
	if logger != nil {
		logger.Printf("TavilySearch函数返回 - 结果长度: %d字符", len(finalResult))
		logger.Printf("性能统计 - HTTP请求: %v, 总耗时: %v", httpDuration, time.Since(start))
	}

	return finalResult
}