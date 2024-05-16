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
	conveyor := make([]Conveyor, u.LINE_CONVEYOR_SIZE)

	conveyor[u.LINE_DEFAULT_M1_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M1", tools: []string{u.TOOL_1, u.TOOL_2, u.TOOL_3}},
	}

	conveyor[u.LINE_DEFAULT_M2_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M2", tools: []string{u.TOOL_1, u.TOOL_2, u.TOOL_3}},
	}

	return conveyor
}

func initType2Conveyor() []Conveyor {
	conveyor := make([]Conveyor, u.LINE_CONVEYOR_SIZE)

	conveyor[u.LINE_DEFAULT_M1_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M3", tools: []string{u.TOOL_1, u.TOOL_4, u.TOOL_5}},
	}

	conveyor[u.LINE_DEFAULT_M2_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M4", tools: []string{u.TOOL_1, u.TOOL_4, u.TOOL_6}},
	}

	return conveyor
}

type ProcessingLine struct {
	plc           *plc.Cell
	id            string
	conveyorLine  []Conveyor
	waitingPieces []*freeLineWaiter
	readyForNext  bool
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

func (pl *ProcessingLine) isReady() bool {
	return pl.readyForNext
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
	u.Assert(pl.isReady(), "[ProcessingLine.claimPiece] Processing line is not ready")
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
	topCompatible := pl.isMachineCompatibleWith(u.LINE_DEFAULT_M1_POS, &currentStep)
	botCompatible := pl.isMachineCompatibleWith(u.LINE_DEFAULT_M2_POS, &currentStep)
	if !topCompatible && !botCompatible {
		return nil
	}

	if topCompatible {
		chainSteps := false
		toolBot := currentStep.Tool // Doesn't matter if there is no step chain

		if piece.CurrentStep+1 < len(piece.Steps) {
			nextStep := piece.Steps[piece.CurrentStep+1]
			toolBot = nextStep.Tool
			chainSteps = pl.isMachineCompatibleWith(u.LINE_DEFAULT_M2_POS, &nextStep)
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
	u.Assert(pl.isReady(), "[ProcessingLine.addItem] Processing line is not ready")
	u.Assert(pl.conveyorLine[0].item == nil, "[ProcessingLine.addItem] Conveyor is not empty")

	pl.readyForNext = false
	pl.conveyorLine[0].item = item
}

func (pl *ProcessingLine) progressConveyor() int16 {
	inItem := pl.conveyorLine[0].item
	if inItem != nil {
		inItem.handler.lineEntryCh <- pl.id
	}

	m1 := pl.conveyorLine[u.LINE_DEFAULT_M1_POS].machine
	m1Item := pl.conveyorLine[u.LINE_DEFAULT_M1_POS].item
	if m1 != nil && m1Item != nil && m1Item.useM1 {
		m1Item.handler.transformCh <- pl.id
	}

	m2 := pl.conveyorLine[u.LINE_DEFAULT_M2_POS].machine
	m2Item := pl.conveyorLine[u.LINE_DEFAULT_M2_POS].item
	if m2 != nil && m2Item != nil && m2Item.useM2 {
		m2Item.handler.transformCh <- pl.id
	}

	var outID int16 = 0
	outItem := pl.conveyorLine[u.LINE_CONVEYOR_SIZE-1].item
	if outItem != nil {
		outItem.handler.lineExitCh <- pl.id
		outID = outItem.controlID
	}
	// Move items along the conveyor line
	for i := u.LINE_CONVEYOR_SIZE - 1; i > 0; i-- {
		pl.conveyorLine[i].item = pl.conveyorLine[i-1].item
	}
	pl.conveyorLine[0].item = nil
	pl.readyForNext = true

	return outID
}

func (pl *ProcessingLine) ProgressInternalState() {
	reportedInPieceId := pl.plc.InPieceTxId()
	lastCommandId := pl.plc.LastCommandTxId()
	u.Assert(
		reportedInPieceId == lastCommandId,
		"[factoryStateUpdate] In piece ID does not match last command ID",
	)

	reportedOutPieceId := pl.plc.OutPieceTxId()
	iterations := u.LINE_CONVEYOR_SIZE
	for {
		outPieceId := pl.progressConveyor()
		u.Assert(
			outPieceId <= reportedOutPieceId,
			"[factoryStateUpdate] Out piece ID is greater reported by the PLC",
		)
		if outPieceId == reportedOutPieceId {
			break
		}
		u.Assert(
			iterations > 0,
			"[factoryStateUpdate] Infinite loop detected in progressItems",
		)
		iterations--
	}

	u.Assert(
		pl.isReady(),
		"[factoryStateUpdate] Line progressed but was not marked as ready",
	)
	pl.claimWaitingPiece()
}
