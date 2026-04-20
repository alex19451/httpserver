package server

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/alex19451/httpserver/internal/signature"
	"github.com/rs/zerolog"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	written    bool
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if w.written {
		return
	}
	w.statusCode = statusCode
	w.written = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func LoggingMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           bytes.NewBuffer(nil),
				written:        false,
			}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			logger.Info().
				Str("uri", r.RequestURI).
				Str("method", r.Method).
				Dur("duration", duration).
				Int("status_code", ww.statusCode).
				Msg("request")
		})
	}
}

func SignatureMiddleware(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			expectedHash := r.Header.Get("HashSHA256")
			if expectedHash == "" {
				http.Error(w, "missing HashSHA256 header", http.StatusBadRequest)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			if !signature.VerifyHash(bodyBytes, expectedHash, key) {
				http.Error(w, "invalid signature", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			rec := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           bytes.NewBuffer(nil),
				written:        false,
			}

			next.ServeHTTP(rec, r)

			if rec.body.Len() > 0 {
				sign := signature.CalculateHash(rec.body.Bytes(), key)
				w.Header().Set("HashSHA256", sign)
			}

			if !rec.written {
				w.WriteHeader(rec.statusCode)
			}
			if rec.body.Len() > 0 {
				w.Write(rec.body.Bytes())
			}
		})
	}
}
