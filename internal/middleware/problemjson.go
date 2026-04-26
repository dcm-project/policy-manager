package middleware

import (
	"net/http"
	"strings"
)

// ProblemJSON is a Chi middleware that rewrites the Content-Type header
// from application/json to application/problem+json for error responses
// (status >= 400), per AEP-193 and RFC 9457.
func ProblemJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&problemJSONWriter{ResponseWriter: w}, r)
	})
}

type problemJSONWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *problemJSONWriter) WriteHeader(code int) {
	if !w.wroteHeader && code >= 400 {
		ct := w.Header().Get("Content-Type")
		if strings.HasPrefix(ct, "application/json") {
			w.Header().Set("Content-Type", strings.Replace(ct, "application/json", "application/problem+json", 1))
		}
	}
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *problemJSONWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *problemJSONWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
