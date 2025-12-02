package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// DiscordBot represents a Discord bot credential
type DiscordBot struct {
	ID            string     `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	ApplicationID string     `json:"application_id" db:"application_id"`
	PublicKey     string     `json:"public_key,omitempty" db:"public_key"`
	Token         string     `json:"-" db:"token"` // Never expose in JSON
	ClientID      string     `json:"client_id" db:"client_id"`
	ClientSecret  string     `json:"-" db:"client_secret"` // Never expose in JSON
	IsActive      bool       `json:"is_active" db:"is_active"`
	IsDefault     bool       `json:"is_default" db:"is_default"`
	CreatedBy     string     `json:"created_by" db:"created_by"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// DiscordChannel represents a Discord channel configuration
type DiscordChannel struct {
	ID          string    `json:"id" db:"id"`
	BotID       string    `json:"bot_id" db:"bot_id"`
	ChannelID   string    `json:"channel_id" db:"channel_id"`
	GuildID     string    `json:"guild_id,omitempty" db:"guild_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	WebhookURL  string    `json:"-" db:"webhook_url"` // Store webhook if using webhook mode
	WebhookID   string    `json:"webhook_id,omitempty" db:"webhook_id"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// TaskDiscordConfig links tasks to specific bot/channel combinations
type TaskDiscordConfig struct {
	ID              string         `json:"id" db:"id"`
	TaskID          string         `json:"task_id" db:"task_id"`
	BotID           *string        `json:"bot_id,omitempty" db:"bot_id"`
	ChannelID       *string        `json:"channel_id,omitempty" db:"channel_id"`
	WebhookURL      string         `json:"-" db:"webhook_url"` // Direct webhook override
	MessageTemplate string         `json:"message_template,omitempty" db:"message_template"`
	EmbedConfig     EmbedConfig    `json:"embed_config,omitempty" db:"embed_config"`
	Username        string         `json:"username,omitempty" db:"username"`
	AvatarURL       string         `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}

// EmbedConfig for Discord embeds
type EmbedConfig struct {
	Color       int    `json:"color,omitempty"`
	Title       string `json:"title,omitempty"`
	FooterText  string `json:"footer_text,omitempty"`
	FooterIcon  string `json:"footer_icon,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

func (e EmbedConfig) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func (e *EmbedConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, e)
}

// Request/Response types

type CreateDiscordBotRequest struct {
	Name          string `json:"name" validate:"required,min=2,max=100"`
	ApplicationID string `json:"application_id" validate:"required"`
	PublicKey     string `json:"public_key"`
	Token         string `json:"token" validate:"required"`
	ClientID      string `json:"client_id" validate:"required"`
	ClientSecret  string `json:"client_secret"`
	IsDefault     bool   `json:"is_default"`
}

type UpdateDiscordBotRequest struct {
	Name          *string `json:"name,omitempty"`
	Token         *string `json:"token,omitempty"`
	ClientSecret  *string `json:"client_secret,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
	IsDefault     *bool   `json:"is_default,omitempty"`
}

type CreateDiscordChannelRequest struct {
	BotID       string `json:"bot_id" validate:"required"`
	ChannelID   string `json:"channel_id" validate:"required"`
	GuildID     string `json:"guild_id"`
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description"`
	WebhookURL  string `json:"webhook_url"`
}

type UpdateDiscordChannelRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	WebhookURL  *string `json:"webhook_url,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type SetTaskDiscordConfigRequest struct {
	BotID           *string     `json:"bot_id,omitempty"`
	ChannelID       *string     `json:"channel_id,omitempty"`
	WebhookURL      string      `json:"webhook_url,omitempty"`
	MessageTemplate string      `json:"message_template,omitempty"`
	EmbedConfig     EmbedConfig `json:"embed_config,omitempty"`
	Username        string      `json:"username,omitempty"`
	AvatarURL       string      `json:"avatar_url,omitempty"`
}

// DiscordBotWithChannels includes bot with its configured channels
type DiscordBotWithChannels struct {
	DiscordBot
	Channels []DiscordChannel `json:"channels,omitempty"`
}

// DiscordConfigSummary provides a summary view for API responses
type DiscordConfigSummary struct {
	BotID       string `json:"bot_id,omitempty"`
	BotName     string `json:"bot_name,omitempty"`
	ChannelID   string `json:"channel_id,omitempty"`
	ChannelName string `json:"channel_name,omitempty"`
	HasWebhook  bool   `json:"has_webhook"`
}
