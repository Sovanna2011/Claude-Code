// Package config reads the environment into a validated configuration.
//
// Everything the application needs to run differently in development, test,
// UAT and production comes from environment variables. Nothing is read from a
// file baked into the image, and no secret has a default.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/auth"
)

// Config is the whole runtime configuration.
type Config struct {
	Env       string // development, test, uat, production
	LogLevel  string
	LogFormat string // json or text

	HTTPAddr        string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
	RateLimit       int
	RateInterval    time.Duration
	StaticDir       string

	// Store is "postgres" or "memory". The memory store backs the demo profile
	// and needs no database at all.
	Store            string
	DatabaseURL      string
	DBMaxConns       int32
	DBMinConns       int32
	DBConnLifetime   time.Duration
	DBConnectTimeout time.Duration
	MigrateOnStart   bool

	Auth auth.Config

	// SeedDemo loads the reference scenario at start-up. It is refused in
	// production, because demonstration data must never appear in a real plant.
	SeedDemo       bool
	SeedActualDays int

	// Integration configures the outbound interface. With no endpoint set the
	// dispatcher publishes to the application log, which is a real destination
	// rather than a pretend one - the events appear where the operator already
	// looks - and pointing INTEGRATION_ENDPOINT at the ERP is the only change
	// needed to start feeding it.
	IntegrationEndpoint   string
	IntegrationAuthHeader string
	IntegrationAuthValue  string
	IntegrationTimeout    time.Duration
	IntegrationSource     string
	// DispatchInterval is how often the background dispatcher runs. Zero turns
	// it off, which leaves the outbox to be drained by hand.
	DispatchInterval time.Duration
	DispatchBatch    int
	// AlertInterval is how often the alerts are evaluated into people's
	// inboxes. Zero turns it off, which leaves the alerts visible on the
	// dashboard and reaching nobody - said out loud in the log rather than
	// left to be discovered.
	AlertInterval time.Duration

	FactoryTimeZone string
	Version         string
}

// Load reads the environment and validates it.
func Load() (Config, error) {
	c := Config{
		Env:       env("APP_ENV", "development"),
		LogLevel:  env("LOG_LEVEL", "info"),
		LogFormat: env("LOG_FORMAT", "json"),

		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		RequestTimeout:  envDuration("HTTP_REQUEST_TIMEOUT", 30*time.Second),
		ShutdownTimeout: envDuration("HTTP_SHUTDOWN_TIMEOUT", 20*time.Second),
		AllowedOrigins:  envList("HTTP_ALLOWED_ORIGINS"),
		RateLimit:       envInt("HTTP_RATE_LIMIT", 600),
		RateInterval:    envDuration("HTTP_RATE_INTERVAL", time.Minute),
		StaticDir:       env("HTTP_STATIC_DIR", ""),

		Store:            strings.ToLower(env("STORE", "postgres")),
		DatabaseURL:      env("DATABASE_URL", ""),
		DBMaxConns:       int32(envInt("DB_MAX_CONNS", 20)),
		DBMinConns:       int32(envInt("DB_MIN_CONNS", 2)),
		DBConnLifetime:   envDuration("DB_CONN_LIFETIME", time.Hour),
		DBConnectTimeout: envDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
		MigrateOnStart:   envBool("DB_MIGRATE_ON_START", false),

		SeedDemo:       envBool("SEED_DEMO", false),
		SeedActualDays: envInt("SEED_ACTUAL_DAYS", 14),

		IntegrationEndpoint:   env("INTEGRATION_ENDPOINT", ""),
		IntegrationAuthHeader: env("INTEGRATION_AUTH_HEADER", ""),
		IntegrationAuthValue:  env("INTEGRATION_AUTH_VALUE", ""),
		IntegrationTimeout:    envDuration("INTEGRATION_TIMEOUT", 15*time.Second),
		IntegrationSource:     env("INTEGRATION_SOURCE", "sugarplan"),
		DispatchInterval:      envDuration("INTEGRATION_DISPATCH_INTERVAL", 30*time.Second),
		DispatchBatch:         envInt("INTEGRATION_DISPATCH_BATCH", 50),
		AlertInterval:         envDuration("ALERT_INTERVAL", 15*time.Minute),

		FactoryTimeZone: env("FACTORY_TIMEZONE", "Asia/Phnom_Penh"),
		Version:         env("APP_VERSION", "dev"),
	}

	c.Auth = auth.Config{
		Mode:        strings.ToLower(env("AUTH_MODE", "oidc")),
		Issuer:      env("AUTH_ISSUER", ""),
		Audience:    env("AUTH_AUDIENCE", ""),
		RolesClaim:  env("AUTH_ROLES_CLAIM", "roles"),
		RoleMapping: envMap("AUTH_ROLE_MAPPING"),
		DevSecret:   env("AUTH_DEV_SECRET", ""),
	}
	if c.Auth.Mode == "dev" {
		c.Auth.DevUsers = defaultDevUsers()
	}

	return c, c.Validate()
}

