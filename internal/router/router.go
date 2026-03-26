package router

import (
	"net/http"
	"strings"

	"github.com/SandeepXT/Card-Transaction-Engine/internal/handlers"
	"github.com/SandeepXT/Card-Transaction-Engine/internal/store"
)

func Build(db *store.MemoryStore, staticDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/transaction", handlers.Transaction(db))
	mux.HandleFunc("/api/card/balance/", handlers.Balance(db))
	mux.HandleFunc("/api/card/transactions/", handlers.TxnHistory(db))
	mux.HandleFunc("/api/health", handlers.Health)
	mux.HandleFunc("/api/cards", handlers.AllCards(db))

	fs := http.FileServer(http.Dir(staticDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})

	return wrap(mux, CORS, Logger)
}
