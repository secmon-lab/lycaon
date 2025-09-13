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
	"net/url"
	"strconv"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/secmon-lab/lycaon/pkg/utils/async"
	"github.com/slack-go/slack/slackevents"
)

// Handler handles Slack webhook endpoints
type Handler struct {
	slackConfig        *config.SlackConfig
	messageUC          interfaces.SlackMessage
	incidentUC         interfaces.Incident
	taskUC             interfaces.Task
	eventHandler       *EventHandler
	interactionHandler *InteractionHandler
}

// NewHandler creates a new Slack handler
func NewHandler(ctx context.Context, slackConfig *config.SlackConfig, repo interfaces.Repository, messageUC interfaces.SlackMessage, incidentUC interfaces.Incident, taskUC interfaces.Task, slackInteractionUC interfaces.SlackInteraction, slackClient interfaces.SlackClient) *Handler {
	statusUC := usecase.NewStatusUseCase(repo, slackClient)
	return &Handler{
		slackConfig:        slackConfig,
		messageUC:          messageUC,
		incidentUC:         incidentUC,
		taskUC:             taskUC,
		eventHandler:       NewEventHandler(ctx, messageUC, taskUC, incidentUC, statusUC, slackClient),
		interactionHandler: NewInteractionHandler(ctx, slackInteractionUC),
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

	// Log token and app ID for debugging multiple app issue
	ctxlog.From(r.Context()).Debug("Parsed Slack event",
		"token", eventsAPIEvent.Token,
		"api_app_id", eventsAPIEvent.APIAppID,
		"team_id", eventsAPIEvent.TeamID,
		"type", eventsAPIEvent.Type,
	)

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
	if err := h.verifySlackSignature(r.Context(), r, body); err != nil {
		ctxlog.From(r.Context()).Warn("Invalid Slack signature", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "invalid signature"), http.StatusUnauthorized)
		return
	}

	// Handle callback events
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		// Log event details for debugging
		ctxlog.From(r.Context()).Debug("Processing Slack event",
			"type", eventsAPIEvent.Type,
			"team_id", eventsAPIEvent.TeamID,
			"api_app_id", eventsAPIEvent.APIAppID,
		)

		// Acknowledge receipt immediately
		w.WriteHeader(http.StatusOK)

		// Process event asynchronously with preserved context
		backgroundCtx := async.NewBackgroundContext(r.Context())
		async.Dispatch(backgroundCtx, func(ctx context.Context) error {
			return h.eventHandler.HandleEvent(ctx, &eventsAPIEvent)
		})
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

	// Read the raw body first for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ctxlog.From(r.Context()).Error("Failed to read request body", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "failed to read request body"), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature using the raw body
	if err := h.verifySlackSignature(r.Context(), r, body); err != nil {
		ctxlog.From(r.Context()).Warn("Invalid Slack signature for interaction", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "invalid signature"), http.StatusUnauthorized)
		return
	}

	// Parse the form data from the body
	values, err := url.ParseQuery(string(body))
	if err != nil {
		ctxlog.From(r.Context()).Error("Failed to parse form data", "error", err)
		h.writeError(w, r.Context(), goerr.Wrap(err, "failed to parse form data"), http.StatusBadRequest)
		return
	}

	// Get payload
	payload := values.Get("payload")
	if payload == "" {
		h.writeError(w, r.Context(), goerr.New("payload not found"), http.StatusBadRequest)
		return
	}

	// Acknowledge receipt immediately
	w.WriteHeader(http.StatusOK)

	// Process interaction asynchronously with preserved context
	backgroundCtx := async.NewBackgroundContext(r.Context())
	async.Dispatch(backgroundCtx, func(ctx context.Context) error {
		return h.interactionHandler.HandleInteraction(ctx, []byte(payload))
	})
}

// verifySlackSignature verifies the Slack request signature
func (h *Handler) verifySlackSignature(ctx context.Context, r *http.Request, body []byte) error {
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

	// Debug logging for signature verification
	bodyPreview := string(body)
	if len(bodyPreview) > 100 {
		bodyPreview = bodyPreview[:100] + "..."
	}

	// Hash the body for comparison
	bodyHash := sha256.Sum256(body)
	bodyHashHex := hex.EncodeToString(bodyHash[:])

	ctxlog.From(ctx).Debug("Verifying Slack signature",
		"timestamp", timestamp,
		"received_signature", signature,
		"expected_signature", expectedSignature,
		"signing_secret_length", len(h.slackConfig.SigningSecret),
		"body_length", len(body),
		"body_hash", bodyHashHex[:16], // First 16 chars of hash for identification
		"body_preview", bodyPreview,
	)

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		ctxlog.From(ctx).Warn("Signature mismatch",
			"received", signature,
			"expected", expectedSignature,
		)
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

	// Log error with context
	ctxlog.From(ctx).Debug("Writing error response", "status", status, "error", message)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		ctxlog.From(ctx).Error("Failed to encode error response", "error", err)
	}
}

// abs returns the absolute value of an int64
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
