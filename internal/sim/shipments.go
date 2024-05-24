package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mes/internal/net/erp"
	plc "mes/internal/net/plc"
	"mes/internal/utils"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

/*
*
* SHIPMENT ARRIVAL HANDLING
*
 */

// ShipmentArrivalForm is a form used to post the arrival of a shipment to the ERP.
// It contains the ID of the shipment that arrived.
//
// Implements the ErpPoster interface.
type ShipmentArrivalForm struct {
	ID int `json:"shipment_id"`
}

func (s *ShipmentArrivalForm) Post(ctx context.Context) error {
	data := url.Values{
		"shipment_id": {strconv.Itoa(s.ID)},
	}
	config := erp.ConfigDefaultWithEndpoint(erp.ENDPOINT_SHIPMENT_ARRIVAL)
	return erp.Post(ctx, config, data)
}

/*
*
*  SHIPMENT HANDLING
*
 */
type Shipment struct {
	MaterialKind string `json:"material_type"`
	ID           int    `json:"shipment_id"`
	NPieces      int    `json:"quantity"`
}

// TODO: Check if erp is returning shipments that already arrived and fix it
func GetShipments(ctx context.Context, day uint) ([]Shipment, error) {
	endpoint := fmt.Sprintf("%s?day=%d", erp.ENDPOINT_EXPECTED_SHIPMENT, day)
	config := erp.ConfigDefaultWithEndpoint(endpoint)
	resp, err := erp.Get(ctx, config)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[GetShipments] unexpected status code: %d", resp.StatusCode)
	}

	var shipments []Shipment
	if err := json.NewDecoder(resp.Body).Decode(&shipments); err != nil {
		return nil, fmt.Errorf("[GetShipments] failed to unmarshal response: %w", err)
	}
	return shipments, nil
}

func (s *Shipment) arrived() *ShipmentArrivalForm {
	return &ShipmentArrivalForm{
		ID: s.ID,
	}
}

type ShipmentHandler struct {
	// Send new shipments to this channel
	ShipCh chan<- []Shipment
	// ShipmentAckCh chan<- Shipment
	ShipAckCh chan<- int16
	// Errors are reported on this channel
	ErrCh <-chan error
}

func StartShipmentHandler(
	ctx context.Context,
	pieceWakeUpCh chan<- struct{},
) *ShipmentHandler {
	shipCh := make(chan []Shipment)
	shipAckCh := make(chan int16, plc.NUMBER_OF_SUPPLY_LINES+1)
	errCh := make(chan error)

	go func() {
		defer close(errCh)
		defer close(pieceWakeUpCh)

		for {
			select {
			case <-ctx.Done():
				return

			case shipments := <-shipCh:
				for _, shipment := range shipments {
					log.Printf(
						"[ShipmentHandler] New shipment (id %d): %d pieces of type %v",
						shipment.ID,
						shipment.NPieces,
						shipment.MaterialKind,
					)

					// 1 - Communicate new shipments to the PLCs
					log.Printf(
						"[ShipmentHandler] Communicating shipment %d to PLCs",
						shipment.ID,
					)
					nArrived := 0
					for nArrived < shipment.NPieces {
						// NOTE: Running in a func to defer the mutex unlock
						var expectedAcks []int16
						func() {
							factory, mutex := getFactoryInstance()
							defer mutex.Unlock()

							writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
							defer cancel()

							for i := 0; i < len(factory.supplyLines); i++ {
								if nArrived >= shipment.NPieces {
									break
								}
								log.Printf("[ShipmentHandler] Communicating with supply line %d", i)
								material := PieceStrToInt(shipment.MaterialKind)
								factory.supplyLines[i].NewShipment(material)
								_, err := factory.plcClient.Write(
									factory.supplyLines[i].CommandOpcuaVars(),
									writeCtx,
								)
								utils.Assert(err == nil, "[ShipmentHandler] Error writing to supply line")
								expectedAcks = append(expectedAcks, factory.supplyLines[i].LastCommandTxId())
								nArrived++
							}
						}()

						utils.Assert(len(expectedAcks) > 0, "[ShipmentHandler] No supply lines to write to")

						// NOTE: Wait all expected shipments to arrive (be acked)
						for len(expectedAcks) > 0 {
							acked := <-shipAckCh
							ackedIdx := -1
							for i, ack := range expectedAcks {
								if ack == acked {
									ackedIdx = i
									break
								}
							}
							utils.Assert(ackedIdx != -1, "[ShipmentHandler] Unexpected ack")
							expectedAcks = append(expectedAcks[:ackedIdx], expectedAcks[ackedIdx+1:]...)
						}
					}

					// 2 - Communicate the arrival of each shipment to the ERP
					log.Printf("[ShipmentHandler] Shipment %d arrived", shipment.ID)
					if err := shipment.arrived().Post(ctx); err != nil {
						errCh <- fmt.Errorf(
							"[ShipmentHandler] error confirming shipment arrival: %v",
							err.Error(),
						)
					}

					pieceWakeUpCh <- struct{}{}
				}
			}
		}
	}()

	return &ShipmentHandler{
		ShipCh:    shipCh,
		ShipAckCh: shipAckCh,
		ErrCh:     errCh,
	}
}
