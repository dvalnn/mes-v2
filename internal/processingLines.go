package mes

type SupplyLine struct{}

type DeliveryLine struct{}

type Machine struct {
	name  string
	tools []string
}

// For the inner logic
type conveyorItemHandler struct {
	transformCh chan<- string
	lineEntryCh chan<- string
	lineExitCh  chan<- string

	errCh chan<- error
}

// To return to the caller
type itemHandler struct {
	transformCh <-chan string
	lineEntryCh <-chan string
	lineExitCh  <-chan string

	errCh <-chan error
}

type ConveyorItem struct {
	handler   *conveyorItemHandler
	controlID int16
}

type Conveyor struct {
	item    *ConveyorItem
	machine *Machine
}

func initType1Conveyor() []Conveyor {
	conveyor := make([]Conveyor, LINE_CONVEYOR_SIZE)

	conveyor[LINE_DEFAULT_M1_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M1", tools: []string{TOOL_1, TOOL_2, TOOL_3}},
	}

	conveyor[LINE_DEFAULT_M2_POS] = Conveyor{
		item:    nil,
		machine: &Machine{name: "M2", tools: []string{TOOL_4, TOOL_5, TOOL_6}},
	}

	return conveyor
}

type ProcessingLine struct {
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

func (pl *ProcessingLine) isReady() bool {
	return pl.readyForNext
}

func (pl *ProcessingLine) registerWaitingPiece(w *freeLineWaiter) {
	pl.waitingPieces = append(pl.waitingPieces, w)
}

func (pl *ProcessingLine) pruneDeadWaiters() {
	aliveWaiters := make([]*freeLineWaiter, 0, len(pl.waitingPieces))
	for _, w := range pl.waitingPieces {
		select {
		case _, ok := <-w.checkIfAliveCh:
			if ok {
				aliveWaiters = append(aliveWaiters, w)
			}
		default:
			// Do nothing
		}
	}
	pl.waitingPieces = aliveWaiters
}

func (pl *ProcessingLine) claimPiece() {
	assert(pl.isReady(), "[ProcessingLine.claimPiece] Processing line is not ready")
	pl.pruneDeadWaiters()
	for _, w := range pl.waitingPieces {
		_, ok := <-w.checkIfAliveCh
		if !ok {
			continue
		}
		w.claimPieceCh <- pl.id
		break
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

func (pl *ProcessingLine) createBestForm(piece *Piece) *processControlForm {
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
			pieceKind:  currentStep.MaterialKind,
			processTop: true,
			processBot: chainSteps,
		}
	}

	return &processControlForm{
		toolTop:    currentStep.Tool, // doesn't matter as it's not used
		toolBot:    currentStep.Tool,
		pieceKind:  currentStep.MaterialKind,
		processTop: false,
		processBot: true,
	}
}

func (pl *ProcessingLine) addItem(item *ConveyorItem) {
	assert(pl.isReady(), "[ProcessingLine.addItem] Processing line is not ready")
	assert(pl.conveyorLine[0].item == nil, "[ProcessingLine.addItem] Conveyor is not empty")

	pl.readyForNext = false
	pl.conveyorLine[0].item = item
}

func (pl *ProcessingLine) progressItems() int16 {
	outItem := pl.conveyorLine[LINE_CONVEYOR_SIZE-1].item
	if outItem != nil {
		outItem.handler.lineExitCh <- pl.id
	}

	inItem := pl.conveyorLine[0].item
	if inItem != nil {
		inItem.handler.lineEntryCh <- pl.id
	}

	m1 := pl.conveyorLine[LINE_DEFAULT_M1_POS].machine
	m1Item := pl.conveyorLine[LINE_DEFAULT_M1_POS].item
	if m1 != nil && m1Item != nil {
		m1Item.handler.transformCh <- pl.id
	}

	m2 := pl.conveyorLine[LINE_DEFAULT_M2_POS].machine
	m2Item := pl.conveyorLine[LINE_DEFAULT_M2_POS].item
	if m2 != nil && m2Item != nil {
		m2Item.handler.transformCh <- pl.id
	}

	// Move items along the conveyor line
	for i := LINE_CONVEYOR_SIZE - 1; i > 0; i-- {
		pl.conveyorLine[i].item = pl.conveyorLine[i-1].item
	}
	pl.conveyorLine[0].item = nil
	pl.readyForNext = true

	return outItem.controlID
}
