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

	dateCh := dateCounter(ctx)
	shipmentHandler := startShipmentHandler(ctx)
	deliveryHandler := startDeliveryHandler(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case date := <-dateCh:
			shipments, deliveries := date.HandleNew(ctx)
			shipmentHandler.shipCh <- shipments
			deliveryHandler.deliveryCh <- deliveries

		case shipError := <-shipmentHandler.errCh:
			log.Panicf("[Error] [ShipmentHandler] %v\n", shipError)

		case deliveryError := <-deliveryHandler.errCh:
			log.Panicf("[Error] [DeliveryHandler] %v\n", deliveryError)

		}
	}
}
