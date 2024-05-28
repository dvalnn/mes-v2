package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"mes/internal/net/erp"
	"mes/internal/net/plc"
	"mes/internal/utils"
	"net/http"
	"net/url"
)

type Delivery struct {
	ID       string `json:"id"`
	Piece    string `json:"piece"`
	Quantity int    `json:"quantity"`

	nConfirmations int
}

type DeliveryAckMetadata struct {
	txId int16
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
		return nil,
			fmt.Errorf("[GetDeliveries] unexpected status code: %d", resp.StatusCode)
	}

	var deliveries []Delivery
	if err := json.NewDecoder(resp.Body).Decode(&deliveries); err != nil {
		return nil,
			fmt.Errorf("[GetDeliveries] failed to unmarshal response: %w", err)
	}
	return deliveries, nil
}

type DeliveryHandler struct {
	// Send new shipments to this channel
	DeliveryCh chan<- []Delivery
	// Confirmations are sent to this channel
	DeliveryAckCh chan<- DeliveryAckMetadata
	// Errors are reported on this channel
	ErrCh <-chan error
}

func StartDeliveryHandler(ctx context.Context) *DeliveryHandler {
	deliveryCh := make(chan []Delivery)
	deliveryAckCh := make(chan DeliveryAckMetadata, plc.NUMBER_OF_OUTPUTS+1)
	errCh := make(chan error)

	freeLines := [plc.NUMBER_OF_OUTPUTS]bool{true, true, true, true}
	metadataMap := make(map[DeliveryAckMetadata]Delivery) // metadata -> delivery
	confirmationsMap := make(map[string]int)              // delivery ID -> number of confirmations received

	go func() {
		defer close(deliveryCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return

			case metadata := <-deliveryAckCh:
				delivery, ok := metadataMap[metadata]
				utils.Assert(ok, "[DeliveryHandler] Delivery not found for confirmation")

				freeLines[metadata.line] = true
				confirmationsMap[delivery.ID]++
				utils.Assert(confirmationsMap[delivery.ID] <= delivery.nConfirmations,
					fmt.Sprintf("[DeliveryHandler] Too many confirmations received for delivery %v", delivery.ID))

				log.Printf("[DeliveryHandler] Delivery %v partially executed on line %d\n",
					delivery.ID, metadata.line)

				log.Printf("[DeliveryHandler] Delivery %v has %d confirmations out of %d\n",
					delivery.ID, confirmationsMap[delivery.ID], delivery.nConfirmations)

				if confirmationsMap[delivery.ID] == delivery.nConfirmations {
					err := delivery.PostConfirmation(ctx)
					utils.Assert(err == nil, "[DeliveryHandler] Error confirming delivery")
					log.Printf("[DeliveryHandler] Delivery %v confirmed to ERP\n", delivery.ID)
					delete(confirmationsMap, delivery.ID)
				}
				delete(metadataMap, metadata)

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

						writeCtx, cancel := context.WithTimeout(ctx, plc.DEFAULT_OPCUA_TIMEOUT)
						defer cancel()

						delivery.nConfirmations = neededLines
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

								utils.Assert(err == nil, "[DeliveryHandler] Error writing to delivery line")
								freeLines[lIdx] = false
								metadataMap[DeliveryAckMetadata{
									txId: line.LastCommandTxId(),
									line: lIdx,
								}] = delivery
							}

							utils.Assert(piecesRemaining == 0, "[DeliveryHandler] Wrong number of pieces delivered")
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
