package sim

import (
	"context"
	"log"
	"mes/internal/net/erp"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type DateForm struct {
	Day uint
}

func (d *DateForm) post(ctx context.Context) error {
	data := url.Values{
		"day": {strconv.Itoa(int(d.Day))},
	}

	config := erp.ConfigDefaultWithEndpoint(erp.ENDPOINT_NEW_DATE)
	return erp.Post(ctx, config, data)
}

func DateCounter(ctx context.Context, sleepPeriod time.Duration) <-chan DateForm {
	dateCh := make(chan DateForm)
	go func() {
		defer close(dateCh)

		date := DateForm{5}
		dateCh <- date

		sleeper := time.NewTimer(sleepPeriod)
		for {
			select {
			case <-ctx.Done():
				sleeper.Stop()
				log.Println("[DateCounter] timeKeeper stopped")
				return

			case <-sleeper.C:
				date.Day++
				dateCh <- date
				sleeper.Reset(sleepPeriod)
			}
		}
	}()

	return dateCh
}

func (d *DateForm) HandleNew(ctx context.Context) (
	newShipments []Shipment,
	newDeliveries []Delivery,
) {
	// 1. Notify the ERP system about the new date.
	if err := d.post(ctx); err != nil {
		log.Printf("[DateForm.HandleNew] posting date = %d failed: %v\n", d.Day, err)
		return
	}
	log.Printf("[DateForm.HandleNew] date changed to: %d", d.Day)

	wg := sync.WaitGroup{}
	wg.Add(2)

	// 2. Query for shipments that are arriving today.
	go func() {
		defer wg.Done()
		ship, err := GetShipments(ctx, d.Day)
		if err != nil {
			log.Printf("[DateForm.HandleNew] getting expected shipments failed: %v\n", err)
		}
		newShipments = ship
	}()

	// 3. Query for orders that are ready to be delivered.
	go func() {
		defer wg.Done()
		del, err := GetDeliveries(ctx)
		if err != nil {
			log.Printf("[DateForm.HandleNew] getting deliveries failed: %v\n", err)
		}
		newDeliveries = del
	}()

	wg.Wait()

	log.Printf("[DateForm.HandleNew] expected shipments for day %d: %v\n", d.Day, newShipments)
	log.Printf("[DateForm.HandleNew] deliveries for day %d: %v\n", d.Day, newDeliveries)
	return
}
