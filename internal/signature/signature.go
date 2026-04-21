package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
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

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Bad Request: Invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gzr.Close()
			r.Body = gzr
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gzw.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gzw}, r)
	})
}

func SignatureMiddleware(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "cannot read body", http.StatusInternalServerError)
				return
			}
			r.Body.Close()

			clientHash := r.Header.Get("HashSHA256")
			currentHash := signature.CalculateHash(bodyBytes, key)

			if clientHash != currentHash {
				http.Error(w, "bad body sign", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

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
