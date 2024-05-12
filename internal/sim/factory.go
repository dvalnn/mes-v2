package sim

import (
	"context"
	"log"
	u "mes/internal/utils"
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
			log.Printf("[getFactoryInstance] Factory instance created")
		})
	}

	return factoryInstance, factoryMutex
}

type factory struct {
	processLines    map[string]*ProcessingLine
	stateUpdateFunc func(*factory) error
	supplyLines     []SupplyLine
	deliveryLines   []DeliveryLine
}

func InitFactory() *factory {
	processLines := make(map[string]*ProcessingLine)

	processLines[u.ID_L0] = &ProcessingLine{
		id:            u.ID_L0,
		conveyorLine:  make([]Conveyor, u.LINE_CONVEYOR_SIZE),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	for _, lineId := range []string{u.ID_L1, u.ID_L2, u.ID_L3, u.ID_L4, u.ID_L5, u.ID_L6} {
		processLines[lineId] = &ProcessingLine{
			id:            lineId,
			conveyorLine:  initType1Conveyor(),
			waitingPieces: []*freeLineWaiter{},
			readyForNext:  true,
		}
	}

	// TODO: Implement the stateUpdateFunc using OPCUA to communicate with the PLCs
	stateUpdateFunc := func(f *factory) error {
		for _, line := range f.processLines {
			if line.isReady() {
				line.claimWaitingPiece()
			}
			line.progressItems()
		}
		return nil
	}

	return &factory{
		processLines:    processLines,
		supplyLines:     []SupplyLine{},
		deliveryLines:   []DeliveryLine{},
		stateUpdateFunc: stateUpdateFunc,
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

	if piece.Location == u.ID_W2 {
		line := factory.processLines[u.ID_L0]
		line.registerWaitingPiece(waiter)
		return
	}

	nRegistered := 0
	for _, line := range factory.processLines {
		if line.id == u.ID_L0 {
			continue
		}

		if form := line.createBestForm(piece); form != nil {
			line.registerWaitingPiece(waiter)
			nRegistered++
		}
	}

	u.Assert(nRegistered > 0, "[registerWaitingPiece] No lines exist for piece")
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

	u.Assert(controlForm != nil, "[sendToProduction] controlForm is nil")
	factory.processLines[lineID].addItem(&conveyorItem{
		handler: &conveyorItemHandler{
			transformCh: transformCh,
			lineEntryCh: lineEntryCh,
			lineExitCh:  lineExitCh,
			errCh:       errCh,
		},
		controlID: controlForm.id,
		useM1:     controlForm.processTop,
		useM2:     controlForm.processBot,
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
	claimed := make(chan struct{})
	claimPieceCh := make(chan string)
	lock := &sync.Mutex{}
	waiter := &freeLineWaiter{
		claimed:      claimed,
		claimPieceCh: claimPieceCh,
		claimLock:    lock,
	}

	registerWaitingPiece(waiter, &piece)

	// Once a line is available, the check
	line, open := <-claimPieceCh
	close(claimed)
	lock.Unlock()

	u.Assert(open, "[sendToProduction] claimPieceCh closed before piece was claimed")
	log.Printf("[sendToProduction] Piece %v claimed by line %s", piece.ErpIdentifier, line)

	return sendToLine(line, &piece)
}

func updateFactoryState() {
	factory, mutex := getFactoryInstance()
	defer mutex.Unlock()

	err := factory.stateUpdateFunc(factory)
	u.Assert(err == nil, "[updateFactoryState] Error updating factory state")
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

func StartFactoryHandler(ctx context.Context) <-chan error {
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
				time.Sleep(250 * time.Millisecond)

				// 2 - update the line status for any line that progressed
				updateFactoryState()
			}
		}
	}()

	return errCh
}
