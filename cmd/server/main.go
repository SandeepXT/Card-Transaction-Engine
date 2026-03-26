package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/SandeepXT/Card-Transaction-Engine/internal/router"
	"github.com/SandeepXT/Card-Transaction-Engine/internal/store"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}

	static := filepath.Join(cwd, "static")
	if _, err := os.Stat(static); os.IsNotExist(err) {
		log.Fatalf("static dir not found at %s — run from project root", static)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db := store.NewMemoryStore()
	addr := fmt.Sprintf(":%s", port)

	log.Printf("card-transaction-engine  listening on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, router.Build(db, static)); err != nil {
		log.Fatalf("server: %v", err)
	}
}
