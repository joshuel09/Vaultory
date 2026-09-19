// Command vaultory-api serves Vaultory's REST API.
//
// This is the only place the pieces are wired together: configuration, the database pool, the
// image store, the identity seam, the application services, and the HTTP transport. Everything
// below is composed here and nowhere else.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/config"
	"github.com/joshuel09/vaultory/backend/internal/identity"
	"github.com/joshuel09/vaultory/backend/internal/imagestore"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
	"github.com/joshuel09/vaultory/backend/internal/transport/httpapi"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := run(); err != nil {
		slog.Error("vaultory-api failed to start", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Identity is established by verifying a Better Auth session: the cookie's signature is
	// recomputed with the shared secret, and the session row is confirmed live in PostgreSQL.
	// Neither step trusts the presentation layer, which is what FR-013 requires and what the
	// constitution means by "client-provided user IDs MUST NOT be accepted without verification".
	//
	// There is no longer a development resolver to refuse. config.Load has already refused to
	// start without a session secret of usable length (FR-019, FR-019a), which is the condition
	// that can actually occur now — the old "no resolver configured" check guarded one that
	// cannot.
	//
	// This sits after the pool because verification is a database lookup. The fail-fast property
	// the old ordering protected is unaffected: an unusable secret is rejected by config.Load,
	// before anything touches the network.
	resolver := identity.NewSessionResolver(pool, cfg.SessionSecret)

	if cfg.ImageStore != "filesystem" {
		return fmt.Errorf("image store %q is not implemented", cfg.ImageStore)
	}
	images, err := imagestore.NewFilesystem(cfg.ImageStorePath)
	if err != nil {
		return err
	}

	service := collection.NewService(postgres.NewStore(pool), images, cfg.IdempotencyWindow)
	server := httpapi.NewServer(service, resolver)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		// Generous enough for a 10 MB upload over a slow connection.
		ReadTimeout:  2 * time.Minute,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("vaultory-api listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
