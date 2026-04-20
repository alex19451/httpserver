package server

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/alex19451/httpserver/internal/signature"
	"github.com/rs/zerolog"
)

type responseWriterWithHash struct {
	http.ResponseWriter
	body    *bytes.Buffer
	key     string
	status  int
	written bool
}

func (w *responseWriterWithHash) WriteHeader(statusCode int) {
	if w.written {
		return
	}
	w.status = statusCode
	w.written = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWithHash) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
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

			// Проверяем подпись запроса
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

			// Оборачиваем ResponseWriter для захвата ответа
			ww := &responseWriterWithHash{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				key:            key,
				written:        false,
			}

			next.ServeHTTP(ww, r)

			// Добавляем подпись ответа
			if ww.body.Len() > 0 {
				sign := signature.CalculateHash(ww.body.Bytes(), key)
				w.Header().Set("HashSHA256", sign)
			}

			if !ww.written {
				w.WriteHeader(ww.status)
			}
			if ww.body.Len() > 0 {
				w.Write(ww.body.Bytes())
			}
		})
	}
}
