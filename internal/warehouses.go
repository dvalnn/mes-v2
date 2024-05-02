package mes

import (
	"context"
	"net/url"
)

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
