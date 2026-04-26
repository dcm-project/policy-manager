package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMiddleware(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Middleware Suite")
}

var _ = Describe("ProblemJSON", func() {
	makeHandler := func(status int, contentType string) http.Handler {
		return ProblemJSON(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(status)
			w.Write([]byte(`{"type":"INTERNAL","status":500,"title":"error"}`))
		}))
	}

	It("should rewrite Content-Type for 400 responses", func() {
		rec := httptest.NewRecorder()
		makeHandler(400, "application/json").ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		Expect(rec.Header().Get("Content-Type")).To(Equal("application/problem+json"))
	})

	It("should rewrite Content-Type for 500 responses", func() {
		rec := httptest.NewRecorder()
		makeHandler(500, "application/json").ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		Expect(rec.Header().Get("Content-Type")).To(Equal("application/problem+json"))
	})

	It("should preserve Content-Type for 200 responses", func() {
		rec := httptest.NewRecorder()
		handler := ProblemJSON(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"ok"}`))
		}))
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))
	})

	It("should preserve Content-Type with charset for 400 responses", func() {
		rec := httptest.NewRecorder()
		makeHandler(404, "application/json; charset=utf-8").ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		Expect(rec.Header().Get("Content-Type")).To(Equal("application/problem+json; charset=utf-8"))
	})

	It("should not touch non-JSON content types on errors", func() {
		rec := httptest.NewRecorder()
		makeHandler(500, "text/plain").ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		Expect(rec.Header().Get("Content-Type")).To(Equal("text/plain"))
	})
})
