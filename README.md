# Multi-Worker: Dynamic AI-Powered Task Scheduler

A highly configurable Go-based task scheduler with AI integration for scraping jobs, news, and content, then intelligently processing and delivering notifications to Discord.

## Features

- **Dynamic Task Management**: Create, update, and delete tasks via REST API
- **Pipeline Architecture**: Chain multiple steps (scrape → AI process → notify)
- **Multi-Provider AI**: Support for OpenAI, Anthropic Claude, Google Gemini, OpenRouter, and DeepSeek
- **Multiple Scrapers**: RemoteOK, HackerNews Jobs, WeWorkRemotely, Dev.to, and more
- **RSS Feed Support**: Subscribe to any RSS/Atom feed
- **Discord Notifications**: Rich embeds with customizable templates
- **Deduplication**: Content caching to avoid duplicate notifications
- **JWT + API Key Auth**: Secure API access
- **Docker Ready**: Full Docker Compose setup with PostgreSQL

## Quick Start

### Using Docker (Recommended)

1. Clone and configure:
```bash
cd multi-worker
cp .env.example .env
# Edit .env with your API keys
```

2. Start the services:
```bash
docker compose up -d
```

3. The API is now running at `http://localhost:8080`

### Manual Setup

1. Install Go 1.22+ and PostgreSQL 16+

2. Configure environment:
```bash
cp .env.example .env
# Edit .env with your settings
```

3. Run:
```bash
make deps
make run
```

## API Endpoints

### Authentication

```bash
# Register
POST /api/v1/auth/register
{
  "email": "user@example.com",
  "password": "password123",
  "name": "John Doe"
}

# Login
POST /api/v1/auth/login
{
  "email": "user@example.com",
  "password": "password123"
}
# Returns: { "token": "jwt-token", "expires_at": 1234567890 }
```

### Tasks

```bash
# Create Task
POST /api/v1/tasks
Authorization: Bearer <token>

# List Tasks
GET /api/v1/tasks?limit=50&offset=0&status=enabled

# Get Task
GET /api/v1/tasks/{id}

# Update Task
PUT /api/v1/tasks/{id}

# Delete Task
DELETE /api/v1/tasks/{id}

# Trigger Task Manually
POST /api/v1/tasks/{id}/run

# Get Task Executions
GET /api/v1/tasks/{id}/executions
```

### Health & Status

```bash
GET /api/v1/health
GET /api/v1/status
```

### Discord Bot Management

```bash
# Create a Discord Bot
POST /api/v1/discord/bots
{
  "name": "My Bot",
  "application_id": "123456789",
  "public_key": "abc123...",
  "token": "bot-token-here",
  "client_id": "123456789",
  "client_secret": "secret",
  "is_default": true
}

# List Bots
GET /api/v1/discord/bots

# Get Bot with Channels
GET /api/v1/discord/bots/{botId}

# Update Bot
PUT /api/v1/discord/bots/{botId}

# Delete Bot
DELETE /api/v1/discord/bots/{botId}

# Create Channel Config
POST /api/v1/discord/channels
{
  "bot_id": "uuid",
  "channel_id": "discord-channel-id",
  "guild_id": "discord-guild-id",
  "name": "alerts",
  "webhook_url": "https://discord.com/api/webhooks/..."
}

# List Channels
GET /api/v1/discord/channels?bot_id=uuid

# Set Task Discord Config (link task to bot/channel)
PUT /api/v1/tasks/{taskId}/discord
{
  "bot_id": "uuid",
  "channel_id": "uuid",
  "webhook_url": "optional-direct-webhook",
  "username": "Custom Bot Name",
  "embed_config": {
    "color": 5814783
  }
}

# Get Task Discord Config
GET /api/v1/tasks/{taskId}/discord
```

## Task Pipeline Configuration

### Example: Job Scraper → AI Summary → Discord

```json
{
  "name": "Golang Remote Jobs",
  "description": "Scrape and notify golang remote jobs every 2 hours",
  "schedule": "0 */2 * * *",
  "pipeline": [
    {
      "type": "scraper",
      "name": "Scrape RemoteOK",
      "config": {
        "source": "remoteok",
        "query": "golang",
        "limit": 10
      }
    },
    {
      "type": "filter",
      "name": "Deduplicate",
      "config": {
        "deduplicate": true,
        "include_keywords": ["remote", "golang", "go"],
        "exclude_keywords": ["senior only"]
      }
    },
    {
      "type": "ai_processor",
      "name": "Summarize with AI",
      "config": {
        "provider": "openai",
        "prompt": "Summarize these job listings. Highlight salary, requirements, and company info. Format for Discord.",
        "system_prompt": "You are a helpful assistant that summarizes job listings concisely."
      }
    },
    {
      "type": "discord",
      "name": "Notify Discord",
      "config": {
        "webhook_url": "https://discord.com/api/webhooks/...",
        "username": "Job Bot",
        "color": 5814783
      }
    }
  ]
}
```

### Example: Tech News RSS → Discord

