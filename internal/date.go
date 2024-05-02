package mes

import (
	"context"
	"log"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type DateForm struct {
	Day uint
}

func (d *DateForm) Post(ctx context.Context) error {
	data := url.Values{
		"day": {strconv.Itoa(int(d.Day))},
	}

	return PostToErp(ctx, ENDPOINT_NEW_DATE, data)
}

func dateCounter(ctx context.Context) <-chan DateForm {
	sleepTime, exists := ctx.Value(KEY_SIM_TIME).(time.Duration)
	assert(exists, "[DateCounter] simulation time not found in context")
	assert(sleepTime > 0, "[DateCounter] simulation time must be positive")

	dateCh := make(chan DateForm)
	go func() {
		defer close(dateCh)

		date := DateForm{1}
		dateCh <- date

		sleeper := time.NewTimer(sleepTime)
		for {
			select {
			case <-ctx.Done():
				sleeper.Stop()
				log.Println("[DateCounter] timeKeeper stopped")
				return

			case <-sleeper.C:
				date.Day++
				dateCh <- date
				sleeper.Reset(sleepTime)
			}
		}
	}()

	return dateCh
}

func (d *DateForm) HandleNew(ctx context.Context) {
	// 1. Notify the ERP system about the new date.
	if err := d.Post(ctx); err != nil {
		log.Printf("[Error] [DateForm.HandleNew] posting date = %d failed: %v\n", d.Day, err)
		return
	}
	log.Printf("date changed to: %d", d.Day)

	wg := sync.WaitGroup{}
	wg.Add(2)

	// 2. Query for shipments that are arriving today.
	go func() {
		defer wg.Done()
		shipments, err := GetExpectedShipments(ctx, d.Day)
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
