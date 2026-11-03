package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Serve(
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

	sig := make(chan os.Signal, 1)

	shutdown := make(chan struct{})

	go func() {
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		s := <-sig

		ctx, stopServer := context.WithTimeout(context.Background(), 20*time.Second)
		defer stopServer()
		srv.Shutdown(ctx)

		fmt.Printf("gracefully shutting down, signal: %v\n", s)
		shutdown <- struct{}{}
	}()

	log.Printf("starting server on port %s", srv.Addr)

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-shutdown
	return nil
}
