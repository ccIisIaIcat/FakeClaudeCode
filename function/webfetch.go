package function

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ccIisIaIcat/GoAgent/agent/ConversationManager"
	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

func WebFetch(url string, prompt string) string {
	start := time.Now()
	
	// 记录日志
	logFile, err := os.OpenFile("./log/webfetch.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var logger *log.Logger
	if err == nil {
		defer logFile.Close()
		logger = log.New(logFile, "", log.LstdFlags)
		logger.Printf("WebFetch函数调用 - url: %s, prompt_length: %d", url, len(prompt))
		defer func() {
			duration := time.Since(start)
			logger.Printf("WebFetch函数执行完成 - 耗时: %v", duration)
		}()
	}

	if url == "" || prompt == "" {
		if logger != nil {
			logger.Printf("WebFetch函数返回 - 错误: both url and prompt are required")
		}
		return "Error: both url and prompt are required"
	}

	originalUrl := url
	if strings.HasPrefix(url, "http://") {
		url = strings.Replace(url, "http://", "https://", 1)
		if logger != nil {
			logger.Printf("URL协议升级: %s -> %s", originalUrl, url)
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if logger != nil {
		logger.Printf("开始HTTP请求 - URL: %s", url)
	}
	
	// 记录HTTP请求开始时间
	httpStart := time.Now()
	resp, err := client.Get(url)
	httpDuration := time.Since(httpStart)
	
	if logger != nil {
		logger.Printf("HTTP请求完成 - 耗时: %v", httpDuration)
	}

	if err != nil {
		if logger != nil {
			logger.Printf("HTTP请求失败: %v", err)
		}
		return fmt.Sprintf("Error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	if logger != nil {
		logger.Printf("HTTP响应状态: %d %s", resp.StatusCode, resp.Status)
		logger.Printf("Content-Type: %s", resp.Header.Get("Content-Type"))
		logger.Printf("Content-Length: %s", resp.Header.Get("Content-Length"))
	}

	if resp.StatusCode != http.StatusOK {
		if logger != nil {
			logger.Printf("HTTP状态码错误: %d %s", resp.StatusCode, resp.Status)
		}
		return fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.Printf("读取响应内容失败: %v", err)
		}
		return fmt.Sprintf("Error reading response: %v", err)
	}

	originalContentLength := len(body)
	content := string(body)
	
	if len(content) > 50000 {
		content = content[:50000] + "... [content truncated]"
		if logger != nil {
			logger.Printf("内容截断 - 原始长度: %d bytes, 截断后: %d bytes", originalContentLength, len(content))
		}
	} else {
		if logger != nil {
			logger.Printf("获取网页内容 - 长度: %d bytes", originalContentLength)
		}
	}

	config, err := general.LoadConfig("./LLMConfig.yaml")
	if err != nil {
		if logger != nil {
			logger.Printf("配置加载失败: %v", err)
		}
		return fmt.Sprintf("Failed to load config: %v", err)
	}

	agentManager := general.NewAgentManager()
	agentManager.AddProvider(config.ToProviderConfigs()[0])

	cm := ConversationManager.NewConversationManager(agentManager)
	cm.SetSystemPrompt("You are a helpful AI assistant that processes web content.")

	fullPrompt := fmt.Sprintf("Content from %s:\n\n%s\n\nUser request: %s", url, content, prompt)
	
	if logger != nil {
		logger.Printf("构建AI处理提示 - 完整提示长度: %d字符", len(fullPrompt))
		logger.Printf("开始AI内容处理 - 模型: %s", config.AgentAPIKey.OpenAI.Model)
	}

	info_chan := make(chan general.Message, 10)
	
	ctx := context.Background()
	
	// 记录AI处理开始时间
	aiStart := time.Now()
	_, _, err, usage := cm.Chat(ctx, general.ProviderOpenAI, config.AgentAPIKey.OpenAI.Model, fullPrompt, []string{}, info_chan)
	aiDuration := time.Since(aiStart)

	// 手动关闭通道，确保for range能够结束
	close(info_chan)

	if logger != nil {
		logger.Printf("AI处理完成 - 耗时: %v", aiDuration)
		logger.Printf("开始处理响应通道...")
		if usage != nil {
			logger.Printf("Token使用情况: Prompt=%d, Completion=%d, Total=%d",
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
		}
	}

	var response []string
	messageCount := 0
	if logger != nil {
		logger.Printf("开始读取info_chan通道消息...")
	}
	for msg := range info_chan {
		messageCount++
		if logger != nil {
			logger.Printf("收到第%d条消息, Role: %s", messageCount, msg.Role)
		}
		if msg.Role == general.RoleAssistant {
			for _, content := range msg.Content {
				if content.Type == general.ContentTypeText && content.Text != "" {
					response = append(response, content.Text)
					if logger != nil {
						logger.Printf("添加响应内容，长度: %d", len(content.Text))
					}
				}
			}
		}
	}
	if logger != nil {
		logger.Printf("info_chan通道读取完成，跳出循环")
	}

	if logger != nil {
		logger.Printf("处理响应消息 - 消息数量: %d, 响应内容数量: %d", messageCount, len(response))
	}

	if err != nil {
		if logger != nil {
			logger.Printf("AI内容处理失败: %v", err)
		}
		return fmt.Sprintf("Error processing content: %v", err)
	}

	if len(response) == 0 {
		if logger != nil {
			logger.Printf("AI未返回任何响应")
		}
		return "No response from AI model"
	}

	result := strings.Join(response, "\n")
	if logger != nil {
		logger.Printf("WebFetch函数返回 - 结果长度: %d字符", len(result))
		logger.Printf("WebFetch返回值内容: %s", result)
		logger.Printf("性能统计 - HTTP请求: %v, AI处理: %v, 总耗时: %v", 
			httpDuration, aiDuration, time.Since(start))
	}
	
	return result
}
