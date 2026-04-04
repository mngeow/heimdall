package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/mngeow/symphony/internal/config"
	"github.com/mngeow/symphony/internal/scm/github"
	"github.com/mngeow/symphony/internal/slashcmd"
	"github.com/mngeow/symphony/internal/store"
	"github.com/mngeow/symphony/internal/validation"
)

// App represents the Symphony application
type App struct {
	config        *config.Config
	logger        *slog.Logger
	deps          *validation.Dependencies
	store         *store.Store
	githubPoller  *github.Poller
	commandIntake *slashcmd.Intake
	ready         bool
}

// New creates a new Symphony application
func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	runtimeStore, err := store.New(cfg.Storage.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open runtime store: %w", err)
	}

	if err := runtimeStore.Migrate(ctx); err != nil {
		runtimeStore.Close()
		return nil, fmt.Errorf("failed to migrate runtime store: %w", err)
	}

	if err := syncConfiguredRepositories(ctx, runtimeStore, cfg.Repos); err != nil {
		runtimeStore.Close()
		return nil, fmt.Errorf("failed to sync configured repositories: %w", err)
	}

	githubClient, err := github.NewClient(cfg.GitHub)
	if err != nil {
		runtimeStore.Close()
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}

	queue := store.NewJobQueue(runtimeStore)

	return &App{
		config:        cfg,
		logger:        logger,
		deps:          validation.DefaultDependencies(),
		store:         runtimeStore,
		githubPoller:  github.NewPoller(githubClient, runtimeStore, cfg.GitHub.LookbackWindow),
		commandIntake: slashcmd.NewIntake(runtimeStore, queue, logger),
		ready:         false,
	}, nil
}

// Run starts the application
func (a *App) Run(ctx context.Context) error {
	defer a.store.Close()

	a.logger.Info("starting Symphony", "version", "v0.1.0")

	// Validate dependencies before marking ready
	a.logger.Info("validating dependencies")
	if err := a.deps.Validate(ctx); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}
	a.ready = true
	a.logger.Info("dependencies validated, service ready")

	go a.runGitHubPolling(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", a.healthHandler)
	mux.HandleFunc("/readyz", a.readyHandler)

	server := &http.Server{
		Addr:    a.config.Server.ListenAddress,
		Handler: mux,
	}

	a.logger.Info("HTTP server starting", "addr", server.Addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *App) runGitHubPolling(ctx context.Context) {
	if err := a.pollGitHubOnce(ctx); err != nil {
		a.logger.Error("github polling cycle failed", "error", err)
	}

	ticker := time.NewTicker(a.config.GitHub.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.pollGitHubOnce(ctx); err != nil {
				a.logger.Error("github polling cycle failed", "error", err)
			}
		}
	}
}

func (a *App) pollGitHubOnce(ctx context.Context) error {
	result, err := a.githubPoller.Poll(ctx)
	if err != nil {
		return err
	}

	for _, command := range result.Commands {
		repoConfig, ok := a.repoConfigByRef(command.RepoRef)
		if !ok {
			a.logger.Warn("github poll observed unmanaged repo config", "repo", command.RepoRef)
			continue
		}

		outcome, err := a.commandIntake.Process(ctx, repoConfig, command.PullRequest, command.CommentNodeID, command.ActorLogin, command.Body)
		if err != nil {
			return fmt.Errorf("failed to process github command for %s#%d: %w", command.RepoRef, command.PullRequest.Number, err)
		}

		if outcome.Status != "ignored" {
			a.logger.Info(
				"processed github command observation",
				"repo", command.RepoRef,
				"pr", command.PullRequest.Number,
				"status", outcome.Status,
				"duplicate", outcome.Duplicate,
			)
		}
	}

	if len(result.Reconciled) > 0 {
		a.logger.Info("reconciled managed pull requests", "count", len(result.Reconciled))
	}

	return nil
}

func (a *App) repoConfigByRef(repoRef string) (config.RepoConfig, bool) {
	for _, repo := range a.config.Repos {
		if repo.Name == repoRef {
			return repo, true
		}
	}

	return config.RepoConfig{}, false
}

func syncConfiguredRepositories(ctx context.Context, runtimeStore *store.Store, repos []config.RepoConfig) error {
	for _, repoConfig := range repos {
		owner, name, err := github.ParseRepoRef(repoConfig.Name)
		if err != nil {
			return err
		}

		repository := &store.Repository{
			Provider:        "github",
			RepoRef:         repoConfig.Name,
			Owner:           owner,
			Name:            name,
			DefaultBranch:   repoConfig.DefaultBranch,
			BranchPrefix:    repoConfig.BranchPrefix,
			LocalMirrorPath: repoConfig.LocalMirrorPath,
			IsActive:        true,
		}
		if repository.DefaultBranch == "" {
			repository.DefaultBranch = "main"
		}
		if repository.BranchPrefix == "" {
			repository.BranchPrefix = "symphony"
		}

		if err := runtimeStore.SaveRepository(ctx, repository); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (a *App) readyHandler(w http.ResponseWriter, r *http.Request) {
	if !a.ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"not ready"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}
