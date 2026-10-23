package middleware

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type requestIDKey struct{}
type logMetaKey struct{}

type logMeta struct {
	UserID *uuid.UUID
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(data)
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestIDFrom(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

func logMetaFrom(ctx context.Context) *logMeta {
	meta, _ := ctx.Value(logMetaKey{}).(*logMeta)
	return meta
}

func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.NewString()
		meta := &logMeta{}

		ctx := WithRequestID(r.Context(), requestID)
		ctx = context.WithValue(ctx, logMetaKey{}, meta)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)

		recorder := &responseRecorder{ResponseWriter: w}
		started := time.Now()

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		attrs := []any{
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(started).Milliseconds(),
		}

		if meta.UserID != nil {
			attrs = append(attrs, "user_id", meta.UserID.String())
		}

		if auctionID, err := uuid.Parse(r.PathValue("id")); err == nil {
			attrs = append(attrs, "auction_id", auctionID.String())
		}

		logger.Info("http request", attrs...)
	})
}
