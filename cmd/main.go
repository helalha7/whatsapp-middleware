package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/helalha7/whatsapp-middleware/internal/event"
	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/helalha7/whatsapp-middleware/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	listener.Register(mux, os.Getenv("verify_token"), os.Getenv("app_secret"))

	ibp := worker.NewInboundPool(5, make(<-chan event.InboundMessage))
	ctx, stopWorkers := context.WithCancel(context.Background())
	ibp.Start(ctx)

	if err := Serve(mux, stopWorkers); err != nil {
		log.Fatal(err)
	}
}
