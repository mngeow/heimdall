package validation

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

// Dependencies represents the external executables Symphony depends on
type Dependencies struct {
	Git      string
	OpenSpec string
	OpenCode string
}

// DefaultDependencies returns the default dependency configuration
func DefaultDependencies() *Dependencies {
	return &Dependencies{
		Git:      "git",
		OpenSpec: "openspec",
		OpenCode: "opencode",
	}
}

// Validate checks that all required executables are available
func (d *Dependencies) Validate(ctx context.Context) error {
	logger := slog.Default()

	deps := []struct {
		name string
		cmd  string
	}{
		{"git", d.Git},
		{"openspec", d.OpenSpec},
		{"opencode", d.OpenCode},
	}

	for _, dep := range deps {
		logger.Info("checking dependency", "name", dep.name)

		cmd := exec.CommandContext(ctx, dep.cmd, "--version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("dependency %q not available: %w (output: %s)", dep.name, err, string(output))
		}

		logger.Info("dependency available", "name", dep.name, "version", string(output))
	}

	return nil
}
