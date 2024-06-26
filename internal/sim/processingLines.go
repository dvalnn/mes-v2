package sim

import (
	plc "mes/internal/net/plc"
	u "mes/internal/utils"
	"strconv"
	"sync"
)

type Machine struct {
	name         string
	selectedTool string
	tools        []string
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
	m1Repeats int16
	m2Repeats int16
	useM1     bool
	useM2     bool

	// Metadata for the erp
	toolChangeM1 bool
	toolChangeM2 bool
}

type Conveyor struct {
	item    *conveyorItem
	machine *Machine
}

func initType1Conveyor() []Conveyor {
	conveyor := make([]Conveyor, LINE_CONVEYOR_SIZE)

	conveyor[LINE_DEFAULT_M1_POS] = Conveyor{
		item: nil,
		machine: &Machine{
			name:         "M1",
			selectedTool: u.TOOL_1,
			tools:        []string{u.TOOL_1, u.TOOL_2, u.TOOL_3},
		},
	}

	conveyor[LINE_DEFAULT_M2_POS] = Conveyor{
		item: nil,
		machine: &Machine{
			name:         "M2",
			selectedTool: u.TOOL_1,
			tools:        []string{u.TOOL_1, u.TOOL_2, u.TOOL_3},
		},
	}

	return conveyor
}

func initType2Conveyor() []Conveyor {
	conveyor := make([]Conveyor, LINE_CONVEYOR_SIZE)

	conveyor[LINE_DEFAULT_M1_POS] = Conveyor{
		item: nil,
		machine: &Machine{
			name:         "M3",
			selectedTool: u.TOOL_1,
			tools:        []string{u.TOOL_1, u.TOOL_4, u.TOOL_5},
		},
	}

	conveyor[LINE_DEFAULT_M2_POS] = Conveyor{
		item: nil,
		machine: &Machine{
			name:         "M4",
			selectedTool: u.TOOL_1,
			tools:        []string{u.TOOL_1, u.TOOL_4, u.TOOL_6},
		},
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
	// Fields to control the plc
	toolTop    string
	toolBot    string
	pieceKind  string
	id         int16
	repeatTop  int16
	repeatBot  int16
	processTop bool
	processBot bool

	// Metadata for decision making in the MES simulation

	// Number of steps completed for this command (0-based) out of the total
	// steps in the piece's recipe
	stepsCompleted int
	totalSteps     int
	// Time needed to process this command to completion (in seconds)
	// assuming no delays (e.g. waiting for a machine to be free)
	intrinsicTime int
	// Time needed to process this command to completion (in seconds)
	// taking into account possible delays in queue
	queueSize int

	// Whether or not a tool change is needed
	changeM1 bool
	changeM2 bool
}

func (pcf *processControlForm) metadataScore() int {
	return pcf.intrinsicTime*TIME_WEIGHT +
		pcf.queueSize*QUEUE_WEIGHT +
		(pcf.totalSteps-pcf.stepsCompleted)*STEP_WEIGHT
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
		TxId:      plc.OpcuaInt16{Value: pcf.id},
		PieceKind: plc.OpcuaInt16{Value: PieceStrToInt(pcf.pieceKind)},

		ProcessTop: plc.OpcuaBool{Value: pcf.processTop},
		ToolTop:    plc.OpcuaInt16{Value: ToolStrToInt(pcf.toolTop)},
		RepeatTop:  plc.OpcuaInt16{Value: pcf.repeatTop},

		ProcessBot: plc.OpcuaBool{Value: pcf.processBot},
		ToolBot:    plc.OpcuaInt16{Value: ToolStrToInt(pcf.toolBot)},
		RepeatBot:  plc.OpcuaInt16{Value: pcf.repeatBot},
	}
}

type freeLineWaiter struct {
	pieceClaimedCh <-chan struct{}
	claimPieceCh   chan<- string
	claimLock      *sync.Mutex
	claimCountLock *sync.Mutex
	claimCount     int
}

func (flw *freeLineWaiter) incrementClaimCount() {
	flw.claimCountLock.Lock()
	flw.claimCount++
	flw.claimCountLock.Unlock()
}

