package main

import (
	"fmt"
	"net/http"

	"github.com/obelaith/MLOpsTrack/backend/internal/api"
)

func main() {
	port := ":8080"

	router := api.NewRouter()

	fmt.Println("MLOpsTrack server running on", port)

	err := http.ListenAndServe(port, router)
	if err != nil {
		fmt.Println("server error:", err)
	}
}
