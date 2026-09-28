package main

import (
	"fmt"
	"net/http"
)

func main() {
	port := ":8080"

	fmt.Println("MLOpsTrack server running on", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Println("server error:", err)
	}
}
