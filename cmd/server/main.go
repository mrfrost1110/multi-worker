package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/multi-worker/internal/api"
	"github.com/multi-worker/internal/config"
	"github.com/multi-worker/internal/executor/ai"
	"github.com/multi-worker/internal/executor/discord"
	"github.com/multi-worker/internal/executor/filter"
	"github.com/multi-worker/internal/executor/rss"
	"github.com/multi-worker/internal/executor/scraper"
	"github.com/multi-worker/internal/middleware"
	"github.com/multi-worker/internal/scheduler"
	"github.com/multi-worker/internal/storage"

	_ "github.com/multi-worker/docs" // swagger docs
)

// @title Multi-Worker API
// @version 1.0
// @description Dynamic AI-Powered Task Scheduler with multi-provider AI integration for scraping jobs, news, and content, then intelligently processing and delivering notifications to Discord.

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your JWT token with the `Bearer ` prefix, e.g. "Bearer eyJhbGci..."

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description Enter your API key

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	log.Println("Connecting to database...")
	db, err := storage.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	log.Println("Running migrations...")
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	userRepo := storage.NewUserRepository(db)
	taskRepo := storage.NewTaskRepository(db)
	execRepo := storage.NewExecutionRepository(db)
	cacheRepo := storage.NewCacheRepository(db)
	discordRepo := storage.NewDiscordRepository(db)

	// Create default admin user if not exists
	ctx := context.Background()
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminEmail != "" && adminPassword != "" {
		admin, err := userRepo.CreateAdmin(ctx, adminEmail, adminPassword, "Admin")
		if err != nil {
			log.Printf("Warning: Failed to create admin user: %v", err)
		} else {
			log.Printf("Admin user ready: %s", admin.Email)
		}
	}

	// Initialize AI providers
	log.Println("Initializing AI providers...")
	aiRegistry := ai.NewProviderRegistry(&cfg.AI)
	log.Printf("Available AI providers: %v", aiRegistry.Available())

	// Initialize executors
	aiExecutor := ai.NewExecutor(aiRegistry)
	scraperRegistry := scraper.NewRegistry(cfg.Scraper)
	scraperExecutor := scraper.NewExecutor(scraperRegistry, cacheRepo)
	rssExecutor := rss.NewExecutor(cacheRepo)
	discordExecutor := discord.NewExecutor(cfg.Discord)
	filterExecutor := filter.NewExecutor(cacheRepo)

	// Initialize pipeline runner
	runner := scheduler.NewPipelineRunner(
		taskRepo,
		execRepo,
		cacheRepo,
		discordRepo,
		aiExecutor,
		scraperExecutor,
		rssExecutor,
		discordExecutor,
		filterExecutor,
	)

	// Initialize scheduler
	sched := scheduler.NewScheduler(taskRepo, execRepo, runner)

	// Start scheduler
	log.Println("Starting scheduler...")
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT, userRepo)

	// Initialize API handlers
	handler := api.NewHandler(userRepo, taskRepo, execRepo, sched, runner, authMiddleware)
	discordHandler := api.NewDiscordHandler(discordRepo)

	// Setup router
	router := api.NewRouter(handler, discordHandler, authMiddleware)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Stop scheduler
	sched.Stop()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
