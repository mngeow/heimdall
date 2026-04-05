package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	gh "github.com/google/go-github/v57/github"
	"github.com/mngeow/symphony/internal/config"
)

var defaultGitHubAPIBaseURL = mustParseURL("https://api.github.com/")

const (
	defaultPRMonitorLabelColor       = "0e8a16"
	defaultPRMonitorLabelDescription = "Symphony monitors PR events on this pull request."
)

// Client wraps GitHub App authentication and API operations.
type Client struct {
	appID          string
	installationID int64
	privateKey     *rsa.PrivateKey
	httpClient     *http.Client
	apiBaseURL     *url.URL
	now            func() time.Time
}

// NewClient creates a new GitHub client.
func NewClient(cfg config.GitHubConfig) (*Client, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("github app id not configured")
	}
	if cfg.InstallationID == 0 {
		return nil, fmt.Errorf("github installation id not configured")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("github private key not configured")
	}

	privateKey, err := parsePrivateKey([]byte(cfg.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse github private key: %w", err)
	}

	return &Client{
		appID:          cfg.AppID,
		installationID: cfg.InstallationID,
		privateKey:     privateKey,
		httpClient:     http.DefaultClient,
		apiBaseURL:     defaultGitHubAPIBaseURL,
		now:            time.Now,
	}, nil
}

// GetInstallationToken retrieves an installation token for API operations.
func (c *Client) GetInstallationToken(ctx context.Context) (string, error) {
	appJWT, err := c.signAppJWT()
	if err != nil {
		return "", err
	}

	endpoint, err := c.apiBaseURL.Parse(fmt.Sprintf("app/installations/%d/access_tokens", c.installationID))
	if err != nil {
		return "", fmt.Errorf("failed to build installation token endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create installation token request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("installation token request failed with status %s", resp.Status)
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode installation token response: %w", err)
	}
	if payload.Token == "" {
		return "", fmt.Errorf("installation token response did not include a token")
	}

	return payload.Token, nil
}

// ListIssueCommentsSince lists repository issue comments updated since the given time.
func (c *Client) ListIssueCommentsSince(ctx context.Context, owner, repo string, since time.Time) ([]*gh.IssueComment, error) {
	apiClient, err := c.newAPIClient(ctx)
	if err != nil {
		return nil, err
	}

	options := &gh.IssueListCommentsOptions{
		Sort:      gh.String("updated"),
		Direction: gh.String("asc"),
		Since:     &since,
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	var comments []*gh.IssueComment
	for {
		pageComments, response, err := apiClient.Issues.ListComments(ctx, owner, repo, 0, options)
		if err != nil {
			return nil, fmt.Errorf("failed to list issue comments for %s/%s: %w", owner, repo, err)
		}
		comments = append(comments, pageComments...)
		if response.NextPage == 0 {
			break
		}
		options.Page = response.NextPage
	}

	return comments, nil
}

// CreatePullRequest creates or updates a pull request.
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo, title, head, base, body string) (*gh.PullRequest, error) {
	apiClient, err := c.newAPIClient(ctx)
	if err != nil {
		return nil, err
	}

	pullRequest, _, err := apiClient.PullRequests.Create(ctx, owner, repo, &gh.NewPullRequest{
		Title: gh.String(title),
		Head:  gh.String(head),
		Base:  gh.String(base),
		Body:  gh.String(body),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request for %s/%s: %w", owner, repo, err)
	}

	return pullRequest, nil
}

// EnsurePRMonitorLabel creates the configured PR monitor label when it is missing.
func (c *Client) EnsurePRMonitorLabel(ctx context.Context, owner, repo, labelName string) error {
	apiClient, err := c.newAPIClient(ctx)
	if err != nil {
		return err
	}

	_, _, err = apiClient.Issues.GetLabel(ctx, owner, repo, labelName)
	if err == nil {
		return nil
	}

	var responseErr *gh.ErrorResponse
	if !errors.As(err, &responseErr) || responseErr.Response == nil || responseErr.Response.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to look up label %q in %s/%s: %w", labelName, owner, repo, err)
	}

	_, _, err = apiClient.Issues.CreateLabel(ctx, owner, repo, &gh.Label{
		Name:        gh.String(labelName),
		Color:       gh.String(defaultPRMonitorLabelColor),
		Description: gh.String(defaultPRMonitorLabelDescription),
	})
	if err != nil {
		return fmt.Errorf("failed to create label %q in %s/%s: %w", labelName, owner, repo, err)
	}

	return nil
}

// AddPRMonitorLabel adds the configured monitor label to a pull request without replacing unrelated labels.
func (c *Client) AddPRMonitorLabel(ctx context.Context, owner, repo string, number int, labelName string) error {
	apiClient, err := c.newAPIClient(ctx)
	if err != nil {
		return err
	}

	_, _, err = apiClient.Issues.AddLabelsToIssue(ctx, owner, repo, number, []string{labelName})
	if err != nil {
		return fmt.Errorf("failed to add label %q to %s/%s#%d: %w", labelName, owner, repo, number, err)
	}

	return nil
}

// FindOpenPullRequestByHead returns the first open pull request that matches the head/base branch pair.
func (c *Client) FindOpenPullRequestByHead(ctx context.Context, owner, repo, head, base string) (*gh.PullRequest, error) {
	apiClient, err := c.newAPIClient(ctx)
	if err != nil {
		return nil, err
	}

	options := &gh.PullRequestListOptions{
		State: "open",
		Head:  fmt.Sprintf("%s:%s", owner, head),
		Base:  base,
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	for {
		pullRequests, response, err := apiClient.PullRequests.List(ctx, owner, repo, options)
		if err != nil {
			return nil, fmt.Errorf("failed to list pull requests for %s/%s: %w", owner, repo, err)
		}
		if len(pullRequests) > 0 {
			return pullRequests[0], nil
		}
		if response.NextPage == 0 {
			break
		}
		options.Page = response.NextPage
	}

	return nil, nil
}

// CreateComment adds a comment to a pull request.
func (c *Client) CreateComment(ctx context.Context, owner, repo string, number int, body string) error {
	apiClient, err := c.newAPIClient(ctx)
	if err != nil {
		return err
	}

	_, _, err = apiClient.Issues.CreateComment(ctx, owner, repo, number, &gh.IssueComment{Body: gh.String(body)})
	if err != nil {
		return fmt.Errorf("failed to create comment on %s/%s#%d: %w", owner, repo, number, err)
	}

	return nil
}

// GetPullRequest retrieves a pull request by number.
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*gh.PullRequest, error) {
	apiClient, err := c.newAPIClient(ctx)
	if err != nil {
		return nil, err
	}

	pullRequest, _, err := apiClient.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request %s/%s#%d: %w", owner, repo, number, err)
	}

	return pullRequest, nil
}

