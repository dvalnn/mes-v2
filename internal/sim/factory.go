package sim

import (
	"context"
	"log"
	"mes/internal/net/plc"
	"mes/internal/utils"
	"sync"
	"time"
)

// TODO: add delivery lines
type factory struct {
	processLines    map[string]*ProcessingLine
	stateUpdateFunc func(*factory, context.Context) error
	plcClient       *plc.Client
	supplyLines     []*plc.SupplyLine
	warehouses      []*plc.Warehouse
}

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

// TODO: Update supply line state when missing fields are added
func factoryStateUpdate(f *factory, ctx context.Context) error {
	for _, warehouse := range f.warehouses {
		func() {
			readCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			f.plcClient.Read(warehouse.OpcuaVars(), readCtx)
		}()
	}

	for _, line := range f.processLines {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		func() {
			readCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			f.plcClient.Read(line.plc.StateOpcuaVars(), readCtx)
		}()

		if !line.plc.Progressed() {
			continue
		}

		line.ProgressInternalState()
	}

	return nil
}

func mockFactoryStateUpdate(f *factory, _ context.Context) error {
	for _, line := range f.processLines {
		if line.isReady() {
			line.claimWaitingPiece()
		}
		line.progressConveyor()
	}

	return nil
}

func InitFactory() *factory {
	processLines := make(map[string]*ProcessingLine)
	linePlcs := plc.InitCells()

	processLines[utils.ID_L0] = &ProcessingLine{
		plc:           linePlcs[0],
		id:            utils.ID_L0,
		conveyorLine:  make([]Conveyor, utils.LINE_CONVEYOR_SIZE),
		waitingPieces: []*freeLineWaiter{},
		readyForNext:  true,
	}

	for idx, lineId := range []string{utils.ID_L1, utils.ID_L2, utils.ID_L3} {
		processLines[lineId] = &ProcessingLine{
			plc:           linePlcs[idx+1],
			id:            lineId,
			conveyorLine:  initType1Conveyor(),
			waitingPieces: []*freeLineWaiter{},
			readyForNext:  true,
		}
	}

	for idx, lineId := range []string{utils.ID_L4, utils.ID_L5, utils.ID_L6} {
		processLines[lineId] = &ProcessingLine{
			plc:           linePlcs[idx+4],
			id:            lineId,
			conveyorLine:  initType2Conveyor(),
			waitingPieces: []*freeLineWaiter{},
			readyForNext:  true,
		}
	}

	stateUpdateFunc := factoryStateUpdate
	return &factory{
		processLines:    processLines,
		stateUpdateFunc: stateUpdateFunc,
		plcClient:       plc.NewClient(plc.OPCUA_ENDPOINT),
		supplyLines:     plc.InitSupplyLines(),
		warehouses:      plc.InitWarehouses(),
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

	if piece.Location == utils.ID_W2 {
		line := factory.processLines[utils.ID_L0]
		line.registerWaitingPiece(waiter)
		return
	}

	nRegistered := 0
	for _, line := range factory.processLines {
		if line.id == utils.ID_L0 {
			continue
		}

		// The control ID does not matter in this case
		if form := line.createBestForm(piece, 0); form != nil {
			line.registerWaitingPiece(waiter)
			nRegistered++
		}
	}

	utils.Assert(nRegistered > 0, "[registerWaitingPiece] No lines exist for piece")
}

func sendToLine(lineID string, piece *Piece) *itemHandler {
	transformCh := make(chan string)
	lineEntryCh := make(chan string)
	lineExitCh := make(chan string)
	errCh := make(chan error)

	factory, mutex := getFactoryInstance()
	defer mutex.Unlock()

	// TODO: tweak the timeout value
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	newTxId := factory.processLines[lineID].plc.LastCommandTxId() + 1
	controlForm := factory.processLines[lineID].createBestForm(piece, newTxId)
	factory.plcClient.Write(controlForm.toCellCommand().OpcuaVars(), ctx)

	utils.Assert(controlForm != nil, "[sendToProduction] controlForm is nil")
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

	utils.Assert(open, "[sendToProduction] claimPieceCh closed before piece was claimed")
	log.Printf("[sendToProduction] Piece %v claimed by line %s", piece.ErpIdentifier, line)

	return sendToLine(line, &piece)
}

func runFactoryStateUpdateFunc(ctx context.Context) {
	factory, mutex := getFactoryInstance()
	defer mutex.Unlock()

	// TODO: tweak the timeout value
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	err := factory.stateUpdateFunc(factory, ctx)
	utils.Assert(err == nil, "[updateFactoryState] Error updating factory state")
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
				runFactoryStateUpdateFunc(ctx)
			}
		}
	}()

	return errCh
}
