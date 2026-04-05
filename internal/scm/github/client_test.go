package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetInstallationToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/installations/99/access_tokens" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			t.Fatalf("expected bearer auth header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"installation-token"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	client.installationID = 99

	token, err := client.GetInstallationToken(t.Context())
	if err != nil {
		t.Fatalf("GetInstallationToken() error = %v", err)
	}
	if token != "installation-token" {
		t.Fatalf("GetInstallationToken() = %q, want installation-token", token)
	}
}

func TestListIssueCommentsSince(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token":"installation-token"}`))
		case "/repos/acme/platform/issues/comments":
			if got := r.Header.Get("Authorization"); got != "token installation-token" {
				t.Fatalf("expected token auth header, got %q", got)
			}
			if r.URL.Query().Get("sort") != "updated" {
				t.Fatalf("expected updated sort, got %q", r.URL.Query().Get("sort"))
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"id":101,"node_id":"IC_101","body":"/symphony status"}]`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)
	comments, err := client.ListIssueCommentsSince(t.Context(), "acme", "platform", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("ListIssueCommentsSince() error = %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].GetNodeID() != "IC_101" {
		t.Fatalf("expected node id IC_101, got %q", comments[0].GetNodeID())
	}
}

func TestFindOpenPullRequestByHead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token":"installation-token"}`))
		case "/repos/acme/platform/pulls":
			if got := r.URL.Query().Get("head"); got != "acme:symphony/ENG-123-add-rate-limiting" {
				t.Fatalf("expected head query, got %q", got)
			}
			if got := r.URL.Query().Get("base"); got != "main" {
				t.Fatalf("expected base query, got %q", got)
			}
			if got := r.URL.Query().Get("state"); got != "open" {
				t.Fatalf("expected open state, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"number":42,"node_id":"PR_node_42","title":"Bootstrap PR","state":"open","html_url":"https://example.test/pr/42"}]`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)
	pullRequest, err := client.FindOpenPullRequestByHead(t.Context(), "acme", "platform", "symphony/ENG-123-add-rate-limiting", "main")
	if err != nil {
		t.Fatalf("FindOpenPullRequestByHead() error = %v", err)
	}
	if pullRequest == nil || pullRequest.GetNumber() != 42 {
		t.Fatalf("expected open pull request #42, got %#v", pullRequest)
	}
}

func TestParseRepoRef(t *testing.T) {
	tests := []struct {
		name    string
		repoRef string
		owner   string
		repo    string
		wantErr bool
	}{
		{name: "github hostname", repoRef: "github.com/acme/platform", owner: "acme", repo: "platform"},
		{name: "plain owner repo", repoRef: "acme/platform", owner: "acme", repo: "platform"},
		{name: "invalid", repoRef: "acme", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepoRef(tt.repoRef)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRepoRef() error = %v", err)
			}
			if owner != tt.owner || repo != tt.repo {
				t.Fatalf("ParseRepoRef() = (%q, %q), want (%q, %q)", owner, repo, tt.owner, tt.repo)
			}
		})
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	apiBaseURL := mustParseURL(server.URL + "/")
	return &Client{
		appID:          "12345",
		installationID: 42,
		privateKey:     privateKey,
		httpClient:     server.Client(),
		apiBaseURL:     apiBaseURL,
		now: func() time.Time {
			return time.Unix(1_700_000_000, 0)
		},
	}
}

func TestParsePrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	encoded := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	parsed, err := parsePrivateKey(encoded)
	if err != nil {
		t.Fatalf("parsePrivateKey() error = %v", err)
	}
	if parsed.D.Cmp(privateKey.D) != 0 {
		t.Fatal("parsed key does not match original key")
	}
}
