package storage

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/multi-worker/internal/model"
)

// DiscordRepository handles Discord bot and channel storage
type DiscordRepository struct {
	db            *Database
	encryptionKey []byte
}

// NewDiscordRepository creates a new Discord repository
func NewDiscordRepository(db *Database) *DiscordRepository {
	// Get encryption key from environment or generate a default
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		key = os.Getenv("JWT_SECRET") // Fallback to JWT secret
	}
	if len(key) < 32 {
		key = key + "00000000000000000000000000000000" // Pad to 32 bytes
	}

	return &DiscordRepository{
		db:            db,
		encryptionKey: []byte(key[:32]),
	}
}

// Encryption helpers

func (r *DiscordRepository) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (r *DiscordRepository) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Bot CRUD operations

func (r *DiscordRepository) CreateBot(ctx context.Context, req *model.CreateDiscordBotRequest, userID string) (*model.DiscordBot, error) {
	// Encrypt sensitive fields
	encryptedToken, err := r.encrypt(req.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	encryptedSecret, err := r.encrypt(req.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
	}

	// If this is set as default, unset other defaults first
	if req.IsDefault {
		_, err := r.db.ExecContext(ctx, `UPDATE discord_bots SET is_default = false WHERE is_default = true`)
		if err != nil {
			return nil, fmt.Errorf("failed to unset default: %w", err)
		}
	}

	var bot model.DiscordBot
	query := `
		INSERT INTO discord_bots (name, application_id, public_key, token, client_id, client_secret, is_default, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, application_id, public_key, client_id, is_active, is_default, created_by, created_at, updated_at
	`
	err = r.db.QueryRowxContext(ctx, query,
		req.Name, req.ApplicationID, req.PublicKey, encryptedToken,
		req.ClientID, encryptedSecret, req.IsDefault, userID,
	).StructScan(&bot)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return &bot, nil
}

func (r *DiscordRepository) GetBot(ctx context.Context, id string) (*model.DiscordBot, error) {
	var bot model.DiscordBot
	query := `
		SELECT id, name, application_id, public_key, client_id, is_active, is_default, created_by, created_at, updated_at
		FROM discord_bots WHERE id = $1
	`
	err := r.db.GetContext(ctx, &bot, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}
	return &bot, nil
}

func (r *DiscordRepository) GetBotWithCredentials(ctx context.Context, id string) (*model.DiscordBot, error) {
	var bot model.DiscordBot
	var encryptedToken, encryptedSecret string

	query := `
		SELECT id, name, application_id, public_key, token, client_id, client_secret, is_active, is_default, created_by, created_at, updated_at
		FROM discord_bots WHERE id = $1 AND is_active = true
	`
	row := r.db.QueryRowxContext(ctx, query, id)
	err := row.Scan(&bot.ID, &bot.Name, &bot.ApplicationID, &bot.PublicKey,
		&encryptedToken, &bot.ClientID, &encryptedSecret,
		&bot.IsActive, &bot.IsDefault, &bot.CreatedBy, &bot.CreatedAt, &bot.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}

	// Decrypt sensitive fields
	bot.Token, _ = r.decrypt(encryptedToken)
	bot.ClientSecret, _ = r.decrypt(encryptedSecret)

	return &bot, nil
}

func (r *DiscordRepository) GetDefaultBot(ctx context.Context) (*model.DiscordBot, error) {
	var bot model.DiscordBot
	var encryptedToken, encryptedSecret string

	query := `
		SELECT id, name, application_id, public_key, token, client_id, client_secret, is_active, is_default, created_by, created_at, updated_at
		FROM discord_bots WHERE is_default = true AND is_active = true
		LIMIT 1
	`
	row := r.db.QueryRowxContext(ctx, query)
	err := row.Scan(&bot.ID, &bot.Name, &bot.ApplicationID, &bot.PublicKey,
		&encryptedToken, &bot.ClientID, &encryptedSecret,
		&bot.IsActive, &bot.IsDefault, &bot.CreatedBy, &bot.CreatedAt, &bot.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get default bot: %w", err)
	}

	bot.Token, _ = r.decrypt(encryptedToken)
	bot.ClientSecret, _ = r.decrypt(encryptedSecret)

	return &bot, nil
}

func (r *DiscordRepository) ListBots(ctx context.Context, includeInactive bool) ([]model.DiscordBot, error) {
	var bots []model.DiscordBot
	query := `
		SELECT id, name, application_id, public_key, client_id, is_active, is_default, created_by, created_at, updated_at
		FROM discord_bots
	`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY is_default DESC, created_at DESC`

	err := r.db.SelectContext(ctx, &bots, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list bots: %w", err)
	}
	return bots, nil
}

func (r *DiscordRepository) UpdateBot(ctx context.Context, id string, req *model.UpdateDiscordBotRequest) (*model.DiscordBot, error) {
	bot, err := r.GetBot(ctx, id)
	if err != nil || bot == nil {
		return nil, err
	}

	// Handle default flag
	if req.IsDefault != nil && *req.IsDefault {
		_, err := r.db.ExecContext(ctx, `UPDATE discord_bots SET is_default = false WHERE is_default = true AND id != $1`, id)
		if err != nil {
			return nil, fmt.Errorf("failed to unset default: %w", err)
		}
	}

	// Build update query
	updates := "updated_at = $1"
	args := []interface{}{time.Now()}
	argNum := 2

	if req.Name != nil {
		updates += fmt.Sprintf(", name = $%d", argNum)
		args = append(args, *req.Name)
		argNum++
	}
	if req.Token != nil {
		encryptedToken, _ := r.encrypt(*req.Token)
		updates += fmt.Sprintf(", token = $%d", argNum)
		args = append(args, encryptedToken)
		argNum++
	}
	if req.ClientSecret != nil {
		encryptedSecret, _ := r.encrypt(*req.ClientSecret)
		updates += fmt.Sprintf(", client_secret = $%d", argNum)
		args = append(args, encryptedSecret)
		argNum++
	}
	if req.IsActive != nil {
		updates += fmt.Sprintf(", is_active = $%d", argNum)
		args = append(args, *req.IsActive)
		argNum++
	}
	if req.IsDefault != nil {
		updates += fmt.Sprintf(", is_default = $%d", argNum)
		args = append(args, *req.IsDefault)
		argNum++
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE discord_bots SET %s WHERE id = $%d
		RETURNING id, name, application_id, public_key, client_id, is_active, is_default, created_by, created_at, updated_at
	`, updates, argNum)

	var updated model.DiscordBot
	err = r.db.QueryRowxContext(ctx, query, args...).StructScan(&updated)
	if err != nil {
		return nil, fmt.Errorf("failed to update bot: %w", err)
	}

	return &updated, nil
}