func (flw *freeLineWaiter) decrementClaimCount() {
	flw.claimCountLock.Lock()
	flw.claimCount--
	flw.claimCountLock.Unlock()

	u.Assert(flw.claimCount >= 0,
		"[freeLineWaiter.decrementClaimCount] claimCount is not positive")
}

func (pl *ProcessingLine) registerWaitingPiece(w *freeLineWaiter) {
	pl.waitingPieces = append(pl.waitingPieces, w)
	w.incrementClaimCount()
}

func (pl *ProcessingLine) pruneDeadWaiters() {
	aliveWaiters := make([]*freeLineWaiter, 0, len(pl.waitingPieces))
	for _, w := range pl.waitingPieces {
		select {
		case <-w.pieceClaimedCh:
			w.decrementClaimCount()
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
		case <-w.pieceClaimedCh:
			w.claimLock.Unlock()
		default:
			w.claimPieceCh <- pl.id
			close(w.claimPieceCh)
			// HACK:
			// not unlocking here on purpose, so that the piece handler
			// can unlock it after the pieceClaimedCh is closed, to avoid
			// double closing or closing while another goroutine is trying to
			// claim the same piece
			break loop
		}
	}
}

func (pl *ProcessingLine) isMachineCompatibleWith(mIndex int, t Transformation) bool {
	m := pl.conveyorLine[mIndex].machine
	u.Assert(m != nil, "[ProcessingLine.isMachineCompatibleWith] machine is null")

	for _, tool := range m.tools {
		if tool == t.Tool {
			return true
		}
	}
	return false
}

func (pl *ProcessingLine) currentTool(mIndex int) string {
	m := pl.conveyorLine[mIndex].machine
	u.Assert(m != nil, "[ProcessingLine.currentTool] machine is null")
	return m.selectedTool
}

func (pl *ProcessingLine) setCurrentTool(mIndex int, tool string) {
	// Line 0 has no machines
	if pl.id == u.ID_L0 {
		return
	}

	if tool == "" {
		return // No tool change
	}

	m := pl.conveyorLine[mIndex].machine
	u.Assert(m != nil, "[ProcessingLine.currentTool] machine is null")
	for _, t := range m.tools {
		if t == tool {
			m.selectedTool = tool
			return
		}
	}
	panic("[ProcessingLine.setCurrentTool] Tool not found in machine")
}

func (pl *ProcessingLine) getNItemsInConveyor() int {
	nItemsInQueue := 0
	for _, conveyor := range pl.conveyorLine {
		if conveyor.item != nil {
			nItemsInQueue++
		}
	}
	return nItemsInQueue
}

func (pl *ProcessingLine) createBestForm(piece *Piece) *processControlForm {
	if pl.id == u.ID_L0 {
		return &processControlForm{
			pieceKind: piece.Kind,
			id:        piece.ControlID,
		}
	}

	currentStepIdx := piece.CurrentStep
	currentStep := piece.Steps[currentStepIdx]

	topCompatible := pl.isMachineCompatibleWith(LINE_DEFAULT_M1_POS, currentStep)
	botCompatible := pl.isMachineCompatibleWith(LINE_DEFAULT_M2_POS, currentStep)

	if !topCompatible && !botCompatible {
		return nil
	}

	if !topCompatible {
		return pl.createBotOnlyForm(piece, currentStepIdx, currentStep)
	}

	topFormCandidate := pl.createTopFormWithPossibleBot(piece, currentStepIdx, currentStep)
	if !botCompatible {
		return topFormCandidate
	}

	botFormCandidate := pl.createBotOnlyForm(piece, currentStepIdx, currentStep)

	if topFormCandidate.metadataScore() < botFormCandidate.metadataScore() {
		return topFormCandidate
	}
	return botFormCandidate
}

