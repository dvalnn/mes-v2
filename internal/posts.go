package mes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PostToErp sends a POST request to the ERP system at the given endpoint
// with the provided form data. It returns an error if the request fails.
// Form data is sent as x-www-form-urlencoded.
//
// Context must be provided with values for
// - KEY_ERP_URL (erp base url - string)
// - KEY_HTTP_TIMEOUT (timeout for client request - time.Duration)
func PostToErp(ctx context.Context, endpoint string, formData url.Values) error {
	timeout, exists := ctx.Value(KEY_HTTP_TIMEOUT).(time.Duration)
	assert(exists, "http timeout not found in context")

	baseUrl, exists := ctx.Value(KEY_ERP_URL).(string)
	assert(exists, "erp url not found in context")

	client := http.Client{
		Timeout: timeout,
	}

	url := fmt.Sprintf("%s%s", baseUrl, endpoint)
	resp, err := client.PostForm(url, formData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// ErpPoster is an interface that defines the methods required to post
// data to the ERP system.
// Data is sent in the x-www-form-urlencoded format.
//
// Context must be provided with values for
// - KEY_ERP_URL (erp base url - string)
// - KEY_HTTP_TIMEOUT (timeout for client request - time.Duration)
type ErpPoster interface {
	Post(ctx context.Context) error
}

type DateForm struct {
	Day uint
}

func (d *DateForm) Post(ctx context.Context) error {
	data := url.Values{
		"day": {strconv.Itoa(int(d.Day))},
	}

	return PostToErp(ctx, ENDPOINT_NEW_DATE, data)
}

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

// WarehouseExitForm is a form used to post the exit of an item from a warehouse to the ERP.
// It contains the ID of the item and the ID of the line the item is exiting to.
//
// Implements the ErpPoster interface.
type WarehouseExitForm struct {
	ItemId string
	LineId string
}

func (w *WarehouseExitForm) Post(ctx context.Context) error {
	data := url.Values{
		"item_id": {w.ItemId},
		"exit":    {w.LineId},
	}

	return PostToErp(ctx, ENDPOINT_WAREHOUSE, data)
}

// WarehouseEntryForm is a form used to post the entry of an item to a warehouse to the ERP.
// It contains the ID of the item and the ID of the warehouse the item is entering.
//
// Implements the ErpPostForm interface.
type WarehouseEntryForm struct {
	ItemId      string
	WarehouseId string
}

func (w *WarehouseEntryForm) Post(ctx context.Context) error {
	data := url.Values{
		"item_id": {w.ItemId},
		"entry":   {w.WarehouseId},
	}
	return PostToErp(ctx, ENDPOINT_WAREHOUSE, data)
}

// TransfCompletionForm is a form used to post the completion of a transformation to the ERP.
// It contains the ID of the material, the ID of the product, the ID of the line,
// the ID of the transformation, and the time taken to complete the transformation.
//
// Implements the ErpPoster interface.
type TransfCompletionForm struct {
	MaterialID       string
	ProductID        string
	LineID           string
	TransformationID int
	TimeTaken        int
}

func (t *TransfCompletionForm) Post(ctx context.Context) error {
	data := url.Values{
		"transf_id":   {strconv.Itoa(t.TransformationID)},
		"material_id": {t.MaterialID},
		"product_id":  {t.ProductID},
		"line_id":     {t.LineID},
		"time_taken":  {strconv.Itoa(t.TimeTaken)},
	}

	return PostToErp(ctx, ENDPOINT_TRANSFORMATION, data)
}
