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
	stopWorkers context.CancelFunc,
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

	go func() {
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		s := <-sig

		time.Sleep(2 * time.Second)

		stopWorkers()
		time.Sleep(2 * time.Second)

		fmt.Printf("gracefully shutting down %v\n", s)
		os.Exit(1)
	}()

	log.Printf("starting server on port %s", srv.Addr)

	return srv.ListenAndServe()
}
