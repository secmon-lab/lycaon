package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack/slackevents"
)

// Handler handles Slack webhook endpoints
type Handler struct {
	slackConfig        *config.SlackConfig
	messageUC          usecase.SlackMessageUseCase
	eventHandler       *EventHandler
	interactionHandler *InteractionHandler
}

// NewHandler creates a new Slack handler
func NewHandler(ctx context.Context, slackConfig *config.SlackConfig, messageUC usecase.SlackMessageUseCase) *Handler {
	return &Handler{
		slackConfig:        slackConfig,
		messageUC:          messageUC,
		eventHandler:       NewEventHandler(ctx, messageUC),
		interactionHandler: NewInteractionHandler(ctx),
	}
}

// HandleEvent handles a single Slack event
func (h *Handler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ctxlog.From(r.Context()).Error("Failed to read request body", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "failed to read request body"), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse event first to check if it's URL verification
	eventsAPIEvent, err := slackevents.ParseEvent(body, slackevents.OptionNoVerifyToken())
	if err != nil {
		ctxlog.From(r.Context()).Error("Failed to parse Slack event", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "failed to parse event"), http.StatusBadRequest)
		return
	}

	// Handle URL verification challenge (no auth needed)
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var response *slackevents.ChallengeResponse
		if err := json.Unmarshal(body, &response); err != nil {
			ctxlog.From(r.Context()).Error("Failed to parse challenge", "error", err)
			h.writeError(w, r.Context(), goerr.Wrap(err, "failed to parse challenge"), http.StatusBadRequest)
			return
		}

		ctxlog.From(r.Context()).Info("Responding to Slack URL verification challenge")
		w.Header().Set("Content-Type", "text")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(response.Challenge)); err != nil {
			ctxlog.From(r.Context()).Error("Failed to write challenge response", "error", err)
		}
		return
	}

	// For other events, check configuration
	if !h.slackConfig.IsConfigured() {
		h.writeError(w, r.Context(), goerr.New("Slack not configured"), http.StatusServiceUnavailable)
		return
	}

	// Verify Slack signature
	if err := h.verifySlackSignature(r, body); err != nil {
		ctxlog.From(r.Context()).Warn("Invalid Slack signature", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "invalid signature"), http.StatusUnauthorized)
		return
	}

	// Handle callback events
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		// Acknowledge receipt immediately
		w.WriteHeader(http.StatusOK)

		// Process event asynchronously
		go func(ctx context.Context) {
			if err := h.eventHandler.HandleEvent(ctx, &eventsAPIEvent); err != nil {
				ctxlog.From(ctx).Error("Failed to handle event",
					"error", err,
					"eventType", eventsAPIEvent.Type,
				)
			}
		}(r.Context())
		return
	}

	// Unknown event type
	ctxlog.From(r.Context()).Warn("Unknown Slack event type", "type", eventsAPIEvent.Type)
	w.WriteHeader(http.StatusOK)
}

// HandleInteraction handles a single Slack interaction
func (h *Handler) HandleInteraction(w http.ResponseWriter, r *http.Request) {
	if !h.slackConfig.IsConfigured() {
		h.writeError(w, r.Context(), goerr.New("Slack not configured"), http.StatusServiceUnavailable)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		ctxlog.From(r.Context()).Error("Failed to parse form", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "failed to parse form"), http.StatusBadRequest)
		return
	}

	// Get payload
	payload := r.FormValue("payload")
	if payload == "" {
		h.writeError(w, r.Context(), goerr.New("payload not found"), http.StatusBadRequest)
		return
	}

	// Verify signature using the raw form data
	body := []byte("payload=" + payload)
	if err := h.verifySlackSignature(r, body); err != nil {
		ctxlog.From(r.Context()).Warn("Invalid Slack signature for interaction", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "invalid signature"), http.StatusUnauthorized)
		return
	}

	// Acknowledge receipt
	w.WriteHeader(http.StatusOK)

	// Process interaction asynchronously
	go func(ctx context.Context) {
		if err := h.interactionHandler.HandleInteraction(ctx, []byte(payload)); err != nil {
			ctxlog.From(ctx).Error("Failed to handle interaction", "error", err)
		}
	}(r.Context())
}

// verifySlackSignature verifies the Slack request signature
func (h *Handler) verifySlackSignature(r *http.Request, body []byte) error {
	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	if timestamp == "" {
		return goerr.New("missing timestamp header")
	}

	// Check timestamp to prevent replay attacks (5 minute window)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return goerr.Wrap(err, "invalid timestamp")
	}

	if abs(time.Now().Unix()-ts) > 60*5 {
		return goerr.New("timestamp too old")
	}

	signature := r.Header.Get("X-Slack-Signature")
	if signature == "" {
		return goerr.New("missing signature header")
	}

	// Compute expected signature
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(h.slackConfig.SigningSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return goerr.New("signature mismatch")
	}

	return nil
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, ctx context.Context, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var message string
	if goErr := goerr.Unwrap(err); goErr != nil {
		message = goErr.Error()
	} else {
		message = err.Error()
	}

	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// abs returns the absolute value of an int64
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
