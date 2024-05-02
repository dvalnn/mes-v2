package mes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var deliveries []Delivery
	if err := json.NewDecoder(resp.Body).Decode(&deliveries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return deliveries, nil
}