func (r *DiscordRepository) DeleteBot(ctx context.Context, id string) error {
	query := `DELETE FROM discord_bots WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bot: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("bot not found")
	}
	return nil
}

// Channel CRUD operations

func (r *DiscordRepository) CreateChannel(ctx context.Context, req *model.CreateDiscordChannelRequest, userID string) (*model.DiscordChannel, error) {
	encryptedWebhook, err := r.encrypt(req.WebhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt webhook: %w", err)
	}

	var channel model.DiscordChannel
	query := `
		INSERT INTO discord_channels (bot_id, channel_id, guild_id, name, description, webhook_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, bot_id, channel_id, guild_id, name, description, webhook_id, is_active, created_by, created_at, updated_at
	`
	err = r.db.QueryRowxContext(ctx, query,
		req.BotID, req.ChannelID, req.GuildID, req.Name, req.Description, encryptedWebhook, userID,
	).StructScan(&channel)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	return &channel, nil
}

func (r *DiscordRepository) GetChannel(ctx context.Context, id string) (*model.DiscordChannel, error) {
	var channel model.DiscordChannel
	query := `
		SELECT id, bot_id, channel_id, guild_id, name, description, webhook_id, is_active, created_by, created_at, updated_at
		FROM discord_channels WHERE id = $1
	`
	err := r.db.GetContext(ctx, &channel, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}
	return &channel, nil
}

func (r *DiscordRepository) GetChannelWithWebhook(ctx context.Context, id string) (*model.DiscordChannel, error) {
	var channel model.DiscordChannel
	var encryptedWebhook sql.NullString

	query := `
		SELECT id, bot_id, channel_id, guild_id, name, description, webhook_url, webhook_id, is_active, created_by, created_at, updated_at
		FROM discord_channels WHERE id = $1 AND is_active = true
	`
	row := r.db.QueryRowxContext(ctx, query, id)
	err := row.Scan(&channel.ID, &channel.BotID, &channel.ChannelID, &channel.GuildID,
		&channel.Name, &channel.Description, &encryptedWebhook, &channel.WebhookID,
		&channel.IsActive, &channel.CreatedBy, &channel.CreatedAt, &channel.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	if encryptedWebhook.Valid {
		channel.WebhookURL, _ = r.decrypt(encryptedWebhook.String)
	}

	return &channel, nil
}

func (r *DiscordRepository) ListChannelsByBot(ctx context.Context, botID string) ([]model.DiscordChannel, error) {
	var channels []model.DiscordChannel
	query := `
		SELECT id, bot_id, channel_id, guild_id, name, description, webhook_id, is_active, created_by, created_at, updated_at
		FROM discord_channels WHERE bot_id = $1 AND is_active = true
		ORDER BY name
	`
	err := r.db.SelectContext(ctx, &channels, query, botID)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	return channels, nil
}

func (r *DiscordRepository) ListAllChannels(ctx context.Context) ([]model.DiscordChannel, error) {
	var channels []model.DiscordChannel
	query := `
		SELECT id, bot_id, channel_id, guild_id, name, description, webhook_id, is_active, created_by, created_at, updated_at
		FROM discord_channels WHERE is_active = true
		ORDER BY name
	`
	err := r.db.SelectContext(ctx, &channels, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	return channels, nil
}

func (r *DiscordRepository) UpdateChannel(ctx context.Context, id string, req *model.UpdateDiscordChannelRequest) (*model.DiscordChannel, error) {
	updates := "updated_at = $1"
	args := []interface{}{time.Now()}
	argNum := 2

	if req.Name != nil {
		updates += fmt.Sprintf(", name = $%d", argNum)
		args = append(args, *req.Name)
		argNum++
	}
	if req.Description != nil {
		updates += fmt.Sprintf(", description = $%d", argNum)
		args = append(args, *req.Description)
		argNum++
	}
	if req.WebhookURL != nil {
		encryptedWebhook, _ := r.encrypt(*req.WebhookURL)
		updates += fmt.Sprintf(", webhook_url = $%d", argNum)
		args = append(args, encryptedWebhook)
		argNum++
	}
	if req.IsActive != nil {
		updates += fmt.Sprintf(", is_active = $%d", argNum)
		args = append(args, *req.IsActive)
		argNum++
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE discord_channels SET %s WHERE id = $%d
		RETURNING id, bot_id, channel_id, guild_id, name, description, webhook_id, is_active, created_by, created_at, updated_at
	`, updates, argNum)

	var channel model.DiscordChannel
	err := r.db.QueryRowxContext(ctx, query, args...).StructScan(&channel)
	if err != nil {
		return nil, fmt.Errorf("failed to update channel: %w", err)
	}

	return &channel, nil
}

