package main

import "net/http"

func (app *application) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/webhooks", app.listener.HandleNotification)
	mux.HandleFunc("POST /v1/webhooks", app.listener.HandleVerification)

	return mux
}
