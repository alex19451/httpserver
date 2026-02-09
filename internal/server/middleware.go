package server

import (
	"bytes"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Header() http.Header {
	return w.ResponseWriter.Header()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wr := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           bytes.NewBuffer(nil),
		}

		next.ServeHTTP(wr, r)

		duration := time.Since(start)

		logrus.WithFields(logrus.Fields{
			"uri":           r.RequestURI,
			"method":        r.Method,
			"duration_ms":   duration.Milliseconds(),
			"status_code":   wr.statusCode,
			"response_size": wr.body.Len(),
		}).Info("request completed")
	})
}
