package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := loadConfig()
	log.Println("loaded configurations")

	app, err := newApplication(cfg)
	if err != nil {
		return err
	}
	defer app.close()
	log.Println("connection to the database established")

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	return app.run(ctx)
}
