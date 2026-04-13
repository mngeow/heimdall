package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mngeow/heimdall/internal/store"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var dsn string
	flag.StringVar(&dsn, "dsn", "", "SQLite database path")
	flag.Parse()

	if dsn == "" {
		return fmt.Errorf("missing required --dsn flag")
	}

	runtimeStore, err := store.New(dsn)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database %q: %w", dsn, err)
	}
	defer runtimeStore.Close()

	if err := runtimeStore.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to migrate SQLite database %q: %w", dsn, err)
	}

	fmt.Printf("initialized Heimdall SQLite schema at %s\n", dsn)
	return nil
}
