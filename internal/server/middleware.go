package server

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/alex19451/httpserver/internal/signature"
	"github.com/rs/zerolog"
)

type hashWriter struct {
	http.ResponseWriter
	Body   *bytes.Buffer
	Status int
}

func (w *hashWriter) Write(b []byte) (int, error) {
	return w.Body.Write(b)
}

func (w *hashWriter) WriteHeader(statusCode int) {
	w.Status = statusCode
}

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

			wrapper := &hashWriter{
				ResponseWriter: w,
				Body:           &bytes.Buffer{},
				Status:         http.StatusOK,
			}

			next.ServeHTTP(wrapper, r)

			hash := signature.CalculateHash(wrapper.Body.Bytes(), key)
			w.Header().Set("HashSHA256", hash)

			w.WriteHeader(wrapper.Status)
			w.Write(wrapper.Body.Bytes())
		})
	}
}