func (pl *ProcessingLine) createBotOnlyForm(
	piece *Piece, currentStepIdx int, currentStep Transformation,
) *processControlForm {
	repeatBot := int16(1)
	stepsCompleted := 1
	intrinsicTime := currentStep.Time

	changeM2 := false
	if currentStep.Tool != pl.currentTool(LINE_DEFAULT_M2_POS) {
		intrinsicTime += MACHINE_TOOL_SWAP_TIME
		changeM2 = true
	}

	for currentStepIdx+1 < len(piece.Steps) {
		nextStep := piece.Steps[currentStepIdx+1]
		if currentStep.Tool != nextStep.Tool {
			break
		}
		repeatBot++
		stepsCompleted++
		currentStepIdx++
		intrinsicTime += nextStep.Time
		currentStep = nextStep
	}

	return &processControlForm{
		id:        piece.ControlID,
		pieceKind: piece.Kind,

		processTop: false,
		toolTop:    "",
		repeatTop:  1,

		toolBot:    currentStep.Tool,
		repeatBot:  repeatBot,
		processBot: true,

		stepsCompleted: stepsCompleted,
		totalSteps:     len(piece.Steps),
		intrinsicTime:  intrinsicTime,
		queueSize:      pl.getNItemsInConveyor(),
		changeM1:       false,
		changeM2:       changeM2,
	}
}

func (pl *ProcessingLine) createTopFormWithPossibleBot(
	piece *Piece,
	currentStepIdx int,
	currentStep Transformation,
) *processControlForm {
	repeatTop := int16(1)
	stepsCompleted := 1
	intrinsicTime := currentStep.Time

	changeM1 := false
	if currentStep.Tool != pl.currentTool(LINE_DEFAULT_M1_POS) {
		intrinsicTime += MACHINE_TOOL_SWAP_TIME
		changeM1 = true
	}

	for currentStepIdx+1 < len(piece.Steps) {
		nextStep := piece.Steps[currentStepIdx+1]
		if currentStep.Tool != nextStep.Tool {
			break
		}
		repeatTop++
		stepsCompleted++
		currentStepIdx++
		intrinsicTime += nextStep.Time
		currentStep = nextStep
	}

	toolBot, repeatBot, changeM2 := pl.determineToolBot(
		piece, currentStepIdx, &intrinsicTime, &stepsCompleted,
	)

	return &processControlForm{
		id:        piece.ControlID,
		pieceKind: piece.Kind,

		toolTop:    currentStep.Tool,
		processTop: true,
		repeatTop:  repeatTop,

		toolBot:    toolBot,
		processBot: toolBot != "",
		repeatBot:  repeatBot,

		stepsCompleted: stepsCompleted,
		totalSteps:     len(piece.Steps),
		intrinsicTime:  intrinsicTime,
		queueSize:      pl.getNItemsInConveyor(),
		changeM1:       changeM1,
		changeM2:       changeM2,
	}
}

func (pl *ProcessingLine) determineToolBot(piece *Piece,
	currentStepIdx int,
	intrinsicTime *int,
	stepsCompleted *int,
) (string, int16, bool) {
	if currentStepIdx+1 >= len(piece.Steps) {
		return "", 0, false
	}
	currentStep := piece.Steps[currentStepIdx+1]
	if !pl.isMachineCompatibleWith(LINE_DEFAULT_M2_POS, currentStep) {
		return "", 0, false
	}
	currentStepIdx++
	*stepsCompleted++
	*intrinsicTime += currentStep.Time

	repeatBot := int16(1)

	for currentStepIdx+1 < len(piece.Steps) {
		nextStep := piece.Steps[currentStepIdx+1]
		if currentStep.Tool != nextStep.Tool {
			break
		}
		*stepsCompleted++
		*intrinsicTime += nextStep.Time
		repeatBot++
		currentStepIdx++
		currentStep = nextStep
	}

	changeM2 := false
	if currentStep.Tool != pl.currentTool(LINE_DEFAULT_M2_POS) {
		*intrinsicTime += MACHINE_TOOL_SWAP_TIME
		changeM2 = true
	}
	return currentStep.Tool, repeatBot, changeM2
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
		for i := int16(0); i < m1Item.m1Repeats; i++ {
			m1Item.handler.transformCh <- pl.id + "," + m1.name + "," +
				strconv.FormatBool(m1Item.toolChangeM1)
		}
	}

	m2 := pl.conveyorLine[LINE_DEFAULT_M2_POS].machine
	m2Item := pl.conveyorLine[LINE_DEFAULT_M2_POS].item
	if m2 != nil && m2Item != nil && m2Item.useM2 {
		for i := int16(0); i < m2Item.m2Repeats; i++ {
			m2Item.handler.transformCh <- pl.id + "," + m2.name + "," +
				strconv.FormatBool(m2Item.toolChangeM2)
		}
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
