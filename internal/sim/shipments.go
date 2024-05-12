package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mes/internal/net/erp"
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
	// Errors are reported on this channel
	ErrCh <-chan error
}

func StartShipmentHandler(
	ctx context.Context,
	pieceWakeUp chan<- struct{},
) *ShipmentHandler {
	shipCh := make(chan []Shipment)
	errCh := make(chan error)

	go func() {
		defer close(shipCh)
		defer close(errCh)

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

					// TODO: 1 - Communicate new shipments to the PLCs
					log.Printf(
						"[ShipmentHandler] Communicating shipment %d to PLCs",
						shipment.ID,
					)
					time.Sleep(time.Second)

					// TODO: 2 - Wait for each shipment arrival to be confirmed
					log.Printf(
						"[ShipmentHandler] Waiting for shipment %d to arrive",
						shipment.ID,
					)
					time.Sleep(time.Second)

					// 3 - Communicate the arrival of each shipment to the ERP
					log.Printf("[ShipmentHandler] Shipment %d arrived", shipment.ID)
					if err := shipment.arrived().Post(ctx); err != nil {
						errCh <- fmt.Errorf(
							"[ShipmentHandler] error confirming shipment arrival: %v",
							err.Error(),
						)
					}

					pieceWakeUp <- struct{}{}
				}
			}
		}
	}()

	return &ShipmentHandler{
		ShipCh: shipCh,
		ErrCh:  errCh,
	}
}
