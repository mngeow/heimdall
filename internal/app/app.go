package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/mngeow/symphony/internal/config"
	"github.com/mngeow/symphony/internal/validation"
)

// App represents the Symphony application
type App struct {
	config *config.Config
	logger *slog.Logger
	deps   *validation.Dependencies
	ready  bool
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

	return &App{
		config: cfg,
		logger: logger,
		deps:   validation.DefaultDependencies(),
		ready:  false,
	}, nil
}

// Run starts the application
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting Symphony", "version", "v0.1.0")

	// Validate dependencies before marking ready
	a.logger.Info("validating dependencies")
	if err := a.deps.Validate(ctx); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}
	a.ready = true
	a.logger.Info("dependencies validated, service ready")

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

	return server.ListenAndServe()
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
