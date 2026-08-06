package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Verifier turns a bearer token into a Principal.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (Principal, error)
	// Mode names the configured mechanism, reported by the health endpoint so
	// an operator can see at a glance whether a box is running in dev mode.
	Mode() string
}

// Config configures authentication from environment variables.
type Config struct {
	// Mode is "oidc" for an enterprise identity provider or "dev" for the
	// local development and demo profile.
	Mode string
	// Issuer is the OIDC issuer URL; its discovery document is read at start-up.
	Issuer string
	// Audience is the expected `aud` claim, normally the API's client id.
	Audience string
	// RolesClaim is the claim that carries the user's roles. Identity providers
	// differ, so the path is configurable ("realm_access.roles", "groups", ...).
	RolesClaim string
	// RoleMapping translates provider group names into application roles.
	RoleMapping map[string]string
	// DevSecret signs tokens in dev mode. It has no effect in oidc mode.
	DevSecret string
	// DevUsers are the accounts the dev login endpoint will issue tokens for.
	DevUsers map[string]DevUser
	// HTTPClient is used to fetch discovery and key material.
	HTTPClient *http.Client
}

// DevUser is a canned account for the development and demo profile.
type DevUser struct {
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Companies   []string `json:"companies"`
	Factories   []string `json:"factories"`
	// HomeFactory is the code of the factory this account belongs to. It is
	// used only when the sandbox holds more than one tenant: the start-up code
	// resolves it to a company and a factory id and scopes the account to those
	// alone, so that "a planner at one mill cannot read another mill's plan" is
	// a claim two real tenants can be held to. Empty means the account is
	// scoped to everything in the sandbox, which is the single-tenant default.
	HomeFactory string `json:"homeFactory,omitempty"`
}

