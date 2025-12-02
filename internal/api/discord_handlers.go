package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/multi-worker/internal/middleware"
	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/storage"
)

// DiscordHandler handles Discord bot and channel API endpoints
type DiscordHandler struct {
	discordRepo *storage.DiscordRepository
}

// NewDiscordHandler creates a new Discord handler
func NewDiscordHandler(discordRepo *storage.DiscordRepository) *DiscordHandler {
	return &DiscordHandler{discordRepo: discordRepo}
}

// Bot handlers

// CreateBot godoc
// @Summary Create a Discord bot
// @Description Register a new Discord bot with credentials
// @Tags Discord Bots
// @Accept json
// @Produce json
// @Param request body model.CreateDiscordBotRequest true "Bot credentials"
// @Success 201 {object} model.DiscordBot
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/bots [post]
func (h *DiscordHandler) CreateBot(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req model.CreateDiscordBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.ApplicationID == "" || req.Token == "" || req.ClientID == "" {
		respondError(w, http.StatusBadRequest, "name, application_id, token, and client_id are required")
		return
	}

	bot, err := h.discordRepo.CreateBot(r.Context(), &req, claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, bot)
}

// ListBots godoc
// @Summary List Discord bots
// @Description Get all registered Discord bots
// @Tags Discord Bots
// @Produce json
// @Param include_inactive query bool false "Include inactive bots"
// @Success 200 {object} map[string]interface{} "List of bots"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/bots [get]
func (h *DiscordHandler) ListBots(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	bots, err := h.discordRepo.ListBots(r.Context(), includeInactive)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"bots":  bots,
		"total": len(bots),
	})
}

// GetBot godoc
// @Summary Get a Discord bot
// @Description Get bot details with associated channels
// @Tags Discord Bots
// @Produce json
// @Param botId path string true "Bot ID"
// @Success 200 {object} model.DiscordBotWithChannels
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Bot not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/bots/{botId} [get]
func (h *DiscordHandler) GetBot(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("botId")
	if botID == "" {
		respondError(w, http.StatusBadRequest, "bot ID required")
		return
	}

	bot, err := h.discordRepo.GetBot(r.Context(), botID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if bot == nil {
		respondError(w, http.StatusNotFound, "bot not found")
		return
	}

	// Get associated channels
	channels, _ := h.discordRepo.ListChannelsByBot(r.Context(), botID)

	respondJSON(w, http.StatusOK, model.DiscordBotWithChannels{
		DiscordBot: *bot,
		Channels:   channels,
	})
}

// UpdateBot godoc
// @Summary Update a Discord bot
// @Description Update bot settings or credentials
// @Tags Discord Bots
// @Accept json
// @Produce json
// @Param botId path string true "Bot ID"
// @Param request body model.UpdateDiscordBotRequest true "Update data"
// @Success 200 {object} model.DiscordBot
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/bots/{botId} [put]
func (h *DiscordHandler) UpdateBot(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("botId")
	if botID == "" {
		respondError(w, http.StatusBadRequest, "bot ID required")
		return
	}

	var req model.UpdateDiscordBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	bot, err := h.discordRepo.UpdateBot(r.Context(), botID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, bot)
}

// DeleteBot godoc
// @Summary Delete a Discord bot
// @Description Delete a bot and all associated channels
// @Tags Discord Bots
// @Produce json
// @Param botId path string true "Bot ID"
// @Success 200 {object} map[string]string "Deletion status"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Bot not found"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/bots/{botId} [delete]
func (h *DiscordHandler) DeleteBot(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("botId")
	if botID == "" {
		respondError(w, http.StatusBadRequest, "bot ID required")
		return
	}

	if err := h.discordRepo.DeleteBot(r.Context(), botID); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Channel handlers

// CreateChannel godoc
// @Summary Create a Discord channel config
// @Description Register a channel with webhook for a bot
// @Tags Discord Channels
// @Accept json
// @Produce json
// @Param request body model.CreateDiscordChannelRequest true "Channel configuration"
// @Success 201 {object} model.DiscordChannel
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/channels [post]
func (h *DiscordHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req model.CreateDiscordChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.BotID == "" || req.ChannelID == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "bot_id, channel_id, and name are required")
		return
	}

	channel, err := h.discordRepo.CreateChannel(r.Context(), &req, claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, channel)
}

// ListChannels godoc
// @Summary List Discord channels
// @Description Get all channels, optionally filtered by bot
// @Tags Discord Channels
// @Produce json
// @Param bot_id query string false "Filter by bot ID"
// @Success 200 {object} map[string]interface{} "List of channels"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/channels [get]
func (h *DiscordHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	botID := r.URL.Query().Get("bot_id")

	var channels []model.DiscordChannel
	var err error

	if botID != "" {
		channels, err = h.discordRepo.ListChannelsByBot(r.Context(), botID)
	} else {
		channels, err = h.discordRepo.ListAllChannels(r.Context())
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
		"total":    len(channels),
	})
}

