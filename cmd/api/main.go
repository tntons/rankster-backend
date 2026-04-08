package main

import (
	"log"
	"net/http"
	"os"

	"rankster-backend/internal/config"
	"rankster-backend/internal/db"
	"rankster-backend/internal/handlers"
	"rankster-backend/internal/server"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}

	router := server.BuildRouter(database)
	handlers.RegisterRoutes(router, database)

	addr := cfg.Host + ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}

	_ = os.Stdout
}