// claims is the subset of the token this application reads.
type claims struct {
	jwt.RegisteredClaims
	Username  string   `json:"preferred_username,omitempty"`
	Name      string   `json:"name,omitempty"`
	Email     string   `json:"email,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	Companies []string `json:"companies,omitempty"`
	Factories []string `json:"factories,omitempty"`
	RawClaims map[string]any
}

// ---------------------------------------------------------------------------
// OIDC verifier
// ---------------------------------------------------------------------------

type oidcVerifier struct {
	cfg      Config
	jwksURI  string
	client   *http.Client
	mu       sync.RWMutex
	keys     map[string]*rsa.PublicKey
	fetched  time.Time
	cacheTTL time.Duration
}

// discovery is the part of the OIDC discovery document we need.
type discovery struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

// NewVerifier builds the verifier for the configured mode.
func NewVerifier(ctx context.Context, cfg Config) (Verifier, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	switch strings.ToLower(cfg.Mode) {
	case "dev":
		if cfg.DevSecret == "" {
			return nil, errors.New("dev authentication needs AUTH_DEV_SECRET")
		}
		return &devVerifier{cfg: cfg}, nil
	case "oidc", "":
		if cfg.Issuer == "" {
			return nil, errors.New("oidc authentication needs AUTH_ISSUER")
		}
		v := &oidcVerifier{cfg: cfg, client: cfg.HTTPClient, cacheTTL: 15 * time.Minute}
		if err := v.discover(ctx); err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unknown authentication mode %q", cfg.Mode)
	}
}

func (v *oidcVerifier) Mode() string { return "oidc" }

func (v *oidcVerifier) discover(ctx context.Context) error {
	url := strings.TrimSuffix(v.cfg.Issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("read openid configuration: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openid configuration returned %s", resp.Status)
	}
	var d discovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return fmt.Errorf("decode openid configuration: %w", err)
	}
	if d.JWKSURI == "" {
		return errors.New("openid configuration has no jwks_uri")
	}
	v.jwksURI = d.JWKSURI
	return nil
}

// jwk is one key from the provider's key set.
type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Use string `json:"use"`
}

func (v *oidcVerifier) keyFor(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Since(v.fetched) < v.cacheTTL
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}
	// A key we have not seen means the provider rotated: refetch once.
	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if key, ok := v.keys[kid]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("signing key %q is not in the provider key set", kid)
}

func (v *oidcVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURI, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("read jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned %s", resp.Status)
	}
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || (k.Use != "" && k.Use != "sig") {
			continue
		}
		pub, err := rsaKey(k)
		if err != nil {
			continue // a key we cannot parse is skipped, not fatal
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("the provider key set contains no usable RSA signing keys")
	}
	v.mu.Lock()
	v.keys, v.fetched = keys, time.Now()
	v.mu.Unlock()
	return nil
}

func rsaKey(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() || e.Int64() > 1<<31 {
		return nil, errors.New("unreasonable RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(e.Int64())}, nil
}

func (v *oidcVerifier) Verify(ctx context.Context, raw string) (Principal, error) {
	var c claims
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		jwt.WithIssuer(v.cfg.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	token, err := parser.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("the token has no key id")
		}
		return v.keyFor(ctx, kid)
	})
	if err != nil {
		return Principal{}, fmt.Errorf("token rejected: %w", err)
	}
	if !token.Valid {
		return Principal{}, errors.New("token rejected")
	}
	if v.cfg.Audience != "" && !c.VerifyAudience(v.cfg.Audience) {
		return Principal{}, errors.New("token rejected: wrong audience")
	}
	return v.cfg.principalFrom(&c, raw)
}

// VerifyAudience reports whether the expected audience is present.
func (c *claims) VerifyAudience(expected string) bool {
	for _, a := range c.Audience {
		if a == expected {
			return true
		}
	}
	return false
}

// principalFrom maps token claims onto the application's principal, applying
// the configured roles claim path and the provider group to role mapping.
func (cfg Config) principalFrom(c *claims, raw string) (Principal, error) {
	roles := c.Roles
	if cfg.RolesClaim != "" {
		if extracted := extractRoles(raw, cfg.RolesClaim); len(extracted) > 0 {
			roles = extracted
		}
	}
	mapped := make([]string, 0, len(roles))
	for _, r := range roles {
		if to, ok := cfg.RoleMapping[r]; ok {
			mapped = append(mapped, to)
			continue
		}
		if _, known := DefaultRoles[r]; known {
			mapped = append(mapped, r)
		}
	}
	username := c.Username
	if username == "" {
		username = c.Subject
	}
	if c.Subject == "" {
		return Principal{}, errors.New("token rejected: no subject claim")
	}
	return NewPrincipal(c.Subject, username, c.Name, c.Email, mapped, c.Companies, c.Factories), nil
}

// extractRoles reads a dotted claim path such as "realm_access.roles" from the
// raw token payload, because identity providers nest roles differently.
func extractRoles(raw, path string) []string {
	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil
	}
	var current any = doc
	for _, segment := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = m[segment]
		if !ok {
			return nil
		}
	}
	list, ok := current.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Development verifier
// ---------------------------------------------------------------------------

// devVerifier issues and verifies HS256 tokens locally. It exists so the
// application can be run and demonstrated without an identity provider, and it
// refuses to start without an explicit secret. It is never enabled in
// production: config.Validate rejects mode=dev when APP_ENV is production.
type devVerifier struct{ cfg Config }

func (d *devVerifier) Mode() string { return "dev" }

func (d *devVerifier) Verify(_ context.Context, raw string) (Principal, error) {
	var c claims
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	token, err := parser.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) {
		return []byte(d.cfg.DevSecret), nil
	})
	if err != nil || !token.Valid {
		return Principal{}, fmt.Errorf("token rejected: %w", err)
	}
	if c.Subject == "" {
		return Principal{}, errors.New("token rejected: no subject claim")
	}
	return NewPrincipal(c.Subject, c.Username, c.Name, c.Email, c.Roles, c.Companies, c.Factories), nil
}

// IssueDevToken mints a token for one of the configured development accounts.
func IssueDevToken(cfg Config, username string, ttl time.Duration) (string, Principal, error) {
	user, ok := cfg.DevUsers[username]
	if !ok {
		return "", Principal{}, fmt.Errorf("unknown development user %q", username)
	}
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "dev|" + username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "sugarplan-dev",
		},
		Username:  username,
		Name:      user.DisplayName,
		Email:     user.Email,
		Roles:     user.Roles,
		Companies: user.Companies,
		Factories: user.Factories,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(cfg.DevSecret))
	if err != nil {
		return "", Principal{}, fmt.Errorf("sign development token: %w", err)
	}
	p := NewPrincipal(c.Subject, username, user.DisplayName, user.Email,
		user.Roles, user.Companies, user.Factories)
	return signed, p, nil
}
