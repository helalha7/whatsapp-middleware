package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/helalha7/whatsapp-middleware/internal/event"
	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/helalha7/whatsapp-middleware/internal/messaging"
	"github.com/helalha7/whatsapp-middleware/internal/queue"
	"github.com/helalha7/whatsapp-middleware/internal/sender"
	"github.com/helalha7/whatsapp-middleware/internal/tenant"
	"github.com/helalha7/whatsapp-middleware/internal/worker"
)

const (
	inboundQueueCapacity  = 100
	inboundWorkerCount    = 5
	inboundProcessTimeout = 15 * time.Second
)

type application struct {
	db             *sql.DB
	handler        http.Handler
	inboundQueue   *queue.Queue[event.InboundMessage]
	inboundWorkers *worker.WorkerPool[event.InboundMessage]
}

func newApplication(cfg config) (*application, error) {
	db, err := connectDB(cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	inboundQueue := queue.New[event.InboundMessage](inboundQueueCapacity)
	mux := http.NewServeMux()
	stores := newStores(db)
	messageProcessor := messaging.NewProcessor(
		stores.tenants,
		stores.conversations,
		stores.chatSessions,
		stores.messages,
		sender.New(&http.Client{Timeout: 10 * time.Second}),
		cfg.ai.sessionIdleTimeout,
	)

	listener.RegisterRoutes(
		mux,
		cfg.meta.verifyToken,
		cfg.meta.appSecret,
		stores.inboundReceipts,
		inboundQueue,
	)
	tenant.RegisterRoutes(mux, stores.tenants)

	return &application{
		db:           db,
		handler:      mux,
		inboundQueue: inboundQueue,
		inboundWorkers: worker.NewPool(
			inboundWorkerCount,
			inboundQueue.Messages(),
			processInboundMessage(messageProcessor),
		),
	}, nil
}

func (app *application) run(ctx context.Context) error {
	app.inboundWorkers.Start(context.Background())
	log.Println("started inbound message workers")

	err := Serve(ctx, app.handler)

	app.inboundQueue.Close()
	app.inboundWorkers.Wait()
	log.Println("stopped inbound message workers")

	return err
}

func (app *application) close() {
	if err := app.db.Close(); err != nil {
		log.Printf("close database connection: %v", err)
	}
}

func processInboundMessage(processor *messaging.Processor) func(event.InboundMessage) {
	return func(message event.InboundMessage) {
		ctx, cancel := context.WithTimeout(context.Background(), inboundProcessTimeout)
		defer cancel()

		if err := processor.Process(ctx, message); err != nil {
			log.Printf(
				"process inbound message id=%s waba_id=%s: %v",
				message.ID,
				message.WABAID,
				err,
			)
			return
		}

		log.Printf(
			"processed inbound message id=%s waba_id=%s type=%s",
			message.ID,
			message.WABAID,
			message.Type,
		)
	}
}
