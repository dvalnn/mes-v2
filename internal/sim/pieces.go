package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mes/internal/net/erp"
	"mes/internal/utils"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
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
	MachineID        string
	TransformationID int
	TimeTaken        int
	ToolChange       bool
}

func (t *TransfCompletionForm) Post(ctx context.Context) error {
	total_time := t.TimeTaken
	if t.ToolChange {
		total_time += MACHINE_TOOL_SWAP_TIME
	}

	data := url.Values{
		"transf_id":   {strconv.Itoa(t.TransformationID)},
		"material_id": {t.MaterialID},
		"product_id":  {t.ProductID},
		"line_id":     {t.LineID},
		"machine_id":  {t.MachineID},
		"time_taken":  {strconv.Itoa(total_time)},
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

func (t *Transformation) Complete(
	lineID string, machineID string, toolChange bool,
) *TransfCompletionForm {
	return &TransfCompletionForm{
		MaterialID:       t.MaterialID,
		ProductID:        t.ProductID,
		LineID:           lineID,
		MachineID:        machineID,
		ToolChange:       toolChange,
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
	codition := (p.Location == utils.ID_W2 && lineID == utils.ID_L0) || p.Location == utils.ID_W1
	utils.Assert(codition, "Piece not in correct warehouse before exiting to line")

	p.Location = lineID
	return &WarehouseExitForm{
		ItemId: p.ErpIdentifier,
		LineId: lineID,
	}
}

func (p *Piece) enterWarehouse(warehouseID string) *WarehouseEntryForm {
	condition := (p.Location == utils.ID_L0 && warehouseID == utils.ID_W1) || warehouseID == utils.ID_W2
	msg := fmt.Sprintf(
		"Piece %s not in correct line before entering to warehouse %s",
		p.ErpIdentifier,
		warehouseID,
	)
	utils.Assert(condition, msg)

	p.Location = warehouseID
	return &WarehouseEntryForm{
		ItemId:      p.ErpIdentifier,
		WarehouseId: warehouseID,
	}
}

func (p *Piece) transform(
	lineID string, machineID string, toolChange bool,
) *TransfCompletionForm {
	utils.Assert(
		p.CurrentStep+1 <= len(p.Steps),
		"Piece current step exceeds steps length",
	)

	p.Kind = p.Steps[p.CurrentStep].ProductKind
	p.ErpIdentifier = p.Steps[p.CurrentStep].ProductID
	completed := p.Steps[p.CurrentStep].Complete(lineID, machineID, toolChange)
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
		pieceRecipes[idx].Location = utils.ID_W1
	}

	return pieceRecipes, nil
}

type PieceHandler struct {
	WakeUpCh chan<- struct{}
	ErrCh    <-chan error
}

func (p *Piece) validateCompletion() {
	utils.Assert(
		p.CurrentStep == len(p.Steps),
		"[PieceHandler] Not all steps completed for piece",
	)
	utils.Assert(
		p.Location == utils.ID_W2,
		"[PieceHandler] Piece location not W2 after completion",
	)
	lastStep := p.Steps[len(p.Steps)-1]
	utils.Assert(
		p.ErpIdentifier == lastStep.ProductID,
		"[PieceHandler] Piece ID not the same as the last step product ID",
	)
	utils.Assert(
		p.Kind == lastStep.ProductKind,
		"[PieceHandler] Piece kind not the same as the last step product kind",
	)

	log.Printf(
		"[PieceHandler] Piece %v of type %v successfully produced \n",
		p.ErpIdentifier,
		p.Kind,
	)
}

func StartPieceHandler(ctx context.Context) *PieceHandler {
	errCh := make(chan error)
	wakeUpCh := make(chan struct{})

	piecePool := make(map[string]struct{})
	piecePoolLock := sync.Mutex{}

	pieceTracker := func(ctx context.Context, piece Piece) {
		var handler *itemHandler

		log.Printf("[PieceHandler] Handling piece %v transform from %v to %v)\n",
			piece.Steps[0].MaterialID,
			piece.Steps[0].MaterialKind,
			piece.Steps[len(piece.Steps)-1].ProductKind,
		)

		watchdogTimeout := 10 * time.Minute

	StepLoop:
		for piece.CurrentStep < len(piece.Steps) {

			nextState := "lineEntry"
			func() {
				ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				handler = sendToProduction(ctx, &piece)
			}()
			log.Printf("[PieceHandler] Piece %v sent to production at step (%d of %d)\n",
				piece.ErpIdentifier,
				piece.CurrentStep,
				len(piece.Steps))

			for {
				watchdog := time.NewTimer(watchdogTimeout)
				select {
				case <-watchdog.C:
					log.Printf(
						"[PieceHandler - WARNING] Watchdog timeout. Piece %s waiting for state %s",
						piece.ErpIdentifier, nextState)
					continue StepLoop // restart the loop, register again

				case err, open := <-handler.errCh:
					utils.Assert(open, "[PieceHandler] error channel closed")
					errCh <- fmt.Errorf("[PieceHandler] %w", err)

				case <-ctx.Done():
					errCh <- fmt.Errorf("[PieceHandler] Context cancelled")
					return

				case line, open := <-handler.lineEntryCh:
					nextState = "transform"

					utils.Assert(open, "[PieceHandler] lineEntryCh closed")

					if err := piece.exitToProdLine(line).Post(ctx); err != nil {
						errCh <- fmt.Errorf(
							"[PieceHandler] Piece %s failed to post warehouse exit: %w",
							piece.ErpIdentifier,
							err,
						)
					}
					log.Printf(
						"[PieceHandler] Piece %v left warehouse to line %v\n",
						piece.ErpIdentifier,
						line,
					)

				case lineData, open := <-handler.transformCh:
					nextState = "lineExitCh"

					utils.Assert(open, "[PieceHandler] transformCh closed")

					parts := strings.Split(lineData, ",")
					utils.Assert(len(parts) == 3, "Invalid lineData format")
					line, machine := parts[0], parts[1]
					toolChange, err := strconv.ParseBool(parts[2])
					utils.Assert(err == nil, "Invalid tool change format")

					err = piece.transform(line, machine, toolChange).Post(ctx)
					if err != nil {
						errCh <- fmt.Errorf(
							"[PieceHandler] Piece %s failed to post completion: %w",
							piece.ErpIdentifier,
							err,
						)
					}
					log.Printf(
						"[PieceHandler] Piece %v transformed at line %s, by machine %s (step %d of %d)\n",
						piece.ErpIdentifier, line, machine, piece.CurrentStep, len(piece.Steps))

				case line, open := <-handler.lineExitCh:
					nextState = "lineEntry"

					utils.Assert(open, "[PieceHandler] lineExitCh closed")

					wID := utils.ID_W2
					if line == utils.ID_L0 {
						wID = utils.ID_W1
					}

					// Ack the warehouse entry
					func() {
						factory, mutex := getFactoryInstance()
						defer mutex.Unlock()

						writeContext, cancel := context.WithTimeout(ctx, 10*time.Second)
						defer cancel()

						plc := factory.processLines[line].plc
						plc.AckPiece(piece.ControlID)

						_, err := factory.plcClient.Write(plc.AckOpcuaVars(), writeContext)
						utils.Assert(err == nil,
							"[PieceHandler] Error acknowledging warehouse entry")
					}()

					if err := piece.enterWarehouse(wID).Post(ctx); err != nil {
						errCh <- fmt.Errorf(
							"[PieceHandler] Piece %s failed to post warehouse entry: %w",
							piece.ErpIdentifier,
							err,
						)
					}
					log.Printf(
						"[PieceHandler] Piece %v entered warehouse %v\n",
						piece.ErpIdentifier,
						wID,
					)

					watchdog.Stop()
					continue StepLoop
				}
				watchdog.Stop()
			}
		}

		piece.validateCompletion()
		piecePoolLock.Lock()
		defer piecePoolLock.Unlock()
		delete(piecePool, piece.ErpIdentifier)
	}

	go func() {
		defer close(errCh)
		defer close(wakeUpCh)

		for {
			select {
			case <-ctx.Done():
				return

			case _, open := <-wakeUpCh:
				utils.Assert(open, "[PieceHandler] wakeUpCh closed")

				if newPieces, err := GetPieces(ctx, 32); err != nil {
					errCh <- err
				} else {
					// Should never happen as this function is only
					// waken up when there are new pieces to handle
					utils.Assert(len(newPieces) > 0, "[PieceHandler] No new pieces to handle")

					func() {
						piecePoolLock.Lock()
						defer piecePoolLock.Unlock()

						for _, piece := range newPieces {
							if _, ok := piecePool[piece.ErpIdentifier]; !ok {
								piecePool[piece.ErpIdentifier] = struct{}{}
								go pieceTracker(ctx, piece)
							}
						}
					}()

				}
			}
		}
	}()

	return &PieceHandler{
		WakeUpCh: wakeUpCh,
		ErrCh:    errCh,
	}
}
