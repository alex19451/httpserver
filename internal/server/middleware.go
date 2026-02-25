package server

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		logrus.WithFields(logrus.Fields{
			"uri":         r.RequestURI,
			"method":      r.Method,
			"duration":    duration.String(),
			"status_code": ww.statusCode,
		}).Info("request")
	})
}
