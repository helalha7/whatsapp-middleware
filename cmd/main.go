package main

import (
	"fmt"
	"net/http"
	"whatsconnect/internal/whatsapp"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /webhook", whatsapp.VerifyWebhook)
	mux.HandleFunc("POST /webhook", whatsapp.HandleWebhook)

	fmt.Println("Server is running...")
	http.ListenAndServe(":8080", mux)
}
