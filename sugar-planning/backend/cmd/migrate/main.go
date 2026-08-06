// Command migrate applies or reverses database migrations.
//
//	migrate up                 apply every pending migration
//	migrate down -steps 1      reverse the most recent migration
//	migrate status             list applied and pending migrations
//	migrate seed               load the demonstration scenario (refused in production)
//
// The migrations are embedded in the binary, so the image that runs the API is
// the image that migrates the database and the two cannot drift apart.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kss/sugarplan/internal/config"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// The command comes first and its flags follow it, which is what a reader
	// expects from `migrate down -steps 2`. The standard flag package stops at
	// the first non-flag argument, so the command is taken off the front before
	// the flags are parsed; otherwise `-steps` after the command is silently
	// ignored and only one migration is reversed.
	if len(os.Args) < 2 {
		return fmt.Errorf("no command given; use up, down, status or seed")
	}
	command := os.Args[1]

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	steps := fs.Int("steps", 1, "number of migrations to reverse for the down command")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}
	if *steps < 1 {
		return fmt.Errorf("-steps must be at least 1, got %d", *steps)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Store != "postgres" {
		return fmt.Errorf("migrations need STORE=postgres, got %q", cfg.Store)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	st, err := postgres.Open(ctx, postgres.Config{
		DSN: cfg.DatabaseURL, MaxConns: 4, ConnectTimeout: cfg.DBConnectTimeout,
	})
	if err != nil {
		return err
	}
	defer st.Close()

	switch command {
	case "up":
		applied, err := st.MigrateUp(ctx)
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			fmt.Println("the database is already up to date")
			return nil
		}
		for _, m := range applied {
			fmt.Println("applied", m)
		}
		return nil

	case "down":
		if cfg.IsProduction() {
			return fmt.Errorf(
				"reversing a migration in production is a deliberate operation; " +
					"run it with APP_ENV set to something else after taking a backup")
		}
		reverted, err := st.MigrateDown(ctx, *steps)
		if err != nil {
			return err
		}
		for _, m := range reverted {
			fmt.Println("reverted", m)
		}
		return nil

	case "status":
		all, err := postgres.LoadMigrations()
		if err != nil {
			return err
		}
		applied, err := st.Applied(ctx)
		if err != nil {
			return err
		}
		done := map[string]bool{}
		for _, v := range applied {
			done[v] = true
		}
		for _, m := range all {
			state := "pending"
			if done[m.Version] {
				state = "applied"
			}
			fmt.Printf("%-6s %s_%s\n", state, m.Version, m.Name)
		}
		return nil

	case "seed":
		if cfg.IsProduction() {
			return fmt.Errorf("demonstration data must never be loaded into production")
		}
		planning := service.NewPlanning(st, func() time.Time { return time.Now().UTC() })
		res, err := seed.LoadWithActuals(ctx, st, planning, cfg.SeedActualDays)
		if err != nil {
			return err
		}
		fmt.Printf("seeded season %s, version %s: %s t of cane over %d days\n",
			res.SeasonID, res.BudgetID,
			res.Generated.Summary.CaneAllocated, res.Generated.Summary.WorkingDays)
		return nil

	default:
		return fmt.Errorf("unknown command %q; use up, down, status or seed", command)
	}
}
