package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tiamiru/omnistash/internal/blob"
	fsblobstore "github.com/tiamiru/omnistash/internal/blobstore/fs"
	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/internal/manifest"
	"github.com/tiamiru/omnistash/internal/metastore/sqlite"
	"github.com/tiamiru/omnistash/internal/namespace"
	"github.com/tiamiru/omnistash/rest"
)

const (
	shutdownTimeout = 30 * time.Second
)

var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals
	date    = "unknown" //nolint:gochecknoglobals
)

type config struct {
	addr          string
	metastoreDSN  string
	blobstorePath string
}

func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("omnistash", flag.ContinueOnError)

	var cfg config
	fs.StringVar(&cfg.addr, "addr", ":10080", "listen address")
	fs.StringVar(&cfg.metastoreDSN, "metastore-dsn", "omnistash.db", "SQLite database path")
	fs.StringVar(&cfg.blobstorePath, "blobstore-path", "blobs", "root directory for blob storage")

	err := fs.Parse(args)
	if err != nil {
		return config{}, err
	}

	return cfg, nil
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2) //nolint:mnd
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	err = run(logger, cfg)
	if err != nil {
		logger.Error("main", logtag.Err(err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger, cfg config) error {
	ctx := context.Background()

	meta, err := sqlite.NewSQLiteMetadataStore(ctx, cfg.metastoreDSN)
	if err != nil {
		return fmt.Errorf("open metastore: %w", err)
	}
	defer func() {
		closeErr := meta.Close()
		if closeErr != nil {
			logger.Warn("run: close metastore", logtag.Err(closeErr))
		}
	}()

	ns := namespace.NewService(meta)

	blobStore := fsblobstore.NewFilesystemBlobStore(cfg.blobstorePath, fsblobstore.WithLogger(logger))
	blobSvc := blob.NewService(meta, blobStore)

	manifestSvc := manifest.NewService(logger)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	handler := rest.NewRegistryHandler(logger, ns, blobSvc, manifestSvc, version, commit, date)
	server := rest.NewServer(handler, cfg.addr)

	logger.Info("main: server started", slog.String("addr", server.Addr))

	serveErrChan := make(chan error, 1)
	go func() {
		serveErrChan <- server.ListenAndServe()
	}()

	select {
	case sig := <-signalChan:
		logger.Info("main: shutting down", slog.String("signal", sig.String()))
	case err = <-serveErrChan:
		return fmt.Errorf("serve: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	err = <-serveErrChan
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