func (c *Client) newAPIClient(ctx context.Context) (*gh.Client, error) {
	token, err := c.GetInstallationToken(ctx)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &installationTransport{
			token: token,
			base:  transportForClient(c.httpClient),
		},
	}

	apiClient := gh.NewClient(httpClient)
	apiClient.BaseURL = cloneURL(c.apiBaseURL)
	apiClient.UploadURL = cloneURL(c.apiBaseURL)
	return apiClient, nil
}

func (c *Client) signAppJWT() (string, error) {
	now := c.now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-1 * time.Minute)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
		Issuer:    c.appID,
	})

	signed, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign github app jwt: %w", err)
	}

	return signed, nil
}

func parsePrivateKey(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("pem block not found")
	}

	if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not rsa")
	}

	return privateKey, nil
}

func ParseRepoRef(repoRef string) (string, string, error) {
	trimmed := strings.TrimSpace(repoRef)
	trimmed = strings.TrimPrefix(trimmed, "https://github.com/")
	trimmed = strings.TrimPrefix(trimmed, "http://github.com/")
	trimmed = strings.TrimPrefix(trimmed, "github.com/")
	trimmed = strings.TrimPrefix(trimmed, "/")

	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid github repo ref %q", repoRef)
	}

	return parts[0], parts[1], nil
}

type installationTransport struct {
	token string
	base  http.RoundTripper
}

func (t *installationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = clone.Header.Clone()
	clone.Header.Set("Accept", "application/vnd.github+json")
	clone.Header.Set("Authorization", "token "+t.token)
	clone.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return t.base.RoundTrip(clone)
}

func cloneURL(u *url.URL) *url.URL {
	clone := *u
	if !strings.HasSuffix(clone.Path, "/") {
		clone.Path += "/"
	}
	return &clone
}

func transportForClient(client *http.Client) http.RoundTripper {
	if client != nil && client.Transport != nil {
		return client.Transport
	}
	return http.DefaultTransport
}

func mustParseURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}
