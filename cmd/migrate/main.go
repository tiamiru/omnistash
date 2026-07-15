package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/migration"
)

var errFlagDSNRequired = errors.New("-dsn is required")

func main() {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	metastoreDSN := fs.String("dsn", "", "SQLite database path (required)")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("migrate", logtag.Err(err))
		os.Exit(2) //nolint:mnd
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	if *metastoreDSN == "" {
		logger.Error("migrate", logtag.Err(errFlagDSNRequired))
		os.Exit(2) //nolint:mnd
	}

	err = migration.ApplySQLiteMigrations(context.Background(), *metastoreDSN)
	if err != nil {
		logger.Error("migrate", logtag.Err(err))
		os.Exit(1)
	}
}
