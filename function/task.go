package function

import (
	"context"
	"fmt"
	"strings"

	"github.com/ccIisIaIcat/GoAgent/agent/ConversationManager"
	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

type TaskRequest struct {
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

func Task(description string, prompt string) string {
	req := TaskRequest{
		Description: description,
		Prompt:      prompt,
	}

	config, err := general.LoadConfig("./LLMConfig.yaml")
	if err != nil {
		return fmt.Sprintf("Failed to load config: %v", err)
	}

	agentManager := general.NewAgentManager()
	agentManager.AddProvider(config.ToProviderConfigs()[0])

	cm := ConversationManager.NewConversationManager(agentManager)
	cm.SetSystemPrompt("You are a helpful AI assistant that can perform various tasks using available tools.")

	info_chan := make(chan general.Message, 10)
	defer close(info_chan)

	ctx := context.Background()
	_, _, err, _ = cm.Chat(ctx, general.ProviderOpenAI, config.AgentAPIKey.OpenAI.Model, req.Prompt, []string{}, info_chan)

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
		return fmt.Sprintf("Error: %v", err)
	}

	if len(response) == 0 {
		return "Task completed successfully"
	}

	result := strings.Join(response, "\n")
	return result
}
