package plc

import "strconv"

type CellCommand struct {
	TxId       OpcuaInt16
	PieceKind  OpcuaInt16
	ProcessBot OpcuaBool
	ProcessTop OpcuaBool
	ToolBot    OpcuaInt16
	ToolTop    OpcuaInt16
}

type CellState struct {
	TxIdPieceIN  OpcuaInt16 // tx id of the piece that entered the line
	TxIdPieceOut OpcuaInt16 // tx id of the piece that left the line
}

type Cell struct {
	command *CellCommand
	state   *CellState
}

func initCells() []*Cell {
	cells := make([]*Cell, NUMBER_OF_CELLS)

	for i := range NUMBER_OF_CELLS {
		commandPrefix := NODE_ID_CELL + strconv.Itoa(i+1)
		controlPrefix := NODE_ID_CELL_CONTROL + strconv.Itoa(i+1)

		cells[i] = &Cell{
			command: &CellCommand{
				TxId:       OpcuaInt16{nodeID: commandPrefix + CELL_ID_POSTFIX},
				PieceKind:  OpcuaInt16{nodeID: commandPrefix + CELL_PIECE_POSTFIX},
				ProcessBot: OpcuaBool{nodeID: commandPrefix + CELL_PROCESSBOT_POSTFIX},
				ProcessTop: OpcuaBool{nodeID: commandPrefix + CELL_PROCESSTOP_POSTFIX},
				ToolBot:    OpcuaInt16{nodeID: commandPrefix + CELL_TOOLBOT_POSTFIX},
				ToolTop:    OpcuaInt16{nodeID: commandPrefix + CELL_TOOLTOP_POSTFIX},
			},
			state: &CellState{
				TxIdPieceIN:  OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_IN_POSTFIX},
				TxIdPieceOut: OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_OUT_POSTFIX},
			},
		}
	}

	return cells
}

type SupplyLine struct {
	TxId      OpcuaInt16
	PieceKind OpcuaInt16
}

func InitSupplyLines() []*SupplyLine {
	supplyLines := make([]*SupplyLine, NUMBER_OF_SUPPLY_LINES)

	for i := range NUMBER_OF_SUPPLY_LINES {
		nodeIDPrefix := NODE_ID_SUPPLY_LINE + strconv.Itoa(i+1)

		supplyLines[i] = &SupplyLine{
			TxId: OpcuaInt16{
				nodeID: nodeIDPrefix + SUPPLY_LINE_ID_POSTFIX,
				Value:  0,
			},
			PieceKind: OpcuaInt16{
				nodeID: nodeIDPrefix + SUPPLY_LINE_PIECE_POSTFIX,
				Value:  0,
			},
		}
	}

	return supplyLines
}

type Warehouse struct {
	Quantity OpcuaInt16
}

// Unused warehouse fields
// QuantityP1 OpcuaInt16
// QuantityP2 OpcuaInt16
// QuantityP3 OpcuaInt16
// QuantityP4 OpcuaInt16
// QuantityP5 OpcuaInt16
// QuantityP6 OpcuaInt16
// QuantityP7 OpcuaInt16
// QuantityP8 OpcuaInt16
// QuantityP9 OpcuaInt16

func InitWarehouses() []*Warehouse {
	warehouses := make([]*Warehouse, NUMBER_OF_SUPPLY_LINES)

	for i := range NUMBER_OF_WAREHOUSES {
		warehouses[i] = &Warehouse{
			Quantity: OpcuaInt16{
				nodeID: NODE_ID_WAREHOUSE_TOTAL + strconv.Itoa(i+1),
				Value:  0,
			},
		}
	}

	return warehouses
}
