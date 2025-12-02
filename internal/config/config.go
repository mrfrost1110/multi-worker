package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	AI       AIConfig
	Discord  DiscordConfig
	Scraper  ScraperConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

type AIConfig struct {
	DefaultProvider string
	OpenAI          OpenAIConfig
	Anthropic       AnthropicConfig
	Google          GoogleConfig
	OpenRouter      OpenRouterConfig
	DeepSeek        DeepSeekConfig
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type AnthropicConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type GoogleConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type OpenRouterConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type DeepSeekConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type DiscordConfig struct {
	DefaultWebhook string
	RateLimitMs    int
}

type ScraperConfig struct {
	UserAgent       string
	RequestTimeout  time.Duration
	RateLimitMs     int
	MaxRetries      int
	ProxyURL        string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 30)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvAsInt("DB_PORT", 5432),
			User:         getEnv("DB_USER", "postgres"),
			Password:     getEnv("DB_PASSWORD", "postgres"),
			Database:     getEnv("DB_NAME", "multiworker"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production-please"),
			ExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 72),
		},
		AI: AIConfig{
			DefaultProvider: getEnv("AI_DEFAULT_PROVIDER", "openai"),
			OpenAI: OpenAIConfig{
				APIKey:  getEnv("OPENAI_API_KEY", ""),
				Model:   getEnv("OPENAI_MODEL", "gpt-4o-mini"),
				BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			},
			Anthropic: AnthropicConfig{
				APIKey:  getEnv("ANTHROPIC_API_KEY", ""),
				Model:   getEnv("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),
				BaseURL: getEnv("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
			},
			Google: GoogleConfig{
				APIKey:  getEnv("GOOGLE_API_KEY", ""),
				Model:   getEnv("GOOGLE_MODEL", "gemini-1.5-flash"),
				BaseURL: getEnv("GOOGLE_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
			},
			OpenRouter: OpenRouterConfig{
				APIKey:  getEnv("OPENROUTER_API_KEY", ""),
				Model:   getEnv("OPENROUTER_MODEL", "openai/gpt-4o-mini"),
				BaseURL: getEnv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
			},
			DeepSeek: DeepSeekConfig{
				APIKey:  getEnv("DEEPSEEK_API_KEY", ""),
				Model:   getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
				BaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1"),
			},
		},
		Discord: DiscordConfig{
			DefaultWebhook: getEnv("DISCORD_DEFAULT_WEBHOOK", ""),
			RateLimitMs:    getEnvAsInt("DISCORD_RATE_LIMIT_MS", 1000),
		},
		Scraper: ScraperConfig{
			UserAgent:      getEnv("SCRAPER_USER_AGENT", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
			RequestTimeout: time.Duration(getEnvAsInt("SCRAPER_REQUEST_TIMEOUT", 30)) * time.Second,
			RateLimitMs:    getEnvAsInt("SCRAPER_RATE_LIMIT_MS", 2000),
			MaxRetries:     getEnvAsInt("SCRAPER_MAX_RETRIES", 3),
			ProxyURL:       getEnv("SCRAPER_PROXY_URL", ""),
		},
	}
}

func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + strconv.Itoa(c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Database +
		" sslmode=" + c.SSLMode
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
