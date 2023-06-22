package middleware

import (
	"net/http"

	"github.com/weaveworks/common/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

// Tracer is a middleware which traces incoming requests.
type Tracer struct {
	RouteMatcher RouteMatcher
	SourceIPs    *SourceIPExtractor
}

// Wrap implements Interface
func (t Tracer) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := telemetry.GetTracer()
		// Start a new span for the incoming request
		ctx, span := tracer.Start(r.Context(), "HTTP "+r.Method)
		defer span.End()

		// add a tag with the client's user agent to the span
		userAgent := r.Header.Get("User-Agent")
		if userAgent != "" {
			span.SetAttributes(attribute.String("http.user_agent", userAgent))
		}

		// add a tag with the client's sourceIPs to the span, if a
		// SourceIPExtractor is given.
		if t.SourceIPs != nil {
			span.SetAttributes(attribute.String("sourceIPs", t.SourceIPs.Get(r)))
		}

		// Pass the modified context with the span to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
