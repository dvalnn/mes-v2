package mes

import (
	"context"
	"log"
	"mes/internal/sim"
	"time"
)

// Run starts the MES operation.
// It blocks until the context is canceled.
// simTime (> 0) is the simulation time period.
func Run(ctx context.Context, simTime time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dateCh := sim.DateCounter(ctx, simTime)
	deliveryHandler := sim.StartDeliveryHandler(ctx)
	pieceHandler := sim.StartPieceHandler(ctx)
	shipmentHandler := sim.StartShipmentHandler(ctx, pieceHandler.WakeUpCh)
	factoryErrorCh := sim.StartFactoryHandler(ctx, shipmentHandler.ShipAckCh)

	defer close(shipmentHandler.ShipCh)
	defer close(deliveryHandler.DeliveryCh)

	for {
		select {
		case <-ctx.Done():
			return

		case date := <-dateCh:
			shipments, deliveries := date.HandleNew(ctx)
			shipmentHandler.ShipCh <- shipments
			deliveryHandler.DeliveryCh <- deliveries

		case shipError := <-shipmentHandler.ErrCh:
			log.Panicf("[mes.Run] %v\n", shipError)

		case deliveryError := <-deliveryHandler.ErrCh:
			log.Panicf("[mes.Run] %v\n", deliveryError)

		case pieceError := <-pieceHandler.ErrCh:
			log.Panicf("[mes.Run] %v\n", pieceError)

		case factoryError := <-factoryErrorCh:
			log.Panicf("[mes.Run] %v\n", factoryError)

		}
	}
}
