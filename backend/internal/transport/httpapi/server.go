package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/identity"
)

// Server wires the transport to the application services.
type Server struct {
	service  *collection.Service
	resolver identity.Resolver
	dev      *identity.DevResolver // nil outside development
}

func NewServer(service *collection.Service, resolver identity.Resolver, dev *identity.DevResolver) *Server {
	return &Server{service: service, resolver: resolver, dev: dev}
}

// Routes returns the handler for the whole API.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// The four operations in contracts/openapi.yaml. Nothing else is exposed: there is no
	// collectible-detail endpoint because the spec is gallery-only.
	mux.Handle("POST /api/collectibles", s.requireCollector(s.handleAddCollectible))
	mux.Handle("GET /api/collectibles", s.requireCollector(s.handleListCollectibles))
	mux.Handle("POST /api/images", s.requireCollector(s.handleUploadImage))
	mux.Handle("GET /api/images/{imageId}/rendition", s.requireCollector(s.handleGetRendition))

	// Development sign-in. Present only when VAULTORY_DEV_IDENTITY=enabled, and deliberately
	// absent from the contract: it is local scaffolding, not part of the API. It mints a session
	// for anyone who asks, which is exactly why it must never ship.
	if s.dev != nil {
		mux.HandleFunc("POST /api/dev/session", s.handleDevSession)
	}

	return requestLogger(mux)
}

// collectorHandler is a handler that has already had its collector resolved.
type collectorHandler func(w http.ResponseWriter, r *http.Request, collector identity.CollectorID)

// requireCollector resolves the acting collector before the handler runs, and refuses the request
// if it cannot.
//
// Every API route goes through this. It is the only place identity is established, and it consults
// only the resolver — never a header, query parameter, or body field the client controls
// (FR-028). An unresolvable collector yields 401 and no collection content (FR-029).
func (s *Server) requireCollector(h collectorHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := s.resolver.Resolve(r)
		if err != nil {
			WriteUnauthenticated(w)
			return
		}
		ctx := identity.WithCollector(r.Context(), id)
		h(w, r.WithContext(ctx), id)
	})
}

func (s *Server) handleDevSession(w http.ResponseWriter, r *http.Request) {
	// "second" selects the other seeded collector, so the privacy walkthroughs can be run from a
	// browser or curl rather than only from the integration suite.
	collector := identity.CollectorFor(r.URL.Query().Get("collector"))
	s.dev.IssueSession(w, collector)
	WriteJSON(w, http.StatusOK, map[string]string{"collectorId": collector.String()})
}

// requestLogger records the shape of each request. It deliberately logs no request body, no query
// values, and no collector-supplied content: a log that echoes what a collector typed is a log
// that leaks their collection.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.written {
		r.status = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.written = true
	return r.ResponseWriter.Write(b)
}
