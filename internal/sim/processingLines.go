package sim

import (
	plc "mes/internal/net/plc"
	u "mes/internal/utils"
	"sync"
)

type Machine struct {
	name  string
	tools []string
}

// For the inner logic
type conveyorItemHandler struct {
	transformCh chan<- string
	lineEntryCh chan<- string
	lineExitCh  chan<- string
	errCh       chan<- error
}

// To return to the caller
type itemHandler struct {
	transformCh <-chan string
	lineEntryCh <-chan string
	lineExitCh  <-chan string

	errCh <-chan error
}

type conveyorItem struct {
	handler   *conveyorItemHandler
	controlID int16
	useM1     bool
	useM2     bool
}

type Conveyor struct {
	item    *conveyorItem
	machine *Machine
}

func initType1Conveyor() []Conveyor {
	conveyor := make([]Conveyor, LINE_CONVEYOR_SIZE)

	conveyor[LINE_DEFAULT_M1_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M1", tools: []string{u.TOOL_1, u.TOOL_2, u.TOOL_3}},
	}

	conveyor[LINE_DEFAULT_M2_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M2", tools: []string{u.TOOL_1, u.TOOL_2, u.TOOL_3}},
	}

	return conveyor
}

func initType2Conveyor() []Conveyor {
	conveyor := make([]Conveyor, LINE_CONVEYOR_SIZE)

	conveyor[LINE_DEFAULT_M1_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M3", tools: []string{u.TOOL_1, u.TOOL_4, u.TOOL_5}},
	}

	conveyor[LINE_DEFAULT_M2_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M4", tools: []string{u.TOOL_1, u.TOOL_4, u.TOOL_6}},
	}

	return conveyor
}

type ProcessingLine struct {
	plc             *plc.Cell
	id              string
	conveyorLine    []Conveyor
	waitingPieces   []*freeLineWaiter
	readyForNext    bool
	lastLeftPieceId int16
}

type processControlForm struct {
	toolTop    string
	toolBot    string
	pieceKind  string
	id         int16
	processTop bool
	processBot bool
}

func ToolStrToInt(s string) int16 {
	switch s {
	case "T1":
		return 1
	case "T2":
		return 2
	case "T3":
		return 3
	case "T4":
		return 4
	case "T5":
		return 5
	case "T6":
		return 6
	default:
		return 0
	}
}

func PieceStrToInt(s string) int16 {
	switch s {
	case "P1":
		return 1
	case "P2":
		return 2
	case "P3":
		return 3
	case "P4":
		return 4
	case "P5":
		return 5
	case "P6":
		return 6
	case "P7":
		return 7
	case "P8":
		return 8
	case "P9":
		return 9
	default:
		return 0
	}
}

func (pcf *processControlForm) toCellCommand() *plc.CellCommand {
	return &plc.CellCommand{
		TxId:       plc.OpcuaInt16{Value: pcf.id},
		PieceKind:  plc.OpcuaInt16{Value: PieceStrToInt(pcf.pieceKind)},
		ProcessBot: plc.OpcuaBool{Value: pcf.processBot},
		ProcessTop: plc.OpcuaBool{Value: pcf.processTop},
		ToolBot:    plc.OpcuaInt16{Value: ToolStrToInt(pcf.toolBot)},
		ToolTop:    plc.OpcuaInt16{Value: ToolStrToInt(pcf.toolTop)},
	}
}

type freeLineWaiter struct {
	claimed      <-chan struct{}
	claimPieceCh chan<- string
	claimLock    *sync.Mutex
}

func (pl *ProcessingLine) registerWaitingPiece(w *freeLineWaiter) {
	pl.waitingPieces = append(pl.waitingPieces, w)
}

func (pl *ProcessingLine) pruneDeadWaiters() {
	aliveWaiters := make([]*freeLineWaiter, 0, len(pl.waitingPieces))
	for _, w := range pl.waitingPieces {
		select {
		case <-w.claimed:
		default:
			aliveWaiters = append(aliveWaiters, w)
		}
	}

	pl.waitingPieces = aliveWaiters
}

func (pl *ProcessingLine) claimWaitingPiece() {
	u.Assert(pl.readyForNext, "[ProcessingLine.claimPiece] Processing line is not ready")
	pl.pruneDeadWaiters()

loop:
	for _, w := range pl.waitingPieces {
		w.claimLock.Lock()
		select {
		case <-w.claimed:
			w.claimLock.Unlock()
		default:
			w.claimPieceCh <- pl.id
			close(w.claimPieceCh)
			// HACK:
			// not unlocking here on purpose, so that the piece handler
			// can unlock it after the claimed ch is properly closed
			break loop
		}
	}
}

func (pl *ProcessingLine) isMachineCompatibleWith(mIndex int, t *Transformation) bool {
	m := pl.conveyorLine[mIndex].machine

	for _, tool := range m.tools {
		if tool == t.Tool {
			return true
		}
	}
	return false
}

