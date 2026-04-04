package httpserver

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/mngeow/symphony/internal/scm/github"
)

// WebhookHandler handles GitHub webhook events
type WebhookHandler struct {
	client *github.Client
	logger *slog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(client *github.Client, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		client: client,
		logger: logger,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get signature from header
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		h.logger.Warn("missing webhook signature")
		http.Error(w, "missing signature", http.StatusBadRequest)
		return
	}

	// Read payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature
	if err := h.client.VerifyWebhookSignature(payload, signature); err != nil {
		h.logger.Warn("signature verification failed", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Get event type
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		h.logger.Warn("missing event type")
		http.Error(w, "missing event type", http.StatusBadRequest)
		return
	}

	h.logger.Info("webhook received", "event", eventType)

	// Parse and handle event
	event, err := h.client.ParseWebhook(eventType, payload)
	if err != nil {
		h.logger.Error("failed to parse webhook", "error", err)
		http.Error(w, "failed to parse webhook", http.StatusBadRequest)
		return
	}

	// Handle specific event types
	switch event.Type {
	case "issue_comment":
		h.handleIssueComment(event.Payload)
	case "pull_request":
		h.handlePullRequest(event.Payload)
	default:
		h.logger.Info("unhandled event type", "event", eventType)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleIssueComment(payload interface{}) {
	// TODO: Implement command parsing and queueing
	h.logger.Info("issue comment received")
}

func (h *WebhookHandler) handlePullRequest(payload interface{}) {
	// TODO: Implement PR lifecycle handling
	h.logger.Info("pull request event received")
}
