package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

func Serve(
	ctx context.Context,
	handler http.Handler,
) error {
	srv := http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownComplete := make(chan error, 1)

	go func() {
		<-ctx.Done()

		shutdownCtx, stopServer := context.WithTimeout(context.Background(), 20*time.Second)
		defer stopServer()

		log.Println("shutting down HTTP server")
		shutdownComplete <- srv.Shutdown(shutdownCtx)
	}()

	log.Printf("starting server on port %s", srv.Addr)

	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return <-shutdownComplete
}
