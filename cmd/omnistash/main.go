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

	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/rest"
)

const (
	shutdownTimeout = 30 * time.Second
)

var (
	version = "dev"     //nolint:gochecknoglobals
	commit  = "none"    //nolint:gochecknoglobals
	date    = "unknown" //nolint:gochecknoglobals
)

type config struct {
	addr string
}

func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("omnistash", flag.ContinueOnError)

	var cfg config
	fs.StringVar(&cfg.addr, "addr", ":10080", "listen address")

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
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	handler := rest.NewRegistryHandler(logger, version, commit, date)
	server := rest.NewServer(handler, cfg.addr)

	logger.Info("main: server started", slog.String("addr", server.Addr))

	serveErrChan := make(chan error, 1)
	go func() {
		serveErrChan <- server.ListenAndServe()
	}()

	select {
	case sig := <-signalChan:
		logger.Info("main: shutting down", slog.String("signal", sig.String()))
	case err := <-serveErrChan:
		return fmt.Errorf("serve: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	err = <-serveErrChan
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