func (r *DiscordRepository) DeleteChannel(ctx context.Context, id string) error {
	query := `DELETE FROM discord_channels WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("channel not found")
	}
	return nil
}

// Task Discord Config operations

func (r *DiscordRepository) SetTaskConfig(ctx context.Context, taskID string, req *model.SetTaskDiscordConfigRequest) (*model.TaskDiscordConfig, error) {
	encryptedWebhook, _ := r.encrypt(req.WebhookURL)

	var config model.TaskDiscordConfig
	query := `
		INSERT INTO task_discord_configs (task_id, bot_id, channel_id, webhook_url, message_template, embed_config, username, avatar_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (task_id) DO UPDATE SET
			bot_id = EXCLUDED.bot_id,
			channel_id = EXCLUDED.channel_id,
			webhook_url = EXCLUDED.webhook_url,
			message_template = EXCLUDED.message_template,
			embed_config = EXCLUDED.embed_config,
			username = EXCLUDED.username,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, task_id, bot_id, channel_id, message_template, embed_config, username, avatar_url, created_at, updated_at
	`
	err := r.db.QueryRowxContext(ctx, query,
		taskID, req.BotID, req.ChannelID, encryptedWebhook,
		req.MessageTemplate, req.EmbedConfig, req.Username, req.AvatarURL,
	).StructScan(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to set task config: %w", err)
	}

	return &config, nil
}

func (r *DiscordRepository) GetTaskConfig(ctx context.Context, taskID string) (*model.TaskDiscordConfig, error) {
	var config model.TaskDiscordConfig
	var encryptedWebhook sql.NullString

	query := `
		SELECT id, task_id, bot_id, channel_id, webhook_url, message_template, embed_config, username, avatar_url, created_at, updated_at
		FROM task_discord_configs WHERE task_id = $1
	`
	row := r.db.QueryRowxContext(ctx, query, taskID)
	err := row.Scan(&config.ID, &config.TaskID, &config.BotID, &config.ChannelID,
		&encryptedWebhook, &config.MessageTemplate, &config.EmbedConfig,
		&config.Username, &config.AvatarURL, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task config: %w", err)
	}

	if encryptedWebhook.Valid {
		config.WebhookURL, _ = r.decrypt(encryptedWebhook.String)
	}

	return &config, nil
}

func (r *DiscordRepository) DeleteTaskConfig(ctx context.Context, taskID string) error {
	query := `DELETE FROM task_discord_configs WHERE task_id = $1`
	_, err := r.db.ExecContext(ctx, query, taskID)
	return err
}

// GetWebhookForTask resolves the webhook URL for a task, checking config hierarchy
func (r *DiscordRepository) GetWebhookForTask(ctx context.Context, taskID string) (string, error) {
	// 1. Check task-specific config
	config, err := r.GetTaskConfig(ctx, taskID)
	if err != nil {
		return "", err
	}

	if config != nil {
		// Direct webhook override
		if config.WebhookURL != "" {
			return config.WebhookURL, nil
		}

		// Channel webhook
		if config.ChannelID != nil {
			channel, err := r.GetChannelWithWebhook(ctx, *config.ChannelID)
			if err != nil {
				return "", err
			}
			if channel != nil && channel.WebhookURL != "" {
				return channel.WebhookURL, nil
			}
		}
	}

	// 2. Fallback to default bot's first channel
	defaultBot, err := r.GetDefaultBot(ctx)
	if err != nil || defaultBot == nil {
		return "", nil
	}

	channels, err := r.ListChannelsByBot(ctx, defaultBot.ID)
	if err != nil || len(channels) == 0 {
		return "", nil
	}

	channel, err := r.GetChannelWithWebhook(ctx, channels[0].ID)
	if err != nil || channel == nil {
		return "", nil
	}

	return channel.WebhookURL, nil
}
