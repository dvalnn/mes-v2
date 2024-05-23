package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"mes/internal/net/erp"
	"mes/internal/net/plc"
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
				log.Printf("[DeliveryHandler] Received %d new deliveries\n", len(deliveries))

				linesRemaining := plc.NUMBER_OF_OUTPUTS
				startingLine := 0
				for _, delivery := range deliveries {
					log.Printf(
						"[DeliveryHandler] Delivery %v: %d pieces of type %v",
						delivery.ID,
						delivery.Quantity,
						delivery.Piece,
					)
					if linesRemaining <= 0 {
						log.Printf("[DeliveryHandler] No more delivery lines available")
						continue
					}

					neededLines := int(math.Ceil(float64(delivery.Quantity) / DELIVERY_LINE_CAPACITY))
					log.Printf("[DeliveryHandler] Delivery %s needs: %d lines\n",
						delivery.ID, neededLines)
					linesRemaining -= neededLines

					func() {
						piecesRemaining := delivery.Quantity
						factory, mutex := getFactoryInstance()
						defer mutex.Unlock()

						writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
						defer cancel()

						for i := range neededLines {
							dl := factory.deliveryLines[startingLine+i]
							quantity := piecesRemaining
							if quantity > DELIVERY_LINE_CAPACITY {
								quantity = DELIVERY_LINE_CAPACITY
							}
							piecesRemaining -= quantity
							log.Printf("[DeliveryHandler] Writing to delivery line %d; %v of type %v\n",
								i, quantity, delivery.Piece)
							dl.SetDelivery(int16(quantity), PieceStrToInt(delivery.Piece))
							_, err := factory.plcClient.Write(dl.OpcuaVars(), writeCtx)
							u.Assert(err == nil, "[DeliveryHandler] Error writing to delivery line")
						}
						u.Assert(piecesRemaining == 0, "[DeliveryHandler] Wrong number of pieces delivered")
						startingLine += neededLines
					}()

					// 3 - Confirm the delivery to the ERP
					err := delivery.PostConfirmation(ctx, delivery.ID)
					u.Assert(err == nil, "[DeliveryHandler] Error confirming delivery")
					log.Printf("[DeliveryHandler] Delivery %v confirmed\n", delivery.ID)
				}
			}
		}
	}()

	return &DeliveryHandler{
		DeliveryCh: deliveryCh,
		ErrCh:      errCh,
	}
}
