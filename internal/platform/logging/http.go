package logging

import (
	"log"
	"net/http"
	"time"

	httpmiddleware "github.com/lqhiyul/personality-type-test/internal/http/middleware"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Middleware(next http.Handler, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		started := time.Now()
		next.ServeHTTP(rec, r)
		logger.Printf(
			"request_id=%s method=%s path=%s status=%d duration=%s",
			rec.Header().Get(httpmiddleware.RequestIDHeader),
			r.Method,
			r.URL.Path,
			rec.status,
			time.Since(started).Round(time.Millisecond),
		)
	})
}
