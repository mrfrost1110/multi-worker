package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/multi-worker/internal/config"
	"github.com/multi-worker/internal/model"
)

// Executor handles Discord notifications in pipelines
type Executor struct {
	defaultWebhook string
	rateLimit      time.Duration
	lastSend       time.Time
	mu             sync.Mutex
	client         *http.Client
}

// NewExecutor creates a new Discord executor
func NewExecutor(cfg config.DiscordConfig) *Executor {
	return &Executor{
		defaultWebhook: cfg.DefaultWebhook,
		rateLimit:      time.Duration(cfg.RateLimitMs) * time.Millisecond,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (e *Executor) Type() string {
	return "discord"
}

func (e *Executor) Validate(config map[string]interface{}) error {
	// Webhook can come from:
	// 1. Direct webhook_url in config
	// 2. Default webhook from environment
	// 3. Bot/channel configuration in database (resolved at runtime)
	// 4. Task-specific discord config (resolved at runtime)
	// So we don't strictly validate here - runtime will resolve
	return nil
}

// SetWebhook allows runtime injection of webhook URL (from database config)
func (e *Executor) SetWebhook(url string) {
	if url != "" {
		e.defaultWebhook = url
	}
}

func (e *Executor) Execute(ctx context.Context, input *model.ExecutorResult, config map[string]interface{}) (*model.ExecutorResult, error) {
	// Validate input
	if input == nil {
		return nil, fmt.Errorf("discord executor requires input data")
	}

	// Get webhook URL
	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		webhookURL = e.defaultWebhook
	}

	// Validate webhook URL is present
	if webhookURL == "" {
		return nil, fmt.Errorf("no Discord webhook URL configured: set webhook_url in pipeline config, task discord config, or DISCORD_DEFAULT_WEBHOOK environment variable")
	}

	// Get template
	tmplStr, _ := config["template"].(string)

	// Get username and avatar
	username, _ := config["username"].(string)
	avatarURL, _ := config["avatar_url"].(string)

	// Get embed color
	color := 0x5865F2 // Discord blurple
	if c, ok := config["color"].(float64); ok {
		color = int(c)
	}

	// Format the message
	message, err := e.formatMessage(input, tmplStr, username, avatarURL, color)
	if err != nil {
		return nil, fmt.Errorf("failed to format message: %w", err)
	}

	// Apply rate limiting with mutex protection
	e.mu.Lock()
	if elapsed := time.Since(e.lastSend); elapsed < e.rateLimit {
		e.mu.Unlock()
		time.Sleep(e.rateLimit - elapsed)
		e.mu.Lock()
	}
	e.mu.Unlock()

	// Send to Discord
	if err := e.send(ctx, webhookURL, message); err != nil {
		return nil, fmt.Errorf("failed to send Discord message: %w", err)
	}

	e.mu.Lock()
	e.lastSend = time.Now()
	e.mu.Unlock()

	return &model.ExecutorResult{
		Data: map[string]interface{}{
			"status":  "sent",
			"webhook": maskWebhook(webhookURL),
		},
		ItemCount: input.ItemCount,
		Metadata: map[string]interface{}{
			"items_sent": input.ItemCount,
		},
	}, nil
}

func (e *Executor) formatMessage(input *model.ExecutorResult, tmplStr, username, avatarURL string, color int) (*model.DiscordMessage, error) {
	message := &model.DiscordMessage{
		Username:  username,
		AvatarURL: avatarURL,
	}

	// If input is a string (from AI processor), use it directly
	if str, ok := input.Data.(string); ok {
		// Split long messages
		if len(str) > 2000 {
			str = str[:1997] + "..."
		}
		message.Content = str
		return message, nil
	}

	// If template provided, use it
	if tmplStr != "" {
		content, err := executeTemplate(tmplStr, input.Data)
		if err != nil {
			return nil, err
		}
		if len(content) > 2000 {
			content = content[:1997] + "..."
		}
		message.Content = content
		return message, nil
	}

	// Create embeds for scraped items
	embeds, err := e.createEmbeds(input.Data, color)
	if err != nil {
		// Fallback to JSON representation
		jsonBytes, _ := json.MarshalIndent(input.Data, "", "  ")
		content := string(jsonBytes)
		if len(content) > 2000 {
			content = content[:1997] + "..."
		}
		message.Content = "```json\n" + content + "\n```"
		return message, nil
	}

	message.Embeds = embeds
	return message, nil
}

func (e *Executor) createEmbeds(data interface{}, color int) ([]model.DiscordEmbed, error) {
	var embeds []model.DiscordEmbed

	switch v := data.(type) {
	case []model.ScrapedItem:
		for i, item := range v {
			if i >= 10 { // Discord limit
				break
			}
			embed := model.DiscordEmbed{
				Title:       truncate(item.Title, 256),
				Description: truncate(item.Description, 4096),
				URL:         item.URL,
				Color:       color,
				Footer: &model.DiscordEmbedFooter{
					Text: item.Source,
				},
			}

			var fields []model.DiscordEmbedField
			if item.Company != "" {
				fields = append(fields, model.DiscordEmbedField{
					Name:   "Company",
					Value:  item.Company,
					Inline: true,
				})
			}
			if item.Salary != "" {
				fields = append(fields, model.DiscordEmbedField{
					Name:   "Salary",
					Value:  item.Salary,
					Inline: true,
				})
			}
			if item.Location != "" {
				fields = append(fields, model.DiscordEmbedField{
					Name:   "Location",
					Value:  item.Location,
					Inline: true,
				})
			}
			if len(item.Tags) > 0 {
				fields = append(fields, model.DiscordEmbedField{
					Name:  "Tags",
					Value: strings.Join(item.Tags, ", "),
				})
			}

			embed.Fields = fields
			embeds = append(embeds, embed)
		}

	case []model.RSSItem:
		for i, item := range v {
			if i >= 10 {
				break
			}
			embed := model.DiscordEmbed{
				Title:       truncate(item.Title, 256),
				Description: truncate(item.Description, 4096),
				URL:         item.Link,
				Color:       color,
				Footer: &model.DiscordEmbedFooter{
					Text: item.Source,
				},
			}

			if item.PubDate != "" {
				embed.Timestamp = parseAndFormatDate(item.PubDate)
			}

			embeds = append(embeds, embed)
		}

	default:
		return nil, fmt.Errorf("unsupported data type for embeds")
	}

	return embeds, nil
}

func (e *Executor) send(ctx context.Context, webhookURL string, message *model.DiscordMessage) error {
	jsonBody, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Discord API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendSimple sends a simple text message
func (e *Executor) SendSimple(ctx context.Context, webhookURL, content string) error {
	message := &model.DiscordMessage{Content: content}
	return e.send(ctx, webhookURL, message)
}

// SendEmbed sends a single embed
func (e *Executor) SendEmbed(ctx context.Context, webhookURL string, embed model.DiscordEmbed) error {
	message := &model.DiscordMessage{Embeds: []model.DiscordEmbed{embed}}
	return e.send(ctx, webhookURL, message)
}

func executeTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("discord").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func maskWebhook(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:30] + "***"
}

func parseAndFormatDate(dateStr string) string {
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format(time.RFC3339)
		}
	}

	return ""
}
