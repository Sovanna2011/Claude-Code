// Command server runs the sugar production planning API and, optionally, the
// SAPUI5 application from the same process.
//
//	# demo: no database, reference scenario loaded, development login enabled
//	STORE=memory AUTH_MODE=dev AUTH_DEV_SECRET=local-development-secret \
//	SEED_DEMO=true HTTP_STATIC_DIR=../frontend/webapp go run ./cmd/server
//
//	# production: PostgreSQL and the enterprise identity provider
//	APP_ENV=production DATABASE_URL=... AUTH_ISSUER=... go run ./cmd/server
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

	"github.com/kss/sugarplan/internal/api"
	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/config"
	"github.com/kss/sugarplan/internal/integration"
	"github.com/kss/sugarplan/internal/jobs"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/store/memory"
	"github.com/kss/sugarplan/internal/store/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- storage ------------------------------------------------------------
	st, err := openStore(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := st.Close(); err != nil {
			logger.Error("closing the store", "error", err)
		}
	}()

	// --- services -----------------------------------------------------------
	planning := service.NewPlanning(st, func() time.Time { return time.Now().UTC() })
	analytics := service.NewAnalytics(st, planning)
	materials := service.NewMaterials(st, planning)
	execution := service.NewExecution(st, func() time.Time { return time.Now().UTC() })
	costing := service.NewCosting(st, planning, func() time.Time { return time.Now().UTC() })

	// The publisher decides where integration events go. With no endpoint
	// configured they go to the application log, which is a real destination
	// rather than a pretend one: the events appear where the operator already
	// looks, and pointing INTEGRATION_ENDPOINT at the ERP is the only change
	// needed to start feeding it.
	var publisher service.Publisher = integration.NewLog(logger, cfg.IntegrationSource)
	if webhook := integration.NewHTTP(integration.HTTPOptions{
		Endpoint:   cfg.IntegrationEndpoint,
		AuthHeader: cfg.IntegrationAuthHeader,
		AuthValue:  cfg.IntegrationAuthValue,
		Timeout:    cfg.IntegrationTimeout,
		Source:     cfg.IntegrationSource,
	}); webhook != nil {
		publisher = webhook
	}
	interfaces := service.NewIntegration(st, execution, planning, publisher,
		func() time.Time { return time.Now().UTC() })
	imports := service.NewImports(st, planning, func() time.Time { return time.Now().UTC() })
	notifications := service.NewNotifications(st, analytics,
		func() time.Time { return time.Now().UTC() })
	logger.Info("integration events will be published", "to", publisher.Describe())

	if cfg.SeedDemo {
		logger.Info("loading the demonstration scenario")
		res, err := seed.LoadWithActuals(ctx, st, planning, cfg.SeedActualDays)
		if err != nil {
			return fmt.Errorf("seed demonstration data: %w", err)
		}
		logger.Info("demonstration scenario ready",
			"season", res.SeasonID, "version", res.BudgetID,
			"caneTons", res.Generated.Summary.CaneAllocated.String(),
			"days", res.Generated.Summary.WorkingDays)
	}

	// --- authentication -----------------------------------------------------
	if cfg.Auth.Mode == "dev" {
		// The data scope fails closed: a principal with no companies and no
		// factories sees nothing. The demonstration accounts are therefore
		// scoped to whatever is in this sandbox, which is the seeded factory.
		// In oidc mode the scope comes from the token and no such step exists.
		if err := scopeDevUsers(ctx, st, cfg.Auth.DevUsers); err != nil {
			return fmt.Errorf("scope development users: %w", err)
		}
	}
	verifier, err := auth.NewVerifier(ctx, cfg.Auth)
	if err != nil {
		return fmt.Errorf("configure authentication: %w", err)
	}
	if verifier.Mode() == "dev" {
		logger.Warn("development authentication is enabled; this must never be used in production")
	}

	// --- HTTP ---------------------------------------------------------------
	// One registry, shared between the request path and the scheduler, so a
	// single scrape carries both the traffic and the background work.
	metrics := api.NewMetrics(nil)
	handler := api.NewServer(api.Options{
		Store: st, Planning: planning, Analytics: analytics, Materials: materials,
		Execution: execution, Costing: costing, Integration: interfaces,
		Imports: imports, Notifications: notifications,
		Metrics:  metrics,
		Verifier: verifier, AuthCfg: cfg.Auth, Logger: logger, Version: cfg.Version,
		StaticDir: cfg.StaticDir, AllowedOrigins: cfg.AllowedOrigins,
		RequestTimeout: cfg.RequestTimeout, RateLimit: cfg.RateLimit, RateInterval: cfg.RateInterval,
	})

	// --- background jobs ----------------------------------------------------
	scheduler, err := startJobs(ctx, cfg, st, interfaces, notifications, logger)
	if err != nil {
		return err
	}
	if scheduler != nil {
		scheduler.Observe(metrics.RecordJob)
		defer scheduler.Stop()
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute, // a season-wide PDF export takes a moment
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening",
			"addr", cfg.HTTPAddr, "env", cfg.Env, "store", cfg.Store,
			"auth", verifier.Mode(), "version", cfg.Version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining requests",
			"timeout", cfg.ShutdownTimeout.String())
	}

	// Graceful shutdown: stop accepting, let in-flight requests finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	logger.Info("stopped cleanly")
	return nil
}

