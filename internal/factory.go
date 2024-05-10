package mes

import (
	"context"
	"sync"
	"time"
)

var (
	factoryInstance *factory
	factoryMutex    = &sync.Mutex{}
	factoryOnce     sync.Once

	controlID      int16
	controlIDMutex = &sync.Mutex{}
)

func getFactoryInstance() (*factory, *sync.Mutex) {
	factoryMutex.Lock()

	if factoryInstance == nil {
		factoryOnce.Do(func() {
			factoryInstance = InitFactory()
		})
	}

	return factoryInstance, factoryMutex
}

type factory struct {
	processLines  map[string]*ProcessingLine
	supplyLines   []SupplyLine
	deliveryLines []DeliveryLine
}

func InitFactory() *factory {
	processLines := make(map[string]*ProcessingLine)

	processLines[ID_L0] = &ProcessingLine{
		id:            ID_L0,
		conveyorLine:  make([]Conveyor, LINE_CONVEYOR_SIZE),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	for _, lineId := range []string{ID_L1, ID_L2, ID_L3, ID_L4, ID_L5, ID_L6} {
		processLines[lineId] = &ProcessingLine{
			id:            lineId,
			conveyorLine:  initType1Conveyor(),
			waitingPieces: []*freeLineWaiter{},
			readyForNext:  true,
		}
	}

	return &factory{
		supplyLines:   []SupplyLine{},
		processLines:  processLines,
		deliveryLines: []DeliveryLine{},
	}
}

func getNextControlID() int16 {
	controlIDMutex.Lock()
	defer controlIDMutex.Unlock()

	controlID += 1
	return controlID
}

func registerWaitingPiece(waiter *freeLineWaiter, piece *Piece) {
	factory, mutex := getFactoryInstance()
	defer mutex.Unlock()

	if piece.Location == ID_W2 {
		line := factory.processLines[ID_L0]
		line.registerWaitingPiece(waiter)
		return
	}

	nRegistered := 0
	for _, line := range factory.processLines {
		if line.id == ID_L0 {
			continue
		}

		if form := line.createBestForm(piece); form != nil {
			line.registerWaitingPiece(waiter)
			nRegistered++
		}
	}
	assert(nRegistered > 0, "[registerWaitingPiece] No lines exist for piece")
}

type freeLineWaiter struct {
	checkIfAliveCh <-chan struct{}
	claimPieceCh   chan<- string
}

func sendToLine(lineID string, piece *Piece) *itemHandler {
	transformCh := make(chan string)
	lineEntryCh := make(chan string)
	lineExitCh := make(chan string)
	errCh := make(chan error)

	factory, mutex := getFactoryInstance()
	defer mutex.Unlock()
	controlForm := factory.processLines[lineID].createBestForm(piece)
	// TODO: controlForm.SendToPLC()
	assert(controlForm != nil, "[sendToProduction] controlForm is nil")
	factory.processLines[lineID].addItem(&conveyorItem{
		controlID: controlForm.id,
		handler: &conveyorItemHandler{
			transformCh: transformCh,
			lineEntryCh: lineEntryCh,
			lineExitCh:  lineExitCh,
			errCh:       errCh,
		},
	})

	return &itemHandler{
		transformCh: transformCh,
		lineEntryCh: lineEntryCh,
		lineExitCh:  lineExitCh,
		errCh:       errCh,
	}
}

func sendToProduction(
	piece Piece,
) *itemHandler {
	checkIfAliveCh := make(chan struct{})
	claimPieceCh := make(chan string)

	waiter := &freeLineWaiter{
		checkIfAliveCh: checkIfAliveCh,
		claimPieceCh:   claimPieceCh,
	}

	registerWaitingPiece(waiter, &piece)

	// Blocking wait for a free line
	checkIfAliveCh <- struct{}{}
	close(checkIfAliveCh)

	// Once a line is available, the check
	line := <-claimPieceCh

	return sendToLine(line, &piece)
}

func updateFactoryState() {
	_, mutex := getFactoryInstance()
	defer mutex.Unlock()

	// TODO: update the factory state based on the plc state variables
}

func progressFreeLines() {
	factory, mutex := getFactoryInstance()
	defer mutex.Unlock()
	for _, line := range factory.processLines {
		if line.isReady() {
			// TODO: progress the line until the id of the item that leaves
			// matches the last item left reported by the plc
			line.progressItems()
		}
	}
}

func startFactoryHandler(ctx context.Context) <-chan error {
	errCh := make(chan error)

	// Connect to the factory floor plcs
	time.Sleep(500 * time.Millisecond)

	// Start the factory floor
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			default:
				// 1 - Get a full update of the factory floor
				time.Sleep(1000 * time.Millisecond)

				// 2 - update the line status for any line that progressed
			}
		}
	}()

	return errCh
}
