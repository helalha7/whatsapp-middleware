package main

import "net/http"

func (app *application) Routes() http.Handler {
	mux := http.NewServeMux()

	//webhooks
	mux.HandleFunc("GET /v1/webhooks", app.listener.HandleVerification)
	mux.HandleFunc("POST /v1/webhooks", app.listener.HandleNotification)

	return mux
}
