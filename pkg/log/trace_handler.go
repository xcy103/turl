package log

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// traceHandler decorates a slog.Handler to stamp every context-aware log record
// with the active trace_id and span_id, correlating logs with traces. Records
// logged without a span (e.g. plain slog.Info) are passed through unchanged.
type traceHandler struct {
	slog.Handler
}

// withTrace wraps h so its records carry trace identifiers.
func withTrace(h slog.Handler) slog.Handler {
	return traceHandler{Handler: h}
}

// Handle adds trace_id/span_id when the context carries a recording span.
func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs preserves the trace decoration across derived handlers.
func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup preserves the trace decoration across derived handlers.
func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{Handler: h.Handler.WithGroup(name)}
}
