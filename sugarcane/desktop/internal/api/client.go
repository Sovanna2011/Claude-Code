package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FieldError names one field the service refused, and why.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error is a refusal from the service, carried whole. The service answers every error with a code,
// a message written for a person and, where the fault is in one field, which field — so the window
// shows the service's own words rather than inventing its own, and the code can be branched on
// without matching on prose.
type Error struct {
	Status  int
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields"`
}

func (e *Error) Error() string {
	if len(e.Fields) == 0 {
		return e.Message
	}
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Message)
	}
	return e.Message + " " + strings.Join(parts, " ")
}

// Unauthorized reports whether the token is missing or has expired, so the caller signs in again.
func (e *Error) Unauthorized() bool { return e.Status == http.StatusUnauthorized }

// IsUnauthorized answers the same question for an error of unknown type.
func IsUnauthorized(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.Unauthorized()
}

// Client talks to the Farm Area service.
//
// The token is held in memory only. Nothing is written to disk: a desktop application that cached a
// bearer token in the user's profile would keep granting access after they had signed out.
type Client struct {
	baseURL string
	http    *http.Client

	mu        sync.RWMutex
	token     string
	user      User
	expiresAt time.Time
}

// New returns a client for a service at baseURL, for example http://127.0.0.1:8080.
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// BaseURL is the service this client is pointed at, which the window shows in its status bar.
func (c *Client) BaseURL() string { return c.baseURL }

// SignedIn reports whether a token has been obtained.
func (c *Client) SignedIn() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token != ""
}

// User is who is signed in; the zero value before a successful sign-in.
func (c *Client) User() User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.user
}

// ExpiresAt is when the token stops being accepted.
func (c *Client) ExpiresAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.expiresAt
}

// SignIn exchanges a name and a password for a token.
func (c *Client) SignIn(ctx context.Context, userName, password string) (User, error) {
	body, err := json.Marshal(map[string]string{"userName": userName, "password": password})
	if err != nil {
		return User{}, err
	}

	// The login is the one call that carries no token, so it goes out without the header rather
	// than with a stale one from a previous session.
	c.SignOut()

	var out loginResponse
	if err := c.do(ctx, http.MethodPost, "/api/auth/login", body, &out); err != nil {
		return User{}, err
	}

	c.mu.Lock()
	c.token, c.user, c.expiresAt = out.Token, out.User, out.ExpiresAt
	c.mu.Unlock()
	return out.User, nil
}

// SignOut forgets the token. The service is stateless, so there is nothing to tell it.
func (c *Client) SignOut() {
	c.mu.Lock()
	c.token, c.user, c.expiresAt = "", User{}, time.Time{}
	c.mu.Unlock()
}

// WhoAmI is the service's own view of who is calling, which is how a held token is checked.
func (c *Client) WhoAmI(ctx context.Context) (User, error) {
	var out User
	err := c.get(ctx, "/api/auth/me", &out)
	return out, err
}

// Dashboard returns the KPI cards for the filtered land.
func (c *Client) Dashboard(ctx context.Context, f Filter) (Dashboard, error) {
	var out Dashboard
	err := c.get(ctx, "/api/dashboard/farm-area"+f.Query(), &out)
	return out, err
}

// Tree returns the farm → zone → block hierarchy for the filtered land.
func (c *Client) Tree(ctx context.Context, f Filter) ([]*TreeNode, error) {
	var out []*TreeNode
	err := c.get(ctx, "/api/reports/farm-area-tree"+f.Query(), &out)
	return out, err
}

// Map returns the filtered land as GeoJSON, drawn at one location level.
func (c *Client) Map(ctx context.Context, f Filter, level string) (MapData, error) {
	var out MapData
	err := c.get(ctx, "/api/dashboard/farm-area/map"+f.Query(Param{"level", level}), &out)
	return out, err
}

// Lookup returns one filter dropdown's entries. parentID narrows a child list to its parent —
// the zones of one farm, the blocks of one zone — and is ignored when nil.
func (c *Client) Lookup(ctx context.Context, kind string, parentID *int) ([]LookupItem, error) {
	path := "/api/lookups/" + kind
	if parentID != nil {
		path += "?parentId=" + strconv.Itoa(*parentID)
	}
	var out []LookupItem
	err := c.get(ctx, path, &out)
	return out, err
}

// Projections returns one page of the planting-projection list.
func (c *Client) Projections(ctx context.Context, f Filter, page, pageSize int) (Page[Projection], error) {
	var out Page[Projection]
	err := c.get(ctx, "/api/projections"+f.Query(
		Param{"page", strconv.Itoa(page)},
		Param{"pageSize", strconv.Itoa(pageSize)},
	), &out)
	return out, err
}

// ---------------------------------------------------------------- plumbing

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		// A refused connection is the common case in the field — the service is not running, or
		// the address in the sign-in box is wrong — and it deserves to say so plainly.
		return &Error{Status: 0, Code: "UNREACHABLE",
			Message: "The service at " + c.baseURL + " could not be reached: " + err.Error()}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return &Error{Status: resp.StatusCode, Code: "UNREADABLE_RESPONSE",
			Message: "The service's answer could not be read: " + err.Error()}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return toError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return &Error{Status: resp.StatusCode, Code: "EMPTY_RESPONSE",
			Message: "The service answered with an empty body where a result was expected."}
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return &Error{Status: resp.StatusCode, Code: "UNREADABLE_RESPONSE",
			Message: "The service's answer could not be read: " + err.Error()}
	}
	return nil
}

// toError turns a refusal into the service's own error where it sent one, and into something that
// says what actually arrived where it did not — a proxy's HTML page, say. "The request failed"
// sends whoever reads it hunting.
func toError(status int, raw []byte) error {
	var apiErr Error
	if err := json.Unmarshal(raw, &apiErr); err == nil && apiErr.Code != "" {
		apiErr.Status = status
		return &apiErr
	}

	detail := strings.TrimSpace(string(raw))
	if len(detail) > 300 {
		detail = detail[:300] + "…"
	}
	message := fmt.Sprintf("The service answered %d %s.", status, http.StatusText(status))
	if detail != "" {
		message += " " + detail
	}
	return &Error{Status: status, Code: "HTTP_" + strconv.Itoa(status), Message: message}
}
