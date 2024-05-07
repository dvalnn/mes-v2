package mes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Delivery struct {
	Id       string `json:"id"`
	Piece    string `json:"piece"`
	Quantity int    `json:"quantity"`
}

func GetDeliveries(ctx context.Context) ([]Delivery, error) {
	resp, err := GetFromErp(ctx, ENDPOINT_DELIVERY)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[GetDeliveries] unexpected status code: %d", resp.StatusCode)
	}

	var deliveries []Delivery
	if err := json.NewDecoder(resp.Body).Decode(&deliveries); err != nil {
		return nil, fmt.Errorf("[GetDeliveries] failed to unmarshal response: %w", err)
	}
	return deliveries, nil
}

type DeliveryHandler struct {
	// Send new shipments to this channel
	deliveryCh chan<- []Delivery
	// Errors are reported on this channel
	errCh <-chan error
}

func startDeliveryHandler(ctx context.Context) *DeliveryHandler {
	deliveryCh := make(chan []Delivery)
	errCh := make(chan error)

	go func() {
		defer close(deliveryCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return

			case deliveries := <-deliveryCh:

				if len(deliveries) == 0 {
					log.Println("[DeliveryHandler] No deliveries to process")
				}

				for _, delivery := range deliveries {
					log.Printf(
						"[DeliveryHandler] New delivery (id %v): %d pieces of type %v",
						delivery.Id,
						delivery.Quantity,
						delivery.Piece,
					)

					// TODO: 1 - Communicate new deliveries to the PLCs
					log.Printf(
						"[DeliveryHandler] Communicating delivery %v to PLCs",
						delivery.Id,
					)
					time.Sleep(time.Second)

					// TODO: 2 - Wait for each delivery to be confirmed
					log.Printf(
						"[DeliveryHandler] Delivering %d %v pieces",
						delivery.Quantity,
						delivery.Piece,
					)
					time.Sleep(time.Second)

					// TODO: 3 - Confirm the delivery to the ERP
					log.Printf(
						"[DeliveryHandler] Confirming delivery %v to ERP",
						delivery.Id,
					)
					time.Sleep(time.Second)
				}
			}
		}
	}()

	return &DeliveryHandler{
		deliveryCh: deliveryCh,
		errCh:      errCh,
	}
}
