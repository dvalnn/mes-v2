package mes

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type mesEventType int

type mesEvent struct {
	payload   any
	eventType mesEventType
}

func eventHandler(ctx context.Context, event mesEvent) {
	switch event.eventType {
	case EVENT_TIME:
		date, conversionOK := event.payload.(uint)
		assert(conversionOK, fmt.Sprintf("unexpected date type: %v", event.payload))
		go handleNewDate(ctx, date)

	default:
		log.Panicf("unknown event type: %v", event.eventType)
	}
}

func handleNewDate(ctx context.Context, newDate uint) {
	// 1. Notify the ERP system about the new date.
	dForm := DateForm{newDate}
	if err := dForm.Post(ctx); err != nil {
		log.Printf("[Error] posting date = %d failed: %v\n", newDate, err)
		return
	}
	log.Printf("date changed to: %d", newDate)

	wg := sync.WaitGroup{}
	wg.Add(2)

	// 2. Query for shipments that are arriving today.
	go func() {
		defer wg.Done()
		shipments, err := GetExpectedShipments(ctx, newDate)
		if err != nil {
			log.Printf("[Error] getting expected shipments failed: %v\n", err)
			return
		}

		log.Printf("[Info] expected shipments: %v\n", shipments)
		// for _, shipment := range shipments {
		// 	shipment.handle()
		// }
	}()

	// 3. Query for orders that are ready to be delivered.
	go func() {
		defer wg.Done()
		deliveries, err := GetDeliveries(ctx)
		if err != nil {
			log.Printf("[Error] getting deliveries failed: %v\n", err)
		}

		log.Printf("[Info] deliveries: %v\n", deliveries)

		// for _, delivery := range deliveries {
		// 	delivery.handle()
		// }
	}()

	wg.Wait()
}
