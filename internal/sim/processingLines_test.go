package sim

import (
	"context"
	plc "mes/internal/net/plc"
	u "mes/internal/utils"
	"testing"
	"time"
)

func TestCreateBestFormForOnlyM1(t *testing.T) {
	pLine := ProcessingLine{
		id:            u.ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForM1 := Piece{
		Kind:  u.P_KIND_1,
		Steps: []Transformation{{Tool: u.TOOL_1}},
	}

	best := pLine.createBestForm(&minimalPieceForM1)
	expected := &processControlForm{
		toolTop:    u.TOOL_1,
		toolBot:    u.TOOL_1,
		pieceKind:  u.P_KIND_1,
		processTop: true,
		processBot: false,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestCreateBestFormForOnlyM2(t *testing.T) {
	pLine := ProcessingLine{
		id:            u.ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForM2 := Piece{
		Kind:  u.P_KIND_1,
		Steps: []Transformation{{Tool: u.TOOL_4}},
	}

	best := pLine.createBestForm(&minimalPieceForM2)
	expected := &processControlForm{
		toolTop:    u.TOOL_4,
		toolBot:    u.TOOL_4,
		pieceKind:  u.P_KIND_1,
		processTop: false,
		processBot: true,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestCreateBestFormWithChain(t *testing.T) {
	pLine := ProcessingLine{
		id:            u.ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForChain := Piece{
		Kind:  u.P_KIND_1,
		Steps: []Transformation{{Tool: u.TOOL_1}, {Tool: u.TOOL_4}},
	}

	best := pLine.createBestForm(&minimalPieceForChain)
	expected := &processControlForm{
		toolTop:    u.TOOL_1,
		toolBot:    u.TOOL_4,
		pieceKind:  u.P_KIND_1,
		processTop: true,
		processBot: true,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestCreateBestFormForOnlyM2WithLeftoverSteps(t *testing.T) {
	pLine := ProcessingLine{
		id:            u.ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForM2WithExtra := Piece{
		Kind:  u.P_KIND_1,
		Steps: []Transformation{{Tool: u.TOOL_4}, {Tool: u.TOOL_1}},
	}

	best := pLine.createBestForm(&minimalPieceForM2WithExtra)
	expected := &processControlForm{
		toolTop:    u.TOOL_4,
		toolBot:    u.TOOL_4,
		pieceKind:  u.P_KIND_1,
		processTop: false,
		processBot: true,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestLineAddItem(t *testing.T) {
	pLine := ProcessingLine{
		id:            u.ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}
	conveyorItem := &conveyorItem{
		handler: &conveyorItemHandler{
			transformCh: make(chan<- string),
			lineEntryCh: make(chan<- string),
			lineExitCh:  make(chan<- string),
			errCh:       make(chan<- error),
		},
		controlID: 0,
	}

	pLine.addItem(conveyorItem) // should not panic
	if pLine.conveyorLine[0].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be added to line, but it was not")
	}
	t.Log("Item added to line as expected")

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("AddItem should panic when adding a piece to a line that is not ready for next")
		}
		t.Logf("AddItem utils.Assertion panicked as expected")
	}()

	pLine.addItem(conveyorItem) // should panic
}

func TestProgressItemsSingleItem(t *testing.T) {
	pLine := ProcessingLine{
		id:            u.ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	transformCh := make(chan string)
	lineEntryCh := make(chan string)
	lineExitCh := make(chan string)

	conveyorItem := &conveyorItem{
		handler: &conveyorItemHandler{
			transformCh: transformCh,
			lineEntryCh: lineEntryCh,
			lineExitCh:  lineExitCh,
			errCh:       make(chan<- error),
		},
		controlID: 0,
		useM1:     true,
		useM2:     true,
	}

	receiveOnChannel := func(ch <-chan string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		select {
		case lineID := <-ch:
			cancel()
			if lineID != u.ID_L1 {
				t.Fatalf("Expected to receive line ID %s, got %s", u.ID_L1, lineID)
			}

		case <-ctx.Done():
			t.Fatal("Expected to receive on channel, but did not")
		}
	}

	pLine.addItem(conveyorItem)
	go pLine.progressConveyor()
	receiveOnChannel(lineEntryCh)
	if pLine.conveyorLine[0].item != nil || pLine.conveyorLine[1].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to second position as expected")

	go pLine.progressConveyor()
	receiveOnChannel(transformCh)
	if pLine.conveyorLine[1].item != nil || pLine.conveyorLine[2].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to third position as expected")

	pLine.progressConveyor()
	if pLine.conveyorLine[2].item != nil || pLine.conveyorLine[3].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to fourth position as expected")

	go pLine.progressConveyor()
	receiveOnChannel(transformCh)
	if pLine.conveyorLine[3].item != nil || pLine.conveyorLine[4].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to fifth position as expected")

	go pLine.progressConveyor()
	receiveOnChannel(lineExitCh)
	if pLine.conveyorLine[4].item != nil {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
}

func TestProgressItemsFullLine(t *testing.T) {
	pLine := ProcessingLine{
		id:            u.ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	transformChM1 := make(chan string)
	transformChM2 := make(chan string)
	lineEntryCh := make(chan string)
	lineExitCh := make(chan string)

	conveyorItemEntry := &conveyorItem{
		handler: &conveyorItemHandler{lineEntryCh: lineEntryCh},
	}

	conveyorItemExit := &conveyorItem{
		handler:   &conveyorItemHandler{lineExitCh: lineExitCh},
		controlID: 1,
	}

	conveyorItemM1 := &conveyorItem{
		handler: &conveyorItemHandler{transformCh: transformChM1},
		useM1:   true,
	}

	conveyorItemM2 := &conveyorItem{
		handler: &conveyorItemHandler{transformCh: transformChM2},
		useM2:   true,
	}

	pLine.conveyorLine[0].item = conveyorItemEntry
	pLine.conveyorLine[1].item = conveyorItemM1
	pLine.conveyorLine[2].item = nil
	pLine.conveyorLine[3].item = conveyorItemM2
	pLine.conveyorLine[4].item = conveyorItemExit

	receiveOnChannel := func(ch <-chan string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		select {
		case lineID := <-ch:
			cancel()
			if lineID != u.ID_L1 {
				t.Fatalf("Expected to receive line ID %s, got %s", u.ID_L1, lineID)
			}

		case <-ctx.Done():
			t.Fatal("Expected to receive on channel, but did not")
		}
	}

	go pLine.progressConveyor()
	receiveOnChannel(lineEntryCh)
	t.Log("Entry channel triggered as expected")
	receiveOnChannel(transformChM1)
	t.Log("M1 item channel triggered as expected")
	receiveOnChannel(transformChM2)
	t.Log("M2 item channel triggered as expected")
	receiveOnChannel(lineExitCh)
	t.Log("Exit channel triggered as expected")
}

func TestPruneDeadWaiters(t *testing.T) {
	deadWaitCh1 := make(chan struct{})
	deadWaitCh2 := make(chan struct{})
	deadWaitCh3 := make(chan struct{})
	close(deadWaitCh1)
	close(deadWaitCh2)
	close(deadWaitCh3)

	aliveWaitCh1 := make(chan struct{})

	pLine := ProcessingLine{
		id:           u.ID_L1,
		conveyorLine: initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{
			{pieceClaimedCh: deadWaitCh1},
			{pieceClaimedCh: deadWaitCh2},
			{pieceClaimedCh: aliveWaitCh1},
			{pieceClaimedCh: deadWaitCh3},
		},
	}

	pLine.pruneDeadWaiters()
	if len(pLine.waitingPieces) != 1 {
		t.Fatalf("Expected 1 waiter to be alive, got %d", len(pLine.waitingPieces))
	}
}

func TestTransformCellCommand(t *testing.T) {
	// creates a dummy process control form
	pcf := &processControlForm{
		toolTop:    u.TOOL_1,
		toolBot:    u.TOOL_1,
		pieceKind:  u.P_KIND_1,
		processTop: true,
		processBot: false,
		id:         1,
	}

	result := pcf.toCellCommand()
	expected := &plc.CellCommand{
		TxId:       plc.OpcuaInt16{Value: 1},
		PieceKind:  plc.OpcuaInt16{Value: 1},
		ProcessBot: plc.OpcuaBool{Value: false},
		ProcessTop: plc.OpcuaBool{Value: true},
		ToolBot:    plc.OpcuaInt16{Value: 1},
		ToolTop:    plc.OpcuaInt16{Value: 1},
	}

	if *result != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, result)
	}
}