// GetChannel godoc
// @Summary Get a Discord channel
// @Description Get channel details
// @Tags Discord Channels
// @Produce json
// @Param channelId path string true "Channel config ID"
// @Success 200 {object} model.DiscordChannel
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Channel not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/channels/{channelId} [get]
func (h *DiscordHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channelId")
	if channelID == "" {
		respondError(w, http.StatusBadRequest, "channel ID required")
		return
	}

	channel, err := h.discordRepo.GetChannel(r.Context(), channelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if channel == nil {
		respondError(w, http.StatusNotFound, "channel not found")
		return
	}

	respondJSON(w, http.StatusOK, channel)
}

// UpdateChannel godoc
// @Summary Update a Discord channel
// @Description Update channel settings or webhook
// @Tags Discord Channels
// @Accept json
// @Produce json
// @Param channelId path string true "Channel config ID"
// @Param request body model.UpdateDiscordChannelRequest true "Update data"
// @Success 200 {object} model.DiscordChannel
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/channels/{channelId} [put]
func (h *DiscordHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channelId")
	if channelID == "" {
		respondError(w, http.StatusBadRequest, "channel ID required")
		return
	}

	var req model.UpdateDiscordChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	channel, err := h.discordRepo.UpdateChannel(r.Context(), channelID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, channel)
}

// DeleteChannel godoc
// @Summary Delete a Discord channel
// @Description Remove a channel configuration
// @Tags Discord Channels
// @Produce json
// @Param channelId path string true "Channel config ID"
// @Success 200 {object} map[string]string "Deletion status"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Channel not found"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/channels/{channelId} [delete]
func (h *DiscordHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channelId")
	if channelID == "" {
		respondError(w, http.StatusBadRequest, "channel ID required")
		return
	}

	if err := h.discordRepo.DeleteChannel(r.Context(), channelID); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Task Discord Config handlers

// SetTaskDiscordConfig godoc
// @Summary Set task Discord config
// @Description Configure which bot/channel/webhook a task uses for notifications
// @Tags Task Discord Config
// @Accept json
// @Produce json
// @Param taskId path string true "Task ID"
// @Param request body model.SetTaskDiscordConfigRequest true "Discord configuration"
// @Success 200 {object} model.TaskDiscordConfig
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{taskId}/discord [put]
func (h *DiscordHandler) SetTaskDiscordConfig(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	var req model.SetTaskDiscordConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	config, err := h.discordRepo.SetTaskConfig(r.Context(), taskID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// GetTaskDiscordConfig godoc
// @Summary Get task Discord config
// @Description Get the Discord configuration for a task
// @Tags Task Discord Config
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {object} model.TaskDiscordConfig
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Config not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{taskId}/discord [get]
func (h *DiscordHandler) GetTaskDiscordConfig(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	config, err := h.discordRepo.GetTaskConfig(r.Context(), taskID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if config == nil {
		respondError(w, http.StatusNotFound, "no Discord config for this task")
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// DeleteTaskDiscordConfig godoc
// @Summary Delete task Discord config
// @Description Remove the Discord configuration for a task
// @Tags Task Discord Config
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {object} map[string]string "Deletion status"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{taskId}/discord [delete]
func (h *DiscordHandler) DeleteTaskDiscordConfig(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	if err := h.discordRepo.DeleteTaskConfig(r.Context(), taskID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// TestWebhook godoc
// @Summary Test Discord webhook
// @Description Send a test message to verify webhook configuration
// @Tags Discord
// @Accept json
// @Produce json
// @Param request body object{webhook_url=string,channel_id=string,message=string} true "Test webhook request"
// @Success 200 {object} map[string]interface{} "Test result"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Webhook error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /discord/test [post]
func (h *DiscordHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WebhookURL string `json:"webhook_url"`
		ChannelID  string `json:"channel_id"`
		Message    string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	webhookURL := req.WebhookURL

	// If channel_id provided, get webhook from channel
	if webhookURL == "" && req.ChannelID != "" {
		channel, err := h.discordRepo.GetChannelWithWebhook(r.Context(), req.ChannelID)
		if err != nil || channel == nil {
			respondError(w, http.StatusBadRequest, "channel not found or has no webhook")
			return
		}
		webhookURL = channel.WebhookURL
	}

	if webhookURL == "" {
		respondError(w, http.StatusBadRequest, "webhook_url or channel_id with webhook required")
		return
	}

	message := req.Message
	if message == "" {
		message = "🧪 Test message from Multi-Worker scheduler!"
	}

	// Send test message to Discord webhook
	payload := map[string]interface{}{
		"content": message,
		"embeds": []map[string]interface{}{
			{
				"title":       "Webhook Test",
				"description": "This is a test message to verify webhook configuration.",
				"color":       5814783,
				"footer": map[string]string{
					"text": "Multi-Worker Scheduler",
				},
			},
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to send test message: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		respondError(w, http.StatusBadGateway, "Discord returned error: "+string(body))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "sent",
		"message": message,
	})
}
