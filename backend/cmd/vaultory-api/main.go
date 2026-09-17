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

	// One condition, two refusals, and no third path — and deliberately before anything that
	// touches the network. A security check that only runs after a database connection
	// succeeds is a security check nobody sees fail: `docker run` on the production image
	// would report a missing database URL and never mention authentication at all.
	//
	// Authentication is out of scope for feature 001, so the development resolver is still the
	// only one there is. Refusing to start without a resolver is deliberate: a server that
	// silently resolved every request to one collector would look like it worked and would have
	// no privacy at all.
	//
	// The second refusal is what makes a production image possible. In a production build
	// (-tags production) the development resolver is not compiled in, so its constructor fails
	// and a binary asked to use it stops here instead of starting without authentication
	// (FR-019). The decision lives in the program rather than in an entrypoint script, because a
	// security-relevant condition expressed in shell is a security-relevant condition nobody
	// reviews.
	if !cfg.DevIdentity {
		return errors.New(
			"no collector resolver is configured: set VAULTORY_DEV_IDENTITY=enabled for local " +
				"development, or supply a real authentication resolver before deploying")
	}
	dev, err := identity.NewDevResolver(cfg.SessionSecret)
	if err != nil {
		return err
	}
	slog.Warn("development identity is enabled: sessions are minted without authentication, " +
		"never run this outside local development")

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if cfg.ImageStore != "filesystem" {
		return fmt.Errorf("image store %q is not implemented", cfg.ImageStore)
	}
	images, err := imagestore.NewFilesystem(cfg.ImageStorePath)
	if err != nil {
		return err
	}

	service := collection.NewService(postgres.NewStore(pool), images, cfg.IdempotencyWindow)
	server := httpapi.NewServer(service, dev, dev)

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
