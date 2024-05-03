package mes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

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

type Piece struct {
	Steps       []Transformation `json:"steps"`
	CurrentStep int
	ControlID   int16 // ControlID of the piece in Codesys
}

func GetPieces(ctx context.Context, quantity uint) ([]Piece, error) {
	url := fmt.Sprintf("%s?max_n_items=%d", ENDPOINT_PRODUCTION, quantity)
	resp, err := GetFromErp(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[GetProduction] unexpected status code: %d", resp.StatusCode)
	}

	var pieceRecipes []Piece
	if err := json.NewDecoder(resp.Body).Decode(&pieceRecipes); err != nil {
		return nil, fmt.Errorf("[GetProduction] failed to unmarshal response: %w", err)
	}
	return pieceRecipes, nil
}
