package mes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func GetFromErp(ctx context.Context, endpoint string) (*http.Response, error) {
	timeout, exists := ctx.Value(KEY_HTTP_TIMEOUT).(time.Duration)
	assert(exists, "http timeout not found in context")
	baseUrl, exists := ctx.Value(KEY_ERP_URL).(string)
	assert(exists, "erp url not found in context")

	client := http.Client{
		Timeout: timeout,
	}

	url := fmt.Sprintf("%s%s", baseUrl, endpoint)
	return client.Get(url)
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

// Transformation represents a transformation operation in the ERP system.
//
// Can be converted to a TransfCompletionForm for posting to the ERP using
// the Complete method.
type Transformation struct {
	MaterialID   string `json:"material_id"`
	ProductID    string `json:"product_id"`
	MaterialKind string `json:"material_kind"`
	ProductKind  string `json:"product_kind"`
	Tool         string `json:"tool"`
	ID           int    `json:"transformation_id"`
	Time         int    `json:"operation_time"`
}

func (t *Transformation) Complete(lineID string) *TransfCompletionForm {
	return &TransfCompletionForm{
		MaterialID:       t.MaterialID,
		ProductID:        t.ProductID,
		LineID:           lineID,
		TransformationID: t.ID,
		TimeTaken:        t.Time,
	}
}

type PieceRecipe struct {
	Steps []Transformation `json:"steps"`
}

func GetProduction(ctx context.Context, quantity uint) ([]PieceRecipe, error) {
	url := fmt.Sprintf("%s?max_n_items=%d", ENDPOINT_PRODUCTION, quantity)
	resp, err := GetFromErp(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var pieceRecipes []PieceRecipe
	if err := json.NewDecoder(resp.Body).Decode(&pieceRecipes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return pieceRecipes, nil
}

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
