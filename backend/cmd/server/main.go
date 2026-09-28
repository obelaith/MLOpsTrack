package main

import (
	"fmt"
	"net/http"

	"github.com/obelaith/MLOpsTrack/backend/internal/api"
	"github.com/obelaith/MLOpsTrack/backend/internal/config"
)

func main() {
	cfg := config.Load()

	port := ":" + cfg.Port

	router := api.NewRouter()

	fmt.Println("MLOpsTrack server running on", port)

	err := http.ListenAndServe(port, router)
	if err != nil {
		fmt.Println("server error:", err)
	}
}
