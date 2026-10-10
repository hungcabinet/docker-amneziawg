package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
	"github.com/hungcabinet/docker-amneziawg/web/internal/status"
	"github.com/hungcabinet/docker-amneziawg/web/internal/store"
	"github.com/hungcabinet/docker-amneziawg/web/internal/webui"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("panel stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	ctx := context.Background()
	if err := st.Bootstrap(ctx, cfg.User, cfg.Password); err != nil {
		return err
	}

	provider, err := status.New(cfg)
	if err != nil {
		return err
	}

	app, err := webui.New(cfg, st, provider)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	idle, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("panel listening", "addr", cfg.Listen, "source", cfg.Source)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-idle.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
