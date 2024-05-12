package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mes/internal/net/erp"
	u "mes/internal/utils"
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

	config := erp.ConfigDefaultWithEndpoint(erp.ENDPOINT_TRANSFORMATION)
	return erp.Post(ctx, config, data)
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
	codition := (p.Location == u.ID_W2 && lineID == u.ID_L0) || p.Location == u.ID_W1
	u.Assert(codition, "Piece not in correct warehouse before exiting to line")

	p.Location = lineID
	return &WarehouseExitForm{
		ItemId: p.ErpIdentifier,
		LineId: lineID,
	}
}

func (p *Piece) enterWarehouse(warehouseID string) *WarehouseEntryForm {
	condition := (p.Location == u.ID_L0 && warehouseID == u.ID_W1) || warehouseID == u.ID_W2
	msg := fmt.Sprintf(
		"Piece %s not in correct line before entering to warehouse %s",
		p.ErpIdentifier,
		warehouseID,
	)
	u.Assert(condition, msg)

	p.Location = warehouseID
	return &WarehouseEntryForm{
		ItemId:      p.ErpIdentifier,
		WarehouseId: warehouseID,
	}
}

func (p *Piece) transform(lineID string) *TransfCompletionForm {
	u.Assert(
		p.CurrentStep+1 <= len(p.Steps),
		"Piece current step exceeds steps length",
	)

	p.Kind = p.Steps[p.CurrentStep].ProductKind
	p.ErpIdentifier = p.Steps[p.CurrentStep].ProductID
	completed := p.Steps[p.CurrentStep].Complete(lineID)
	p.CurrentStep++

	return completed
}

func GetPieces(ctx context.Context, quantity uint) ([]Piece, error) {
	endpoint := fmt.Sprintf("%s?max_n_items=%d", erp.ENDPOINT_PRODUCTION, quantity)
	config := erp.ConfigDefaultWithEndpoint(endpoint)
	resp, err := erp.Get(ctx, config)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"[GetProduction] unexpected status code: %d",
			resp.StatusCode,
		)
	}

	var pieceRecipes []Piece
	if err := json.NewDecoder(resp.Body).Decode(&pieceRecipes); err != nil {
		return nil, fmt.Errorf(
			"[GetProduction] failed to unmarshal response: %w",
			err,
		)
	}

	for idx := 0; idx < len(pieceRecipes); idx++ {
		initStep := pieceRecipes[idx].Steps[0]
		pieceRecipes[idx].Kind = initStep.MaterialKind
		pieceRecipes[idx].ErpIdentifier = initStep.MaterialID
		pieceRecipes[idx].Location = u.ID_W1
	}

	return pieceRecipes, nil
}

type PieceHandler struct {
	WakeUpCh chan<- struct{}
	ErrCh    <-chan error
}

func (p *Piece) validateCompletion() {
	u.Assert(
		p.CurrentStep == len(p.Steps),
		"[PieceHandler] Not all steps completed for piece",
	)
	u.Assert(
		p.Location == u.ID_W2,
		"[PieceHandler] Piece location not W2 after completion",
	)
	lastStep := p.Steps[len(p.Steps)-1]
	u.Assert(
		p.ErpIdentifier == lastStep.ProductID,
		"[PieceHandler] Piece ID not the same as the last step product ID",
	)
	u.Assert(
		p.Kind == lastStep.ProductKind,
		"[PieceHandler] Piece kind not the same as the last step product kind",
	)

	log.Printf(
		"[PieceHandler] Piece %v of type %v successfully producted \n",
		p.ErpIdentifier,
		p.Kind,
	)
}

func StartPieceHandler(ctx context.Context) *PieceHandler {
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
			handler = sendToProduction(piece)

			for {
				select {
				case err, open := <-handler.errCh:
					u.Assert(open, "[PieceHandler] error channel closed")
					errCh <- fmt.Errorf("[PieceHandler] %w", err)

				case <-ctx.Done():
					errCh <- fmt.Errorf("[PieceHandler] Context cancelled")
					return

				case line, open := <-handler.lineEntryCh:
					u.Assert(open, "[PieceHandler] lineEntryCh closed")

					if err := piece.exitToProdLine(line).Post(ctx); err != nil {
						errCh <- fmt.Errorf(
							"[PieceHandler] Failed to post warehouse exit: %w",
							err,
						)
					}
					log.Printf(
						"[PieceHandler] Piece %v left warehouse to line %v\n",
						piece.ErpIdentifier,
						line,
					)

				case line, open := <-handler.transformCh:
					u.Assert(open, "[PieceHandler] transformCh closed")

					err := piece.transform(line).Post(ctx)
					if err != nil {
						errCh <- fmt.Errorf(
							"[PieceHandler] Failed to post completion: %w",
							err,
						)
					}
					log.Printf(
						"[PieceHandler] Piece %v transformed at line %v\n",
						piece.ErpIdentifier,
						line,
					)

				case line, open := <-handler.lineExitCh:
					u.Assert(open, "[PieceHandler] lineExitCh closed")

					wID := u.ID_W2
					if line == u.ID_L0 {
						wID = u.ID_W1
					}

					if err := piece.enterWarehouse(wID).Post(ctx); err != nil {
						errCh <- fmt.Errorf(
							"[PieceHandler] Failed to post warehouse entry: %w",
							err,
						)
					}
					log.Printf(
						"[PieceHandler] Piece %v entered warehouse %v\n",
						piece.ErpIdentifier,
						wID,
					)

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
				u.Assert(open, "[PieceHandler] wakeUpCh closed")

				// TODO: rework piece set from erp to always be a new piece
				// TODO: change the hardcoded 100 to a variable or constant
				if newPieces, err := GetPieces(ctx, 100); err != nil {
					errCh <- err
				} else {
					// Should never happen as this function is only
					// waken up when there are new pieces to handle
					u.Assert(len(newPieces) > 0, "[PieceHandler] No new pieces to handle")

					for _, piece := range newPieces {
						// TODO: handle piece priority
						go pieceTracker(ctx, piece)
					}

				}
			}
		}
	}()

	return &PieceHandler{
		WakeUpCh: wakeUpCh,
		ErrCh:    errCh,
	}
}
