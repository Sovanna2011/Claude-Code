// Command api is the Farm Area Monitoring service: a REST API over PostgreSQL with PostGIS,
// serving the SAPUI5 dashboard.
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

	"github.com/sovanna2011/sugarcane-go/backend/internal/auth"
	"github.com/sovanna2011/sugarcane-go/backend/internal/config"
	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/httpapi"
	"github.com/sovanna2011/sugarcane-go/backend/internal/repository"
	"github.com/sovanna2011/sugarcane-go/backend/internal/service"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	defer db.Close()

	if cfg.MigrateOnStart {
		if err := db.Migrate(ctx, cfg.MigrationsDir); err != nil {
			return err
		}
	}

	// The composition root: everything is constructed once, here, and injected downwards. No
	// package reaches for a global.
	support := repository.NewSupportRepository(db)
	services := service.New(db, service.Repositories{
		Master:      repository.NewMasterRepository(db),
		Dashboard:   repository.NewDashboardRepository(db),
		Support:     support,
		Activities:  repository.NewActivityRepository(db),
		Projections: repository.NewProjectionRepository(db),
		Plans:       repository.NewPlanRepository(db),
	})
	tokens := auth.NewTokens(cfg.JWTSecret, cfg.TokenTTL)
	api := httpapi.New(services, tokens, support, db, log)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Routes(cfg.AllowedOrigins),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		log.Info("listening", "addr", cfg.Addr, "origins", cfg.AllowedOrigins)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	// In-flight requests get a moment to finish; a write cut off mid-transaction would roll back,
	// but the caller would never learn which way it went.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
