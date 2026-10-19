package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/obelaith/MLOpsTrack/internal/api"
	"github.com/obelaith/MLOpsTrack/internal/db"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	log.Println("database connected")

	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/experiments", api.ExperimentsHandler(database))
	mux.HandleFunc("/runs", api.RunsHandler(database))
	mux.HandleFunc("/params", api.ParamsHandler(database))
	mux.HandleFunc("/metrics", api.MetricsHandler(database))

	log.Printf("server running on port %s", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	}); err != nil {
		log.Printf("write health response: %v", err)
	}
}