func (pl *ProcessingLine) createBestForm(piece *Piece, id int16) *processControlForm {
	if pl.id == u.ID_L0 {
		return &processControlForm{
			pieceKind: piece.Kind,
			id:        id,
		}
	}

	currentStep := piece.Steps[piece.CurrentStep]
	topCompatible := pl.isMachineCompatibleWith(LINE_DEFAULT_M1_POS, &currentStep)
	botCompatible := pl.isMachineCompatibleWith(LINE_DEFAULT_M2_POS, &currentStep)
	if !topCompatible && !botCompatible {
		return nil
	}

	if topCompatible {
		chainSteps := false
		toolBot := currentStep.Tool // Doesn't matter if there is no step chain

		if piece.CurrentStep+1 < len(piece.Steps) {
			nextStep := piece.Steps[piece.CurrentStep+1]
			toolBot = nextStep.Tool
			chainSteps = pl.isMachineCompatibleWith(LINE_DEFAULT_M2_POS, &nextStep)
		}

		return &processControlForm{
			toolTop:    currentStep.Tool,
			toolBot:    toolBot,
			pieceKind:  piece.Kind,
			id:         id,
			processTop: true,
			processBot: chainSteps,
		}
	}

	return &processControlForm{
		toolTop:    currentStep.Tool,
		toolBot:    currentStep.Tool,
		pieceKind:  piece.Kind,
		id:         id,
		processTop: false,
		processBot: true,
	}
}

func (pl *ProcessingLine) addItem(item *conveyorItem) {
	u.Assert(pl.readyForNext, "[ProcessingLine.addItem] Processing line is not ready")
	u.Assert(pl.conveyorLine[0].item == nil, "[ProcessingLine.addItem] Conveyor is not empty")

	pl.readyForNext = false
	pl.conveyorLine[0].item = item
}

// Moves the newest piece from the start of the conveyor line to the next slot
// This is done separately from the conveyor logic to allow for the piece to be
// acknowledged by the PLC before it is moved along the conveyor, confirming that
// the command was received and processed correctly
func (pl *ProcessingLine) ProgressNewPiece() {
	u.AssertMultiple(
		"[ProcessingLine.AckNewInPiece]",
		[]u.Assertion{
			{
				Message:   "In piece ID does not match last command ID",
				Condition: pl.plc.InPieceTxId() == pl.plc.LastCommandTxId(),
			},
			{
				Message:   "No new piece to ACK",
				Condition: pl.conveyorLine[0].item != nil,
			},
			{
				Message:   "AckNewInPiece called on a line that is marked as ready",
				Condition: !pl.readyForNext,
			},
			{
				Message:   "[ProcessingLine.AckNewInPiece] Next conveyor slot is not empty",
				Condition: pl.conveyorLine[1].item == nil,
			},
		})

	// Not included in the AssertMultiple because it depends on item != nil
	u.Assert(pl.conveyorLine[0].item.controlID == pl.plc.InPieceTxId(),
		"In piece ID does not match the piece ID in the conveyor item",
	)

	nItem := pl.conveyorLine[0].item
	pl.conveyorLine[0].item = nil
	pl.conveyorLine[1].item = nItem
	pl.conveyorLine[1].item.handler.lineEntryCh <- pl.id
	pl.readyForNext = true
}

// Moves the pieces along the conveyor line and sends the necessary signals to the
// item handlers that handle the transformations and communication with the ERP
func (pl *ProcessingLine) progressConveyor() int16 {
	m1 := pl.conveyorLine[LINE_DEFAULT_M1_POS].machine
	m1Item := pl.conveyorLine[LINE_DEFAULT_M1_POS].item
	if m1 != nil && m1Item != nil && m1Item.useM1 {
		m1Item.handler.transformCh <- pl.id
	}

	m2 := pl.conveyorLine[LINE_DEFAULT_M2_POS].machine
	m2Item := pl.conveyorLine[LINE_DEFAULT_M2_POS].item
	if m2 != nil && m2Item != nil && m2Item.useM2 {
		m2Item.handler.transformCh <- pl.id
	}

	outItem := pl.conveyorLine[LINE_CONVEYOR_SIZE-1].item
	if outItem != nil {
		outItem.handler.lineExitCh <- pl.id
		pl.lastLeftPieceId = outItem.controlID
	}

	// Move items along the conveyor line
	// (except the first one that needs to be ACKed)
	for i := LINE_CONVEYOR_SIZE - 1; i > 1; i-- {
		pl.conveyorLine[i].item = pl.conveyorLine[i-1].item
	}
	pl.conveyorLine[1].item = nil

	return pl.lastLeftPieceId
}

func (pl *ProcessingLine) UpdateConveyor() {
	if pl.plc.PieceLeft() {
		reportedOutPieceId := pl.plc.OutPieceTxId()
		iterations := LINE_CONVEYOR_SIZE
		for {
			outPieceId := pl.progressConveyor()
			if outPieceId == reportedOutPieceId {
				break
			}
			iterations--
			u.AssertMultiple(
				"[ProcessingLine.ProgressInternalState]",
				[]u.Assertion{
					{
						Message:   "Out piece ID is greater reported by the PLC",
						Condition: outPieceId < reportedOutPieceId,
					}, {
						Message:   "Infinite loop detected in progressItems",
						Condition: iterations > 0,
					},
				},
			)
		}
	}

	if pl.plc.PieceEnteredM1() {
		// This is here to handle cases where the command has been acked by the PLC
		// but no pieces have left the conveyor yet. In this case, the conveyor should
		// progress but there should not be an outPieceId to report
		if pl.conveyorLine[1].item != nil {
			// There should not be an out piece here, as it would have been caught by
			// the previous progress loop in the pl.plc.PieceLeft() block
			outItem := pl.progressConveyor()
			u.Assert(
				outItem == pl.lastLeftPieceId,
				"[ProcessingLine.ProgressInternalState] Invalid conveyor state",
			)
		}
		pl.ProgressNewPiece()
	}

	if pl.readyForNext {
		pl.claimWaitingPiece()
	}
}
