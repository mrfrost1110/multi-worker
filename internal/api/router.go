package api

import (
	"net/http"

	"github.com/multi-worker/internal/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter creates a new HTTP router with all routes
func NewRouter(h *Handler, dh *DiscordHandler, auth *middleware.AuthMiddleware) http.Handler {
	mux := http.NewServeMux()

	// Swagger documentation
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Public routes
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("GET /api/v1/health", h.Health)

	// Mount protected routes with authentication

	// User routes
	mux.Handle("/api/v1/auth/profile", auth.Authenticate(http.HandlerFunc(h.GetProfile)))
	mux.Handle("/api/v1/auth/api-key/regenerate", auth.Authenticate(http.HandlerFunc(h.RegenerateAPIKey)))

	// Task routes
	mux.Handle("/api/v1/tasks", auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateTask(w, r)
		case http.MethodGet:
			h.GetTasks(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/tasks/{id}", auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetTask(w, r)
		case http.MethodPut:
			h.UpdateTask(w, r)
		case http.MethodDelete:
			h.DeleteTask(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/tasks/{id}/run", auth.Authenticate(http.HandlerFunc(h.TriggerTask)))
	mux.Handle("/api/v1/tasks/{id}/executions", auth.Authenticate(http.HandlerFunc(h.GetTaskExecutions)))
	mux.Handle("/api/v1/tasks/{id}/executions/{execId}", auth.Authenticate(http.HandlerFunc(h.GetExecution)))

	// Task Discord config routes
	mux.Handle("/api/v1/tasks/{taskId}/discord", auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dh.GetTaskDiscordConfig(w, r)
		case http.MethodPut, http.MethodPost:
			dh.SetTaskDiscordConfig(w, r)
		case http.MethodDelete:
			dh.DeleteTaskDiscordConfig(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Execution routes
	mux.Handle("/api/v1/executions/recent", auth.Authenticate(http.HandlerFunc(h.GetRecentExecutions)))

	// Status routes
	mux.Handle("/api/v1/status", auth.Authenticate(http.HandlerFunc(h.Status)))

	// Discord Bot routes
	mux.Handle("/api/v1/discord/bots", auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			dh.CreateBot(w, r)
		case http.MethodGet:
			dh.ListBots(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/discord/bots/{botId}", auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dh.GetBot(w, r)
		case http.MethodPut:
			dh.UpdateBot(w, r)
		case http.MethodDelete:
			dh.DeleteBot(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Discord Channel routes
	mux.Handle("/api/v1/discord/channels", auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			dh.CreateChannel(w, r)
		case http.MethodGet:
			dh.ListChannels(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/discord/channels/{channelId}", auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dh.GetChannel(w, r)
		case http.MethodPut:
			dh.UpdateChannel(w, r)
		case http.MethodDelete:
			dh.DeleteChannel(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Discord Test webhook
	mux.Handle("/api/v1/discord/test", auth.Authenticate(http.HandlerFunc(dh.TestWebhook)))

	// Apply global middleware
	handler := middleware.CORS(middleware.JSON(middleware.Logger(mux)))

	return handler
}
