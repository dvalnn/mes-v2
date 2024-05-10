package mes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	ErpIdentifier string
	Kind          string
	Location      string
	Steps         []Transformation `json:"steps"`
	CurrentStep   int
	ControlID     int16
}

func (p *Piece) exitToProdLine(lineID string) *WarehouseExitForm {
	codition := (p.Location == ID_W2 && lineID == ID_L0) || p.Location == ID_W1
	assert(codition, "Piece not in correct warehouse before exiting to line")

	p.Location = lineID
	return &WarehouseExitForm{
		ItemId: p.ErpIdentifier,
		LineId: lineID,
	}
}

func (p *Piece) enterWarehouse(warehouseID string) *WarehouseEntryForm {
	condition := (p.Location == ID_L0 && warehouseID == ID_W1) || warehouseID == ID_W2
	assert(condition, "Piece not in correct line before entering to warehouse")

	p.Location = warehouseID
	return &WarehouseEntryForm{
		ItemId:      p.ErpIdentifier,
		WarehouseId: warehouseID,
	}
}

func (p *Piece) transform(lineID string) *TransfCompletionForm {
	assert(p.CurrentStep+1 <= len(p.Steps), "Piece current step exceeds steps length")

	p.Kind = p.Steps[p.CurrentStep].ProductKind
	p.ErpIdentifier = p.Steps[p.CurrentStep].ProductID
	completed := p.Steps[p.CurrentStep].Complete(lineID)
	p.CurrentStep++

	return completed
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

type PieceHandler struct {
	wakeUpCh chan<- struct{}
	errCh    <-chan error
}

func (p *Piece) validateCompletion() {
	assert(
		p.CurrentStep == len(p.Steps),
		"[PieceHandler] Not all steps completed for piece",
	)
	assert(
		p.Location == ID_W2,
		"[PieceHandler] Piece location not W2 after completion",
	)
	lastStep := p.Steps[len(p.Steps)-1]
	assert(
		p.ErpIdentifier == lastStep.ProductID,
		"[PieceHandler] Piece ID not the same as the last step product ID",
	)
	assert(
		p.Kind == lastStep.ProductKind,
		"[PieceHandler] Piece kind not the same as the last step product kind",
	)

	log.Printf(
		"[PieceHandler] Piece %v of type %v successfully producted \n",
		p.ErpIdentifier,
		p.Kind,
	)
}

func startPieceHandler(ctx context.Context) *PieceHandler {
	errCh := make(chan error)
	wakeUpCh := make(chan struct{})

	pieceTracker := func(ctx context.Context, piece Piece) {
		var handler *itemHandler

		log.Printf("[PieceHandler] Handling piece %v transform from %v to %v)\n",
			piece.Steps[0].MaterialID,
			piece.Steps[0].MaterialKind,
			piece.Steps[len(piece.Steps)-1].ProductKind,
		)

	StepLoop:
		for piece.CurrentStep < len(piece.Steps) {
			// TODO: 1 - Select/Wait for a free compatible line
			handler = sendToProduction(piece)

			for {
				select {
				case err, open := <-handler.errCh:
					assert(open, "[PieceHandler] error channel closed")
					errCh <- fmt.Errorf("[PieceHandler] %w", err)

				case <-ctx.Done():
					errCh <- fmt.Errorf("[PieceHandler] Context cancelled")
					return

				case line, open := <-handler.lineEntryCh:
					assert(open, "[PieceHandler] lineEntryCh closed")

					if err := piece.enterWarehouse(line).Post(ctx); err != nil {
						errCh <- fmt.Errorf("[PieceHandler] Failed to post warehouse exit: %w", err)
					}

				case line, open := <-handler.transformCh:
					assert(open, "[PieceHandler] transformCh closed")

					err := piece.transform(line).Post(ctx)
					if err != nil {
						errCh <- fmt.Errorf("[PieceHandler] Failed to post completion: %w", err)
					}

				case wID, open := <-handler.lineExitCh:
					assert(open, "[PieceHandler] lineExitCh closed")

					if err := piece.exitToProdLine(wID).Post(ctx); err != nil {
						errCh <- fmt.Errorf("[PieceHandler] Failed to post warehouse entry: %w", err)
					}

					continue StepLoop
				}
			}
		}

		piece.validateCompletion()
	}

	go func() {
		defer close(errCh)
		defer close(wakeUpCh)

		for {
			select {
			case <-ctx.Done():
				return

			case _, open := <-wakeUpCh:
				assert(open, "[PieceHandler] wakeUpCh closed")

				// TODO: rework piece set from erp to always be a new piece
				// TODO: change the hardcoded 100 to a variable or constant
				if newPieces, err := GetPieces(ctx, 100); err != nil {
					errCh <- err
				} else {
					// Should never happen as this function is only
					// waken up when there are new pieces to handle
					assert(len(newPieces) > 0, "[PieceHandler] No new pieces to handle")

					for _, piece := range newPieces {
						// TODO: handle piece priority
						go pieceTracker(ctx, piece)
					}

				}
			}
		}
	}()

	return &PieceHandler{
		wakeUpCh: wakeUpCh,
		errCh:    errCh,
	}
}
