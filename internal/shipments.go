package mes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	return PostToErp(ctx, ENDPOINT_SHIPMENT_ARRIVAL, data)
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

func GetExpectedShipments(ctx context.Context, day uint) ([]Shipment, error) {
	endpoint := fmt.Sprintf("%s?day=%d", ENDPOINT_EXPECTED_SHIPMENT, day)
	resp, err := GetFromErp(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var shipments []Shipment
	if err := json.NewDecoder(resp.Body).Decode(&shipments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return shipments, nil
}

func (s *Shipment) Arrived() *ShipmentArrivalForm {
	return &ShipmentArrivalForm{
		ID: s.ID,
	}
}
