// Package identity resolves which collector a request is acting as.
//
// This is the single seam through which the acting collector is determined. Handlers obtain the
// collector here and nowhere else, and no collector identifier is ever accepted from a request
// body, header, or query parameter (FR-028, Constitution IV).
//
// Authentication is out of scope for feature 001, so the only implementation is
// development-only. When registration and login arrive they replace this package and nothing
// else: the collectors table, the foreign keys, and every ownership-scoped query already exist
// (research.md Decision 1).
package identity

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// ErrUnresolved means no collector could be established for this request. Callers must respond
// 401 and disclose no collection content (FR-029).
var ErrUnresolved = errors.New("acting collector could not be resolved")

// CollectorID identifies a collector.
type CollectorID = uuid.UUID

// Resolver answers "who is acting" for a request.
type Resolver interface {
	Resolve(r *http.Request) (CollectorID, error)
}

type contextKey struct{}

// WithCollector stores the resolved collector on the request context. Only the middleware calls
// this; handlers read it back with Collector.
func WithCollector(ctx context.Context, id CollectorID) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// Collector returns the acting collector previously resolved for this request.
//
// A false second return means the request reached a handler without passing the middleware, which
// is a wiring bug. Handlers must treat it as a failure, never as "any collector".
func Collector(ctx context.Context) (CollectorID, bool) {
	id, ok := ctx.Value(contextKey{}).(CollectorID)
	return id, ok
}
