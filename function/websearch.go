package function

import (
	"context"
	"fmt"
	"strings"

	"github.com/ccIisIaIcat/GoAgent/agent/ConversationManager"
	"github.com/ccIisIaIcat/GoAgent/agent/general"
)

func WebSearch(query string, allowed_domains []string, blocked_domains []string) string {
	if strings.TrimSpace(query) == "" {
		return "Error: query is required"
	}

	if len(query) < 2 {
		return "Error: query must be at least 2 characters long"
	}

	searchPrompt := fmt.Sprintf("Search for: %s", query)

	if len(allowed_domains) > 0 {
		searchPrompt += fmt.Sprintf("\nOnly search in domains: %s", strings.Join(allowed_domains, ", "))
	}

	if len(blocked_domains) > 0 {
		searchPrompt += fmt.Sprintf("\nExclude domains: %s", strings.Join(blocked_domains, ", "))
	}

	config, err := general.LoadConfig("./LLMConfig.yaml")
	if err != nil {
		return fmt.Sprintf("Failed to load config: %v", err)
	}

	agentManager := general.NewAgentManager()
	agentManager.AddProvider(config.ToProviderConfigs()[0])

	cm := ConversationManager.NewConversationManager(agentManager)
	cm.SetSystemPrompt("You are a web search assistant. Provide search results based on the query.")

	info_chan := make(chan general.Message, 10)
	defer close(info_chan)

	ctx := context.Background()
	_, _, err, _ = cm.Chat(ctx, general.ProviderOpenAI, config.AgentAPIKey.OpenAI.Model, searchPrompt, []string{}, info_chan)

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
		return fmt.Sprintf("Error performing search: %v", err)
	}

	if len(response) == 0 {
		return "No search results found"
	}

	return strings.Join(response, "\n")
}