```json
{
  "name": "Tech News Digest",
  "description": "Daily tech news from multiple sources",
  "schedule": "0 9 * * *",
  "pipeline": [
    {
      "type": "rss",
      "config": {
        "urls": [
          "https://hnrss.org/frontpage",
          "https://techcrunch.com/feed/",
          "https://dev.to/feed"
        ],
        "limit": 20,
        "keywords": ["ai", "golang", "rust", "startup"]
      }
    },
    {
      "type": "ai_processor",
      "config": {
        "provider": "anthropic",
        "prompt": "Create a brief morning tech digest from these articles. Group by topic. Keep it under 1500 characters."
      }
    },
    {
      "type": "discord",
      "config": {
        "template": "🌅 **Morning Tech Digest**\n\n{{.}}"
      }
    }
  ]
}
```

### Example: Freelance Jobs → Discord

```json
{
  "name": "Freelance Opportunities",
  "description": "Monitor freelance platforms for relevant projects",
  "schedule": "0 */4 * * *",
  "pipeline": [
    {
      "type": "scraper",
      "config": {
        "sources": ["upwork", "freelancer"],
        "query": "python backend api",
        "limit": 15
      }
    },
    {
      "type": "filter",
      "config": {
        "deduplicate": true,
        "exclude_keywords": ["$5", "$10", "cheap"]
      }
    },
    {
      "type": "discord",
      "config": {
        "color": 16776960
      }
    }
  ]
}
```

## Pipeline Step Types

### `scraper`
Web scraping from job boards and content sites.

| Config | Type | Description |
|--------|------|-------------|
| `source` | string | Single source name |
| `sources` | []string | Multiple sources |
| `category` | string | Scrape all in category ("jobs", "freelance", "news") |
| `query` | string | Search query |
| `keywords` | []string | Search keywords |
| `limit` | int | Max items to fetch |

**Available Sources:**
- Jobs: `remoteok`, `hackernews_jobs`, `weworkremotely`
- Freelance: `upwork`, `freelancer`
- News: `hackernews`, `devto`, `producthunt`

### `rss`
RSS/Atom feed reader.

| Config | Type | Description |
|--------|------|-------------|
| `url` | string | Single feed URL |
| `urls` | []string | Multiple feed URLs |
| `limit` | int | Max items per feed |
| `keywords` | []string | Filter by keywords |

### `ai_processor`
AI-powered content processing.

| Config | Type | Description |
|--------|------|-------------|
| `provider` | string | AI provider (openai, anthropic, google, openrouter, deepseek) |
| `prompt` | string | User prompt |
| `system_prompt` | string | System prompt |

### `filter`
Content filtering and deduplication.

| Config | Type | Description |
|--------|------|-------------|
| `include_keywords` | []string | Must contain one of these |
| `exclude_keywords` | []string | Must not contain any of these |
| `deduplicate` | bool | Skip already-seen content |
| `limit` | int | Max items to pass through |

### `discord`
Discord webhook notifications.

| Config | Type | Description |
|--------|------|-------------|
| `webhook_url` | string | Discord webhook URL |
| `template` | string | Go template for message |
| `username` | string | Bot username |
| `avatar_url` | string | Bot avatar URL |
| `color` | int | Embed color (decimal) |

## Cron Schedule Format

Standard cron format with optional seconds:

```
┌───────────── second (0 - 59) [optional]
│ ┌───────────── minute (0 - 59)
│ │ ┌───────────── hour (0 - 23)
│ │ │ ┌───────────── day of month (1 - 31)
│ │ │ │ ┌───────────── month (1 - 12)
│ │ │ │ │ ┌───────────── day of week (0 - 6)
│ │ │ │ │ │
* * * * * *
```

**Shortcuts:**
- `@hourly` - Every hour
- `@daily` - Every day at midnight
- `@weekly` - Every Sunday at midnight
- `@monthly` - First day of month at midnight

**Examples:**
- `*/30 * * * *` - Every 30 minutes
- `0 */2 * * *` - Every 2 hours
- `0 9 * * 1-5` - 9 AM on weekdays
- `0 0 * * 0` - Midnight on Sundays

## Environment Variables

See `.env.example` for all available configuration options.

### Required for Basic Operation
- `DB_*` - PostgreSQL connection
- `JWT_SECRET` - JWT signing key
- `ADMIN_EMAIL/PASSWORD` - Initial admin credentials

### For AI Processing
At least one AI provider API key:
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GOOGLE_API_KEY`
- `OPENROUTER_API_KEY`
- `DEEPSEEK_API_KEY`

### For Notifications
- `DISCORD_DEFAULT_WEBHOOK`

## Development

```bash
# Run with hot reload
make dev

# Run tests
make test

# Lint code
make lint

# Build binary
make build

# Docker commands
make docker-up      # Start services
make docker-logs    # View logs
make docker-down    # Stop services
make docker-rebuild # Rebuild and restart
```

## Architecture

```
multi-worker/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/             # REST API handlers and router
│   ├── config/          # Configuration management
│   ├── executor/        # Pipeline executors
│   │   ├── ai/          # AI provider integrations
│   │   ├── scraper/     # Web scrapers
│   │   ├── rss/         # RSS feed reader
│   │   ├── discord/     # Discord notifier
│   │   └── filter/      # Content filtering
│   ├── middleware/      # HTTP middleware (auth, CORS)
│   ├── model/           # Data models
│   ├── scheduler/       # Cron scheduler and runner
│   └── storage/         # PostgreSQL repositories
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## License

MIT
