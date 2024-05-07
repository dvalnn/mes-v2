package mes

import (
	"context"
	"sync"
	"time"
)

var (
	factoryInstance *factory
	factoryMutex    = &sync.Mutex{}
)

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
	conveyorLine []Conveyor
	readyForNext bool
}

func (pl *ProcessingLine) isReady() bool {
	return pl.readyForNext
}

func (pl *ProcessingLine) addItem(item *ConveyorItem) {
	assert(pl.isReady(), "[ProcessingLine.addItem] Processing line is not ready")
	assert(pl.conveyorLine[0].item == nil, "[ProcessingLine.addItem] Conveyor is not empty")

	pl.readyForNext = false
	pl.conveyorLine[0].item = item
}

type factory struct {
	processLines  map[string]*ProcessingLine
	supplyLines   []SupplyLine
	deliveryLines []DeliveryLine
}

func InitFactory() *factory {
	processLines := make(map[string]*ProcessingLine)

	processLines[ID_L0] = &ProcessingLine{
		conveyorLine: make([]Conveyor, LINE_CONVEYOR_SIZE),
		readyForNext: true,
	}

	for _, line := range []string{ID_L1, ID_L2, ID_L3, ID_L4, ID_L5, ID_L6} {
		processLines[line] = &ProcessingLine{
			conveyorLine: initType1Conveyor(),
			readyForNext: true,
		}
	}

	return &factory{
		supplyLines:   []SupplyLine{},
		processLines:  processLines,
		deliveryLines: []DeliveryLine{},
	}
}

func getFactoryInstance() *factory {
	if factoryInstance == nil {
		factoryInstance = InitFactory()
	}

	return factoryInstance
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
				// 2 - update the line status for any line that progressed
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return errCh
}

type factoryControlForm struct {
	toolTop    string
	toolBot    string
	pieceKind  string
	id         int16
	processBot bool
	processTop bool
}

func sendToLine(
	line string,
	recipe factoryControlForm,
) *itemHandler {
	transformCh := make(chan string)
	lineEntryCh := make(chan string)
	lineExitCh := make(chan string)
	errCh := make(chan error)

	factoryMutex.Lock()
	defer factoryMutex.Unlock()

	factory := getFactoryInstance()
	factory.processLines[line].addItem(&ConveyorItem{
		controlID: recipe.id,
		handler: &conveyorItemHandler{
			transformCh: transformCh,
			lineEntryCh: lineEntryCh,
			lineExitCh:  lineExitCh,
			errCh:       errCh,
		},
	})

	// TODO: send recipe to PLCs
	time.Sleep(500 * time.Millisecond)

	return &itemHandler{
		transformCh: transformCh,
		lineEntryCh: lineEntryCh,
		lineExitCh:  lineExitCh,
		errCh:       errCh,
	}
}

// TODO: implement this function
// This is a placeholder implementation
func lineGetFreeCompatible(
	ctx context.Context,
	priority int,
	toolTopMachine,
	toolBottomMachine string,
	warehouse string,
) (line string, canCurry bool, err error) {
	time.Sleep(250 * time.Millisecond)

	if warehouse == ID_W2 {
		return ID_L0, false, nil
	}

	return ID_L1, true, nil
}

// TODO: implement this function
func getNextControlID() int16 {
	return 0
}
