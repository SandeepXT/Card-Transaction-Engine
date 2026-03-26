package router

import (
	"log"
	"net/http"
	"time"
)

type recorder struct {
	http.ResponseWriter
	code int
}

func (rc *recorder) WriteHeader(c int) {
	rc.code = c
	rc.ResponseWriter.WriteHeader(c)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		rec := &recorder{w, http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%-6s %-45s %d  %s", r.Method, r.URL.Path, rec.code, time.Since(t))
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func wrap(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
