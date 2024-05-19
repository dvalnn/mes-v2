package sim

import (
	"context"
	"log"
	plc "mes/internal/net/plc"
	"mes/internal/utils"
	"sync"
	"time"
)

// TODO: add delivery lines
type factory struct {
	processLines    map[string]*ProcessingLine
	stateUpdateFunc func(context.Context, *factory) error
	plcClient       *plc.Client
	supplyLines     []*plc.SupplyLine
	deliveryLines   []*plc.DeliveryLine
	warehouses      []*plc.Warehouse
}

var (
	factoryInstance *factory
	factoryMutex    = &sync.Mutex{}
	factoryOnce     sync.Once
)

func getFactoryInstance() (*factory, *sync.Mutex) {
	factoryMutex.Lock()

	if factoryInstance == nil {
		factoryOnce.Do(func() {
			factoryInstance = InitFactory(factoryStateUpdate)
			log.Printf("[getFactoryInstance] Factory instance created")
		})
	}

	return factoryInstance, factoryMutex
}

// TODO: Update supply line state when missing fields are added
func factoryStateUpdate(ctx context.Context, f *factory) error {
	for _, warehouse := range f.warehouses {
		func() {
			readCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			_, err := f.plcClient.Read(warehouse.OpcuaVars(), readCtx)
			utils.Assert(err == nil, "[factoryStateUpdate] Error reading warehouse")
		}()
	}

	for _, supplyLine := range f.supplyLines {
		func() {
			readCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			readResponse, err := f.plcClient.Read(supplyLine.StateOpcuaVars(), readCtx)
			supplyLine.UpdateState(readResponse)
			utils.Assert(err == nil, "[factoryStateUpdate] Error reading supply lines")
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
			readResponse, err := f.plcClient.Read(line.plc.StateOpcuaVars(), readCtx)
			utils.Assert(err == nil, "[factoryStateUpdate] Error reading line state")
			line.plc.UpdateState(readResponse)
		}()

		line.UpdateConveyor()
	}

	return nil
}

func mockFactoryStateUpdate(f *factory, _ context.Context) error {
	for _, line := range f.processLines {
		if line.readyForNext {
			line.claimWaitingPiece()
		}
		line.ProgressNewPiece()
		line.progressConveyor()
	}

	return nil
}

func InitFactory(
	stateUpdateFunc func(context.Context, *factory) error,
) *factory {
	processLines := make(map[string]*ProcessingLine)
	linePlcs := plc.InitCells()

	processLines[utils.ID_L0] = &ProcessingLine{
		plc:           linePlcs[0],
		id:            utils.ID_L0,
		conveyorLine:  make([]Conveyor, LINE_CONVEYOR_SIZE),
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

	return &factory{
		processLines:    processLines,
		stateUpdateFunc: factoryStateUpdate,
		plcClient:       plc.NewClient(plc.OPCUA_ENDPOINT),
		supplyLines:     plc.InitSupplyLines(),
		deliveryLines:   plc.InitDeliveryLines(),
		warehouses:      plc.InitWarehouses(),
	}
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
	lineOffers := make(map[string]int)
	for _, line := range factory.processLines {
		if line.id == utils.ID_L0 {
			continue
		}

		// The control ID does not matter in this case
		if form := line.createBestForm(piece, 0); form != nil {
			score := form.metadataScore()
			lineOffers[line.id] = score
		}
	}

	bestScore := 9999999
	for _, score := range lineOffers {
		if score < bestScore {
			bestScore = score
		}
	}

	leniency := 0.2 // 10% leniency
	for lineId, score := range lineOffers {
		if (1-leniency)*float64(score) <= float64(bestScore) {
			factory.processLines[lineId].registerWaitingPiece(waiter)
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

	factory.processLines[lineID].setCurrentTool(LINE_DEFAULT_M1_POS, controlForm.toolTop)
	factory.processLines[lineID].setCurrentTool(LINE_DEFAULT_M1_POS, controlForm.toolBot)

	factory.processLines[lineID].plc.UpdateCommandOpcuaVars(controlForm.toCellCommand())
	opcuavars := factory.processLines[lineID].plc.CommandOpcuaVars()
	writeResponse, err := factory.plcClient.Write(opcuavars, ctx)

	log.Printf("[sendToLine] line:%v processForm:%v\n", lineID, controlForm)
	log.Printf("[sendToLine] line:%v control form: %+v %+v %+v %+v %+v %+v\n", lineID, opcuavars[0] , opcuavars[1], opcuavars[2],opcuavars[3],opcuavars[4],opcuavars[5])
	log.Printf("[sendToLine] Write response: %+v", writeResponse)

	utils.Assert(err == nil, "[sendToLine] Error writing to PLC")
	utils.Assert(controlForm != nil, "[sendToLine] controlForm is nil")
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
		pieceClaimedCh: claimed,
		claimPieceCh:   claimPieceCh,
		claimLock:      lock,
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

// TODO: rethink this way of handling updates
func runFactoryStateUpdateFunc(ctx context.Context, shipAckCh chan<- int16) {
	factory, mutex := getFactoryInstance()
	defer mutex.Unlock()

	// TODO: tweak the timeout value
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	err := factory.stateUpdateFunc(ctx, factory)
	utils.Assert(err == nil, "[updateFactoryState] Error updating factory state")
	for _, supplyLine := range factory.supplyLines {
		if supplyLine.PieceAcked() {
			shipAckCh <- supplyLine.LastCommandTxId()
		}
	}
}

func StartFactoryHandler(ctx context.Context, shipAckCh chan<- int16) <-chan error {
	errCh := make(chan error)

	// Connect to the factory floor plcs
	func() {
		factory, mutex := getFactoryInstance()
		defer mutex.Unlock()

		err := factory.plcClient.Connect(ctx)
		utils.Assert(err == nil, "[StartFactoryHandler] Error connecting to factory floor")
	}()

	// Start the factory floor
	go func() {
		defer close(errCh)
		defer close(shipAckCh)

		for {
			select {
			case <-ctx.Done():
				return

			default:
				runFactoryStateUpdateFunc(ctx, shipAckCh)
				time.Sleep(3 * time.Second)
			}
		}
	}()

	return errCh
}
