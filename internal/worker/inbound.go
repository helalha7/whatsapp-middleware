package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/helalha7/whatsapp-middleware/internal/event"
)

// worker pool responsible for handling incoming messages
type InboundPool struct {
	// maximum number of worker goroutines running concurrently
	workers int

	// recieve only channel from which workers consume inbound messages
	jobs <-chan event.InboundMessage

	wg sync.WaitGroup

	so sync.Once
}

func NewInboundPool(
	workers int,
	jobs <-chan event.InboundMessage,
) *InboundPool {
	return &InboundPool{
		workers: workers,
		jobs:    jobs,
	}
}

func (p *InboundPool) Start(ctx context.Context) {
	p.so.Do(
		func() {
			for i := 0; i < p.workers; i++ {
				p.wg.Add(1)

				go func(id int) {
					defer p.wg.Done()

					for {
						select {
						case <-ctx.Done():
							return
						case msg, ok := <-p.jobs:
							if !ok {
								return
							}

							fmt.Printf("Worker %d recieved: %+v\n", id, msg)
						}
					}
				}(i)
			}
		},
	)
}

func (p *InboundPool) Wait() {
	p.wg.Wait()
}
