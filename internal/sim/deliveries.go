package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mes/internal/net/erp"
	u "mes/internal/utils"
	"net/http"
	"net/url"
	"time"
)

type Delivery struct {
	ID       string `json:"id"`
	Piece    string `json:"piece"`
	Quantity int    `json:"quantity"`
}

func (d *Delivery) PostConfirmation(ctx context.Context, id string) error {
	formData := url.Values{
		"id": {d.ID},
	}

	config := erp.ConfigDefaultWithEndpoint(erp.ENDPOINT_DELIVERY)
	return erp.Post(ctx, config, formData)
}

func GetDeliveries(ctx context.Context) ([]Delivery, error) {
	config := erp.ConfigDefaultWithEndpoint(erp.ENDPOINT_DELIVERY)
	resp, err := erp.Get(ctx, config)
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
	DeliveryCh chan<- []Delivery
	// Errors are reported on this channel
	ErrCh <-chan error
}

func StartDeliveryHandler(ctx context.Context) *DeliveryHandler {
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
				for _, delivery := range deliveries {
					log.Printf(
						"[DeliveryHandler] New delivery (id %v): %d pieces of type %v",
						delivery.ID,
						delivery.Quantity,
						delivery.Piece,
					)

					// TODO: 1 - Communicate new deliveries to the PLCs
					log.Printf(
						"[DeliveryHandler] Communicating delivery %v to PLCs",
						delivery.ID,
					)
					time.Sleep(500 * time.Millisecond)

					// TODO: 2 - Wait for each delivery to be confirmed
					log.Printf(
						"[DeliveryHandler] Delivering %d %v pieces",
						delivery.Quantity,
						delivery.Piece,
					)
					time.Sleep(500 * time.Millisecond)

					// TODO: 3 - Confirm the delivery to the ERP
					err := delivery.PostConfirmation(ctx, delivery.ID)
					u.Assert(err == nil, "[DeliveryHandler] Error confirming delivery")
				}
			}
		}
	}()

	return &DeliveryHandler{
		DeliveryCh: deliveryCh,
		ErrCh:      errCh,
	}
}
