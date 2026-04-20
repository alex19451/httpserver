package server

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/alex19451/httpserver/internal/signature"
	"github.com/rs/zerolog"
)

func LoggingMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info().
				Str("uri", r.RequestURI).
				Str("method", r.Method).
				Dur("duration", time.Since(start)).
				Msg("request")
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
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

			rec := &responseRecorder{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(rec, r)

			if rec.body.Len() > 0 {
				sign := signature.CalculateHash(rec.body.Bytes(), key)
				w.Header().Set("HashSHA256", sign)
				w.Write(rec.body.Bytes())
			}
		})
	}
}
