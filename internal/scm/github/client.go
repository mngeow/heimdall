package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/go-github/v57/github"
	"github.com/mngeow/symphony/internal/config"
)

// Client wraps GitHub App authentication and API operations
type Client struct {
	appID          string
	privateKey     []byte
	webhookSecret  string
	client         *github.Client
	installationID int64
}

// NewClient creates a new GitHub client
func NewClient(cfg config.GitHubConfig) (*Client, error) {
	return &Client{
		appID:         cfg.AppID,
		privateKey:    []byte(cfg.PrivateKey),
		webhookSecret: cfg.WebhookSecret,
	}, nil
}

// VerifyWebhookSignature validates the GitHub webhook signature
func (c *Client) VerifyWebhookSignature(payload []byte, signature string) error {
	if c.webhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	// Signature format: "sha256=<hex>"
	if len(signature) < 7 || signature[:7] != "sha256=" {
		return fmt.Errorf("invalid signature format")
	}

	expectedMAC := hmac.New(sha256.New, []byte(c.webhookSecret))
	expectedMAC.Write(payload)
	expectedSig := hex.EncodeToString(expectedMAC.Sum(nil))

	if !hmac.Equal([]byte(signature[7:]), []byte(expectedSig)) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// GetInstallationToken retrieves an installation token for API operations
func (c *Client) GetInstallationToken(ctx context.Context) (string, error) {
	// TODO: Implement JWT generation and installation token exchange
	// This requires parsing the private key and making a GitHub API call
	return "", fmt.Errorf("installation token not yet implemented")
}

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	Type       string
	Action     string
	Payload    interface{}
	DeliveryID string
}

// ParseWebhook parses a GitHub webhook payload
func (c *Client) ParseWebhook(eventType string, payload []byte) (*WebhookEvent, error) {
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	return &WebhookEvent{
		Type:    eventType,
		Payload: event,
	}, nil
}

// CreatePullRequest creates or updates a pull request
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo, title, head, base, body string) (*github.PullRequest, error) {
	// TODO: Implement using go-github client with installation token
	return nil, fmt.Errorf("not yet implemented")
}

// CreateComment adds a comment to a pull request
func (c *Client) CreateComment(ctx context.Context, owner, repo string, number int, body string) error {
	// TODO: Implement using go-github client
	return fmt.Errorf("not yet implemented")
}

// GetPullRequest retrieves a pull request by number
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	// TODO: Implement using go-github client
	return nil, fmt.Errorf("not yet implemented")
}
