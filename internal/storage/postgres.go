package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/multi-worker/internal/config"
)

type Database struct {
	*sqlx.DB
}

func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	return &Database{DB: db}, nil
}

func (d *Database) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) RunMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(100) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'user',
			api_key VARCHAR(64) UNIQUE,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			description TEXT,
			schedule VARCHAR(100) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'enabled',
			pipeline JSONB NOT NULL,
			last_run_at TIMESTAMP WITH TIME ZONE,
			next_run_at TIMESTAMP WITH TIME ZONE,
			created_by UUID REFERENCES users(id),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
			task_name VARCHAR(100) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			finished_at TIMESTAMP WITH TIME ZONE,
			duration_ms BIGINT,
			step_results JSONB,
			error TEXT,
			triggered_by VARCHAR(100) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS content_cache (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			content_hash VARCHAR(64) NOT NULL,
			source VARCHAR(100) NOT NULL,
			task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(content_hash, task_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_next_run ON tasks(next_run_at)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_task_id ON executions(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_started_at ON executions(started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_content_cache_hash ON content_cache(content_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_content_cache_task ON content_cache(task_id)`,

		// Discord bot credentials table
		`CREATE TABLE IF NOT EXISTS discord_bots (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			application_id VARCHAR(100) NOT NULL UNIQUE,
			public_key VARCHAR(255),
			token TEXT NOT NULL,
			client_id VARCHAR(100) NOT NULL,
			client_secret TEXT,
			is_active BOOLEAN DEFAULT true,
			is_default BOOLEAN DEFAULT false,
			created_by UUID REFERENCES users(id),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Discord channels table
		`CREATE TABLE IF NOT EXISTS discord_channels (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			bot_id UUID REFERENCES discord_bots(id) ON DELETE CASCADE,
			channel_id VARCHAR(100) NOT NULL,
			guild_id VARCHAR(100),
			name VARCHAR(100) NOT NULL,
			description TEXT,
			webhook_url TEXT,
			webhook_id VARCHAR(100),
			is_active BOOLEAN DEFAULT true,
			created_by UUID REFERENCES users(id),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(bot_id, channel_id)
		)`,

		// Task Discord configuration table
		`CREATE TABLE IF NOT EXISTS task_discord_configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			task_id UUID REFERENCES tasks(id) ON DELETE CASCADE UNIQUE,
			bot_id UUID REFERENCES discord_bots(id) ON DELETE SET NULL,
			channel_id UUID REFERENCES discord_channels(id) ON DELETE SET NULL,
			webhook_url TEXT,
			message_template TEXT,
			embed_config JSONB DEFAULT '{}',
			username VARCHAR(100),
			avatar_url TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for Discord tables
		`CREATE INDEX IF NOT EXISTS idx_discord_bots_active ON discord_bots(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_discord_bots_default ON discord_bots(is_default) WHERE is_default = true`,
		`CREATE INDEX IF NOT EXISTS idx_discord_channels_bot ON discord_channels(bot_id)`,
		`CREATE INDEX IF NOT EXISTS idx_discord_channels_active ON discord_channels(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_task_discord_configs_task ON task_discord_configs(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_task_discord_configs_bot ON task_discord_configs(bot_id)`,

		// Ensure only one default bot
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_discord_bots_single_default
		 ON discord_bots(is_default) WHERE is_default = true`,
	}

	for _, migration := range migrations {
		if _, err := d.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
