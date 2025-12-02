package model

import "context"

// ExecutorResult represents the output of a pipeline step
type ExecutorResult struct {
	Data     interface{} `json:"data"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	ItemCount int `json:"item_count,omitempty"`
}

// Executor interface that all pipeline executors must implement
type Executor interface {
	// Type returns the executor type identifier
	Type() string

	// Execute runs the executor with given input and config
	Execute(ctx context.Context, input *ExecutorResult, config map[string]interface{}) (*ExecutorResult, error)

	// Validate checks if the config is valid for this executor
	Validate(config map[string]interface{}) error
}

// ScrapedItem represents a scraped job or content item
type ScrapedItem struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	URL         string                 `json:"url"`
	Source      string                 `json:"source"`
	Category    string                 `json:"category,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Salary      string                 `json:"salary,omitempty"`
	Company     string                 `json:"company,omitempty"`
	Location    string                 `json:"location,omitempty"`
	PostedAt    string                 `json:"posted_at,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// RSSItem represents an RSS feed item
type RSSItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Link        string   `json:"link"`
	Source      string   `json:"source"`
	PubDate     string   `json:"pub_date"`
	Categories  []string `json:"categories,omitempty"`
	Author      string   `json:"author,omitempty"`
}

// DiscordMessage represents a message to send to Discord
type DiscordMessage struct {
	Content   string          `json:"content,omitempty"`
	Embeds    []DiscordEmbed  `json:"embeds,omitempty"`
	Username  string          `json:"username,omitempty"`
	AvatarURL string          `json:"avatar_url,omitempty"`
}

type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}
