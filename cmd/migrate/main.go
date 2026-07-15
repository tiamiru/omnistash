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

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	var metastoreDSN string
	fs.StringVar(&metastoreDSN, "dsn", "", "SQLite database path (required)")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		logger.Error("migrate", logtag.Err(err))
		os.Exit(2) //nolint:mnd
	}

	if metastoreDSN == "" {
		logger.Error("migrate", logtag.Err(errFlagDSNRequired))
		os.Exit(2) //nolint:mnd
	}

	err = migration.ApplySQLiteMigrations(context.Background(), metastoreDSN)
	if err != nil {
		logger.Error("migrate", logtag.Err(err))
		os.Exit(1)
	}
}
