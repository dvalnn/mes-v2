package mes

import (
	"context"
	"log"
	"time"
)

func assert(condition bool, message string) {
	if !condition {
		log.Panicln("Assertion failed:", message)
	}
}

func timeKeeper(ctx context.Context, eventCh chan<- mesEvent) {
	sleepTime, exists := ctx.Value(KEY_SIM_TIME).(time.Duration)
	assert(exists, "simulation time not found in context")
	assert(sleepTime > 0, "simulation time must be positive")

	date := uint(1)
	eventCh <- mesEvent{
		eventType: EVENT_TIME,
		payload:   date,
	}

	sleeper := time.NewTimer(sleepTime)
	for {
		select {
		case <-ctx.Done():
			sleeper.Stop()
			log.Println("timeKeeper stopped")
			return

		case <-sleeper.C:
			date++
			eventCh <- mesEvent{
				eventType: EVENT_TIME,
				payload:   date,
			}
			sleeper.Reset(sleepTime)
		}
	}
}

type SupplyLine struct{}

type ProcessingLine struct{}

type DeliveryLine struct{}

type Factory struct {
	supplyLines   []SupplyLine
	processLines  []ProcessingLine
	deliveryLines []DeliveryLine
}

func InitFactory() *Factory {
	return &Factory{
		supplyLines:   []SupplyLine{},
		processLines:  []ProcessingLine{},
		deliveryLines: []DeliveryLine{},
	}
}

// Run starts the MES operation.
// It blocks until the context is canceled.
// simTime (> 0) is the simulation time period.
func Run(ctx context.Context, simTime time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = context.WithValue(ctx, KEY_SIM_TIME, simTime)
	ctx = context.WithValue(ctx, KEY_HTTP_TIMEOUT, DEFAULT_HTTP_TIMEOUT)
	ctx = context.WithValue(ctx, KEY_ERP_URL, DEFAULT_ERP_URL)

	eventCh := make(chan mesEvent)

	go timeKeeper(ctx, eventCh)

	for {
		event := <-eventCh
		eventHandler(ctx, event)
	}
}
