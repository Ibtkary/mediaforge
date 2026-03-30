package mediaforge

import (
	"io"
	"log/slog"
)

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// WithLogger sets a structured logger for the Client.
// If not set, logging is disabled (all output discarded).
func WithLogger(l *slog.Logger) Option {
	return func(o *clientOptions) { o.logger = l }
}
