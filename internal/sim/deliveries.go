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

	// metadata for delivery confirmation
	line int
}

func (d *Delivery) PostConfirmation(ctx context.Context) error {
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
	// Confirmations are sent to this channel
	DeliveryAckCh chan<- int16
	// Errors are reported on this channel
	ErrCh <-chan error
}

func StartDeliveryHandler(ctx context.Context) *DeliveryHandler {
	deliveryCh := make(chan []Delivery)
	deliveryAckCh := make(chan int16, plc.NUMBER_OF_OUTPUTS+1)
	freeLines := [plc.NUMBER_OF_OUTPUTS]bool{true, true, true, true}
	errCh := make(chan error)
	queuedDeliveries := make(map[int16]Delivery)

	go func() {
		defer close(deliveryCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return

			case lineIdx := <-deliveryAckCh:
				delivery, ok := queuedDeliveries[lineIdx]
				if !ok {
					log.Printf("[DeliveryHandler] Delivery not found for confirmation\n")
					continue
				}
				// u.Assert(ok, "[DeliveryHandler] Delivery not found for confirmation")
				func() {
					defer delete(queuedDeliveries, lineIdx)

					for i := 0; i < len(freeLines); i++ {
						err := delivery.PostConfirmation(ctx)
						u.Assert(err == nil, "[DeliveryHandler] Error confirming delivery")

						log.Printf("[DeliveryHandler] Delivery %v confirmed to ERP\n", delivery.ID)
						freeLines[delivery.line] = true
					}
				}()

			case deliveries := <-deliveryCh:
				log.Printf("[DeliveryHandler] Received %d new deliveries\n", len(deliveries))
				for _, delivery := range deliveries {
					linesRemaining := 0
					for _, line := range freeLines {
						if line {
							linesRemaining++
						}
					}

					log.Printf(
						"[DeliveryHandler] Delivery %v: %d pieces of type %v",
						delivery.ID,
						delivery.Quantity,
						delivery.Piece,
					)

					neededLines := int(math.Ceil(float64(delivery.Quantity) / DELIVERY_LINE_CAPACITY))
					log.Printf("[DeliveryHandler] Delivery %s needs: %d lines\n",
						delivery.ID, neededLines)

					if linesRemaining < neededLines {
						log.Printf("[DeliveryHandler] No lines available for delivery %s\n",
							delivery.ID)
						continue
					}
					linesRemaining -= neededLines

					func() {
						piecesRemaining := delivery.Quantity
						factory, mutex := getFactoryInstance()
						defer mutex.Unlock()

						writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
						defer cancel()

						for i := 0; i < neededLines; i++ {
							for lIdx, line := range factory.deliveryLines {
								if piecesRemaining == 0 {
									break
								}

								if !freeLines[lIdx] {
									continue
								}

								quantity := piecesRemaining
								if quantity > DELIVERY_LINE_CAPACITY {
									quantity = DELIVERY_LINE_CAPACITY
								}

								piecesRemaining -= quantity

								line.SetDelivery(int16(quantity), PieceStrToInt(delivery.Piece))
								log.Printf("[DeliveryHandler] Delivering %d pieces of type %v to line %d\n",
									quantity, delivery.Piece, lIdx)
								_, err := factory.plcClient.Write(line.CommandOpcuaVars(), writeCtx)
								u.Assert(err == nil, "[DeliveryHandler] Error writing to delivery line")
								freeLines[lIdx] = false
								queuedDeliveries[line.LastCommandTxId()] = delivery
							}
							u.Assert(piecesRemaining == 0, "[DeliveryHandler] Wrong number of pieces delivered")
						}
					}()
				}
			}
		}
	}()

	return &DeliveryHandler{
		DeliveryCh:    deliveryCh,
		DeliveryAckCh: deliveryAckCh,
		ErrCh:         errCh,
	}
}
