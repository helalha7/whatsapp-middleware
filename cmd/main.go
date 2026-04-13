package main

import (
	"fmt"
	"net/http"
	"whatsconnect/internal/infrastructure/webhook"
)

func main() {

	mux := http.NewServeMux()

	webhook.RegisterRoutes(mux, &webhook.Handler{})

	fmt.Println("Server is running...")
	http.ListenAndServe(":8080", mux)
}
