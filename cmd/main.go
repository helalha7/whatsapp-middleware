package main

import (
	"net/http"

	"github.com/helalha7/whatsapp-middleware/infrastructure/listener"
)

func main() {
	mux := http.NewServeMux()

	listener := listener.NewListener("helal212")

	mux.HandleFunc("GET /webhooks", listener.HandleVerification)
	mux.HandleFunc("POST /webhooks", listener.HandleNotification)

	http.ListenAndServe(":8080", mux)
}