// Validate rejects a configuration that would be unsafe or that cannot work.
func (c Config) Validate() error {
	var problems []string

	if c.Store != "postgres" && c.Store != "memory" {
		problems = append(problems, fmt.Sprintf("STORE must be postgres or memory, got %q", c.Store))
	}
	if c.Store == "postgres" && c.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required when STORE=postgres")
	}
	if c.IntegrationEndpoint != "" {
		u, err := url.Parse(c.IntegrationEndpoint)
		switch {
		case err != nil:
			problems = append(problems, fmt.Sprintf("INTEGRATION_ENDPOINT is not a URL: %v", err))
		case u.Scheme != "http" && u.Scheme != "https":
			problems = append(problems, fmt.Sprintf(
				"INTEGRATION_ENDPOINT must be an http or https URL, got %q", u.Scheme))
		case u.Host == "":
			problems = append(problems, "INTEGRATION_ENDPOINT names no host")
		}
	}
	if (c.IntegrationAuthHeader == "") != (c.IntegrationAuthValue == "") {
		problems = append(problems,
			"INTEGRATION_AUTH_HEADER and INTEGRATION_AUTH_VALUE are set together or not at all")
	}
	if _, err := time.LoadLocation(c.FactoryTimeZone); err != nil {
		problems = append(problems, fmt.Sprintf("FACTORY_TIMEZONE %q is not a known IANA time zone", c.FactoryTimeZone))
	}

	switch c.Auth.Mode {
	case "dev":
		if c.Auth.DevSecret == "" {
			problems = append(problems, "AUTH_DEV_SECRET is required when AUTH_MODE=dev")
		}
		if len(c.Auth.DevSecret) < 16 {
			problems = append(problems, "AUTH_DEV_SECRET must be at least 16 characters")
		}
	case "oidc":
		if c.Auth.Issuer == "" {
			problems = append(problems, "AUTH_ISSUER is required when AUTH_MODE=oidc")
		}
	default:
		problems = append(problems, fmt.Sprintf("AUTH_MODE must be oidc or dev, got %q", c.Auth.Mode))
	}

	// The guards that matter: a production deployment must not run with the
	// development login, the in-memory store or demonstration data.
	if c.IsProduction() {
		if c.Auth.Mode != "oidc" {
			problems = append(problems, "AUTH_MODE must be oidc when APP_ENV=production")
		}
		if c.Store != "postgres" {
			problems = append(problems, "STORE must be postgres when APP_ENV=production")
		}
		if c.SeedDemo {
			problems = append(problems, "SEED_DEMO must be off when APP_ENV=production")
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("configuration is not usable:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// IsProduction reports whether this is a production deployment.
func (c Config) IsProduction() bool { return strings.EqualFold(c.Env, "production") }

// defaultDevUsers are the demonstration accounts, one per interesting role, so
// the role model can be shown without an identity provider.
func defaultDevUsers() map[string]auth.DevUser {
	return map[string]auth.DevUser{
		"planner": {DisplayName: "Sokha Planner", Email: "planner@example.com",
			Roles: []string{auth.RoleProductionPlanner}},
		"approver": {DisplayName: "Dara Factory Manager", Email: "approver@example.com",
			Roles: []string{auth.RoleApprover}},
		"supervisor": {DisplayName: "Vanna Shift Supervisor", Email: "supervisor@example.com",
			Roles: []string{auth.RoleShiftSupervisor}},
		"weighbridge": {DisplayName: "Rithy Weighbridge", Email: "weighbridge@example.com",
			Roles: []string{auth.RoleCaneOperator}},
		"warehouse": {DisplayName: "Chanthou Warehouse", Email: "warehouse@example.com",
			Roles: []string{auth.RoleWarehouseOperator}},
		"shipping": {DisplayName: "Sophea Shipment Planner", Email: "shipping@example.com",
			Roles: []string{auth.RoleShipmentPlanner}},
		"quality": {DisplayName: "Nary Laboratory", Email: "quality@example.com",
			Roles: []string{auth.RoleQualityUser}},
		"controller": {DisplayName: "Mealea Cost Controller", Email: "controller@example.com",
			Roles: []string{auth.RoleCostController}},
		"engineer": {DisplayName: "Piseth Maintenance", Email: "engineer@example.com",
			Roles: []string{auth.RoleMaintenanceUser}},
		"executive": {DisplayName: "Bopha Executive", Email: "executive@example.com",
			Roles: []string{auth.RoleExecutiveViewer}},
		"auditor": {DisplayName: "Sovann Auditor", Email: "auditor@example.com",
			Roles: []string{auth.RoleAuditor}},
		"admin": {DisplayName: "System Administrator", Email: "admin@example.com",
			Roles: []string{auth.RoleSystemAdmin, auth.RoleMasterDataAdmin}},
		// The machine account the weighbridge terminal and the laboratory
		// system sign in as. It is offered on the demonstration sign-in so the
		// two inbound interfaces can be tried, and it can do nothing else.
		"interface": {DisplayName: "Gate and laboratory interface", Email: "interface@example.com",
			Roles: []string{auth.RoleIntegration}},
	}
}

// DevUsernames lists the demonstration accounts, for the login screen.
func DevUsernames(users map[string]auth.DevUser) []string {
	out := make([]string, 0, len(users))
	for name := range users {
		out = append(out, name)
	}
	return out
}

// ---------------------------------------------------------------------------
// Environment helpers
// ---------------------------------------------------------------------------

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envList(key string) []string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// envMap parses "provider-group=APP_ROLE,other-group=OTHER_ROLE".
func envMap(key string) map[string]string {
	out := map[string]string{}
	for _, pair := range envList(key) {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

// ErrNoConfig is returned when the environment is empty enough that the caller
// almost certainly forgot to source it.
var ErrNoConfig = errors.New("no configuration found in the environment")
