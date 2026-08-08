package service

import "context"

// Request-scoped values that the transport layer attaches and the services
// record in the audit trail.

type correlationKey struct{}
type sourceIPKey struct{}

// WithCorrelation stores the request correlation id.
func WithCorrelation(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey{}, id)
}

// CorrelationFromContext reads the request correlation id.
func CorrelationFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(correlationKey{}).(string); ok {
		return v
	}
	return ""
}

// WithSourceIP stores the caller's address.
func WithSourceIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, sourceIPKey{}, ip)
}

// SourceIPFromContext reads the caller's address.
func SourceIPFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(sourceIPKey{}).(string); ok {
		return v
	}
	return ""
}