// startJobs starts the recurring work, or reports that it is switched off.
//
// The dispatcher is the only job that must run for the system to behave as
// documented: an event written to the outbox is delivered by it. Setting the
// interval to zero turns it off, which is a legitimate choice for a site that
// drives the dispatcher from its own scheduler, but it is said out loud in the
// log rather than left to be discovered.
func startJobs(ctx context.Context, cfg config.Config, st store.Store,
	interfaces *service.Integration, notifications *service.Notifications,
	logger *slog.Logger,
) (*jobs.Scheduler, error) {

	// The lease owner names this instance, so an operations screen can say
	// which one last ran a job. A hostname is what an operator recognises.
	owner, err := os.Hostname()
	if err != nil || owner == "" {
		owner = "instance"
	}

	// Each job runs as the system rather than as a person, and each is given a
	// principal that may do exactly its own work and nothing else. Running the
	// scheduler as an administrator would be one more way in.
	asSystem := func(ctx context.Context, roles ...string) context.Context {
		return auth.WithPrincipal(ctx, auth.NewSystemPrincipal(roles...))
	}

	var registered []jobs.Job

	if cfg.DispatchInterval > 0 {
		registered = append(registered, jobs.Job{
			Name:    "outbox-dispatch",
			Every:   cfg.DispatchInterval,
			Timeout: 2 * time.Minute,
			Run: func(ctx context.Context) (string, error) {
				ctx = asSystem(ctx, auth.RoleIntegration)
				result, err := interfaces.Dispatch(ctx, cfg.DispatchBatch)
				if err != nil {
					return "", err
				}
				if result.Considered == 0 {
					return "nothing was due", nil
				}
				detail := fmt.Sprintf("published %d of %d", result.Published, result.Considered)
				if result.Failed > 0 {
					detail += fmt.Sprintf(", %d failed", result.Failed)
				}
				if result.Exhausted > 0 {
					detail += fmt.Sprintf(", %d gave up and need somebody", result.Exhausted)
				}
				return detail, nil
			},
		})
	} else {
		logger.Warn("the outbox dispatcher is switched off; events will wait until somebody sends them by hand")
	}

	if cfg.AlertInterval > 0 {
		registered = append(registered, jobs.Job{
			Name:    "alert-evaluation",
			Every:   cfg.AlertInterval,
			Timeout: 5 * time.Minute,
			Run: func(ctx context.Context) (string, error) {
				// The evaluator reads plans and writes nothing but
				// notifications, so a read-only principal is all it needs.
				ctx = asSystem(ctx, auth.RoleExecutiveViewer)
				result, err := notifications.Evaluate(ctx)
				if err != nil {
					return "", err
				}
				if result.Raised == 0 {
					return fmt.Sprintf("%d alerts across %d seasons, nothing new",
						result.Alerts, result.Seasons), nil
				}
				return fmt.Sprintf("raised %d of %d alerts across %d seasons (%d already waiting)",
					result.Raised, result.Alerts, result.Seasons, result.Suppressed), nil
			},
		})
	} else {
		logger.Warn("alert evaluation is switched off; alerts will appear on the dashboard but reach nobody's inbox")
	}

	if len(registered) == 0 {
		return nil, nil
	}

	scheduler, err := jobs.New(st, owner, logger,
		func() time.Time { return time.Now().UTC() }, registered...)
	if err != nil {
		return nil, fmt.Errorf("background jobs: %w", err)
	}
	scheduler.Start(ctx)
	return scheduler, nil
}

// scopeDevUsers gives every demonstration account access to the companies and
// factories that exist in this sandbox. It runs only in dev mode; production
// scopes come from the identity provider's token claims.
func scopeDevUsers(ctx context.Context, st store.Store, users map[string]auth.DevUser) error {
	companies, err := st.MasterData().Companies().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return err
	}
	factories, err := st.MasterData().Factories().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return err
	}
	companyIDs := make([]string, 0, len(companies.Items))
	for _, c := range companies.Items {
		companyIDs = append(companyIDs, c.ID)
	}
	factoryIDs := make([]string, 0, len(factories.Items))
	for _, f := range factories.Items {
		factoryIDs = append(factoryIDs, f.ID)
	}
	for name, u := range users {
		u.Companies, u.Factories = companyIDs, factoryIDs
		users[name] = u
	}
	return nil
}

// openStore builds the configured store, running migrations first when asked.
func openStore(ctx context.Context, cfg config.Config, logger *slog.Logger) (store.Store, error) {
	if cfg.Store == "memory" {
		logger.Warn("using the in-memory store; nothing is persisted across a restart")
		return memory.New(), nil
	}

	pg, err := postgres.Open(ctx, postgres.Config{
		DSN: cfg.DatabaseURL, MaxConns: cfg.DBMaxConns, MinConns: cfg.DBMinConns,
		MaxConnLifetime: cfg.DBConnLifetime, ConnectTimeout: cfg.DBConnectTimeout,
	})
	if err != nil {
		return nil, err
	}
	if cfg.MigrateOnStart {
		applied, err := pg.MigrateUp(ctx)
		if err != nil {
			_ = pg.Close()
			return nil, fmt.Errorf("migrate database: %w", err)
		}
		if len(applied) > 0 {
			logger.Info("database migrated", "applied", applied)
		}
	}
	return pg, nil
}

func newLogger(cfg config.Config) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	if cfg.LogFormat == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
