package function

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ccIisIaIcat/GoAgent/agent/ConversationManager"
	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

func WebFetch(url string, prompt string) string {
	if url == "" || prompt == "" {
		return "Error: both url and prompt are required"
	}

	if strings.HasPrefix(url, "http://") {
		url = strings.Replace(url, "http://", "https://", 1)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Sprintf("Error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}

	content := string(body)
	if len(content) > 50000 {
		content = content[:50000] + "... [content truncated]"
	}

	config, err := general.LoadConfig("./LLMConfig.yaml")
	if err != nil {
		return fmt.Sprintf("Failed to load config: %v", err)
	}

	agentManager := general.NewAgentManager()
	agentManager.AddProvider(config.ToProviderConfigs()[0])

	cm := ConversationManager.NewConversationManager(agentManager)
	cm.SetSystemPrompt("You are a helpful AI assistant that processes web content.")

	fullPrompt := fmt.Sprintf("Content from %s:\n\n%s\n\nUser request: %s", url, content, prompt)

	info_chan := make(chan general.Message, 10)
	defer close(info_chan)

	ctx := context.Background()
	_, _, err, _ = cm.Chat(ctx, general.ProviderOpenAI, config.AgentAPIKey.OpenAI.Model, fullPrompt, []string{}, info_chan)

	var response []string
	for msg := range info_chan {
		if msg.Role == general.RoleAssistant {
			for _, content := range msg.Content {
				if content.Type == general.ContentTypeText && content.Text != "" {
					response = append(response, content.Text)
				}
			}
		}
	}

	if err != nil {
		return fmt.Sprintf("Error processing content: %v", err)
	}

	if len(response) == 0 {
		return "No response from AI model"
	}

	return strings.Join(response, "\n")
}
