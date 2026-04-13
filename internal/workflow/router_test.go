package workflow

import (
	"testing"

	"github.com/mngeow/heimdall/internal/config"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Add rate limiting", "add-rate-limiting"},
		{"Fix bug in API", "fix-bug-in-api"},
		{"Update documentation", "update-documentation"},
		{"Test123", "test123"},
		{"Add   rate limiting!!!", "add-rate-limiting"},
		{"###", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateBranchName(t *testing.T) {
	result := GenerateBranchName("heimdall", "ENG-123", "add-rate-limiting")
	expected := "heimdall/ENG-123-add-rate-limiting"
	if result != expected {
		t.Errorf("GenerateBranchName() = %q, want %q", result, expected)
	}
}

func TestSlugFromDescriptionOrTitle(t *testing.T) {
	tests := []struct {
		name        string
		description string
		title       string
		want        string
	}{
		{
			name:        "description wins when usable",
			description: "Add rate limiting for API requests",
			title:       "Add rate limiting",
			want:        "add-rate-limiting-for-api-requests",
		},
		{
			name:        "title fallback when description empty",
			description: "",
			title:       "Add rate limiting",
			want:        "add-rate-limiting",
		},
		{
			name:        "title fallback when description unusable",
			description: "###",
			title:       "Add rate limiting",
			want:        "add-rate-limiting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlugFromDescriptionOrTitle(tt.description, tt.title); got != tt.want {
				t.Fatalf("SlugFromDescriptionOrTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateChangeName(t *testing.T) {
	result := GenerateChangeName("ENG-123", "add-rate-limiting")
	expected := "ENG-123-add-rate-limiting"
	if result != expected {
		t.Errorf("GenerateChangeName() = %q, want %q", result, expected)
	}
}

func TestGenerateWorktreePath(t *testing.T) {
	result := GenerateWorktreePath("/var/lib/heimdall/repos/github.com/acme/platform.git", "heimdall/ENG-123-add-rate-limiting")
	expected := "/var/lib/heimdall/repos/github.com/acme/platform-worktrees/heimdall-ENG-123-add-rate-limiting"
	if result != expected {
		t.Fatalf("GenerateWorktreePath() = %q, want %q", result, expected)
	}
}

func TestRouter(t *testing.T) {
	repos := []config.RepoConfig{
		{
			Name:           "github.com/acme/platform",
			LinearTeamKeys: []string{"ENG", "PLATFORM"},
		},
		{
			Name:           "github.com/acme/mobile",
			LinearTeamKeys: []string{"MOBILE"},
		},
	}

	router := NewRouter(repos)

	t.Run("RouteByTeamKey", func(t *testing.T) {
		result := router.Resolve("ENG")
		if !result.Matched {
			t.Error("expected match for ENG team")
		}
		if result.Repository.Name != "github.com/acme/platform" {
			t.Errorf("expected platform repo, got %s", result.Repository.Name)
		}
	})

	t.Run("NoMatch", func(t *testing.T) {
		result := router.Resolve("DESIGN")
		if result.Matched {
			t.Error("expected no match for DESIGN team")
		}
	})
}
