package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/helalha7/whatsapp-middleware/internal/event"
	"github.com/helalha7/whatsapp-middleware/internal/idempotency"
	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/helalha7/whatsapp-middleware/internal/queue"
	"github.com/helalha7/whatsapp-middleware/internal/tenant"
	"github.com/helalha7/whatsapp-middleware/internal/worker"
)

const (
	inboundQueueCapacity = 100
	inboundWorkerCount   = 5
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

	listener.RegisterRoutes(
		mux,
		cfg.meta.verifyToken,
		cfg.meta.appSecret,
		&idempotency.Store{DB: db},
		inboundQueue,
	)
	tenant.RegisterRoutes(mux, &tenant.Store{DB: db})

	return &application{
		db:           db,
		handler:      mux,
		inboundQueue: inboundQueue,
		inboundWorkers: worker.NewPool(
			inboundWorkerCount,
			inboundQueue.Messages(),
			processInboundMessage,
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

func processInboundMessage(message event.InboundMessage) {
	log.Printf(
		"processing inbound message id=%s waba_id=%s type=%s",
		message.ID,
		message.WABAID,
		message.Type,
	)
}
