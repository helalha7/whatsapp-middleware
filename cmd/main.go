package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/helalha7/whatsapp-middleware/internal/event"
	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/helalha7/whatsapp-middleware/internal/queue"
	"github.com/helalha7/whatsapp-middleware/internal/sender"
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

	ibp := worker.NewPool[event.InboundMessage](5, queue.InboundMessagesQueue, send)
	ctx, stopWorkers := context.WithCancel(context.Background())
	ibp.Start(ctx)

	if err := Serve(mux, stopWorkers); err != nil {
		log.Fatal(err)
	}
}

func send(e event.InboundMessage) {
	sender.New(http.DefaultClient).SendTextMessage(
		e.PhoneNumberID,
		os.Getenv("access_token"),
		e.From,
		e.Text.Body,
	)
}
