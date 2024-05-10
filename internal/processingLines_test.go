package mes

import (
	"context"
	"testing"
	"time"
)

func TestCreateBestFormForOnlyM1(t *testing.T) {
	pLine := ProcessingLine{
		id:            ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForM1 := Piece{
		Steps: []Transformation{{Tool: TOOL_1, MaterialKind: P_KIND_1}},
	}

	best := pLine.createBestForm(&minimalPieceForM1)
	expected := &processControlForm{
		toolTop:    TOOL_1,
		toolBot:    TOOL_1,
		pieceKind:  P_KIND_1,
		processTop: true,
		processBot: false,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestCreateBestFormForOnlyM2(t *testing.T) {
	pLine := ProcessingLine{
		id:            ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForM2 := Piece{
		Steps: []Transformation{{Tool: TOOL_4, MaterialKind: P_KIND_1}},
	}

	best := pLine.createBestForm(&minimalPieceForM2)
	expected := &processControlForm{
		toolTop:    TOOL_4,
		toolBot:    TOOL_4,
		pieceKind:  P_KIND_1,
		processTop: false,
		processBot: true,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestCreateBestFormWithChain(t *testing.T) {
	pLine := ProcessingLine{
		id:            ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForChain := Piece{
		Steps: []Transformation{{Tool: TOOL_1, MaterialKind: P_KIND_1}, {Tool: TOOL_4}},
	}

	best := pLine.createBestForm(&minimalPieceForChain)
	expected := &processControlForm{
		toolTop:    TOOL_1,
		toolBot:    TOOL_4,
		pieceKind:  P_KIND_1,
		processTop: true,
		processBot: true,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestCreateBestFormForOnlyM2WithLeftoverSteps(t *testing.T) {
	pLine := ProcessingLine{
		id:            ID_L1,
		conveyorLine:  initType1Conveyor(),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	minimalPieceForM2WithExtra := Piece{
		Steps: []Transformation{{Tool: TOOL_4, MaterialKind: P_KIND_1}, {Tool: TOOL_1}},
	}

	best := pLine.createBestForm(&minimalPieceForM2WithExtra)
	expected := &processControlForm{
		toolTop:    TOOL_4,
		toolBot:    TOOL_4,
		pieceKind:  P_KIND_1,
		processTop: false,
		processBot: true,
	}

	if *best != *expected {
		t.Fatalf("Expected %+v, got %+v", expected, best)
	}
}

func TestLineAddItem(t *testing.T) {
	pLine := ProcessingLine{
		id:            ID_L1,
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
		t.Logf("AddItem assertion panicked as expected")
	}()

	pLine.addItem(conveyorItem) // should panic
}

func TestProgressItems(t *testing.T) {
	pLine := ProcessingLine{
		id:            ID_L1,
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
			if lineID != ID_L1 {
				t.Fatalf("Expected to receive line ID %s, got %s", ID_L1, lineID)
			}

		case <-ctx.Done():
			t.Fatal("Expected to receive on channel, but did not")
		}
	}

	pLine.addItem(conveyorItem)
	go pLine.progressItems()
	receiveOnChannel(lineEntryCh)
	if pLine.conveyorLine[0].item != nil || pLine.conveyorLine[1].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to second position as expected")

	go pLine.progressItems()
	receiveOnChannel(transformCh)
	if pLine.conveyorLine[1].item != nil || pLine.conveyorLine[2].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to third position as expected")

	pLine.progressItems()
	if pLine.conveyorLine[2].item != nil || pLine.conveyorLine[3].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to fourth position as expected")

	go pLine.progressItems()
	receiveOnChannel(transformCh)
	if pLine.conveyorLine[3].item != nil || pLine.conveyorLine[4].item != conveyorItem {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
	t.Log("Item progressed to fifth position as expected")

	go pLine.progressItems()
	receiveOnChannel(lineExitCh)
	if pLine.conveyorLine[4].item != nil {
		t.Fatalf("Expected conveyorItem to be moved to next conveyor slot, but it was not")
	}
	if !pLine.readyForNext {
		t.Fatalf("Expected line to be ready for next item, but it was not")
	}
}