package plc

import (
	"strconv"
)

type CellCommand struct {
	TxId       OpcuaInt16
	PieceKind  OpcuaInt16
	ProcessBot OpcuaBool
	ProcessTop OpcuaBool
	ToolBot    OpcuaInt16
	ToolTop    OpcuaInt16
}

func (cc *CellCommand) OpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&cc.TxId,
		&cc.PieceKind,
		&cc.ProcessBot,
		&cc.ProcessTop,
		&cc.ToolBot,
		&cc.ToolTop,
	}
}

type CellState struct {
	TxIdPieceIN  OpcuaInt16 // tx id of the piece that entered the line
	TxIdPieceOut OpcuaInt16 // tx id of the piece that left the line
}

func (cs *CellState) OpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&cs.TxIdPieceIN,
		&cs.TxIdPieceOut,
	}
}

type Cell struct {
	command *CellCommand

	state    *CellState
	oldState *CellState
}

func (c *Cell) StateOpcuaVars() []opcuaVariable {
	c.oldState = c.state // save old state before updating
	return c.state.OpcuaVars()
}

func (c *Cell) InPieceTxId() int16 {
	return c.state.TxIdPieceIN.Value
}

func (c *Cell) OutPieceTxId() int16 {
	return c.state.TxIdPieceOut.Value
}

func (c *Cell) CommandOpcuaVars() []opcuaVariable {
	return c.command.OpcuaVars()
}

func (c *Cell) SetCommand(command *CellCommand) {
	c.command = command
}

func (c *Cell) LastCommandTxId() int16 {
	return c.command.TxId.Value
}

func (c *Cell) Progressed() bool {
	return c.state.TxIdPieceIN.Value != c.oldState.TxIdPieceIN.Value ||
		c.state.TxIdPieceOut.Value != c.oldState.TxIdPieceOut.Value
}

func InitCells() []*Cell {
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
			// init old state = state
			state: &CellState{
				TxIdPieceIN:  OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_IN_POSTFIX},
				TxIdPieceOut: OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_OUT_POSTFIX},
			},
			oldState: &CellState{
				TxIdPieceIN:  OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_IN_POSTFIX},
				TxIdPieceOut: OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_OUT_POSTFIX},
			},
		}
	}

	return cells
}

// TODO: add missing fields
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

func (s *SupplyLine) OpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&s.TxId,
		&s.PieceKind,
	}
}

func (s *SupplyLine) NewShipment(pieceKind int16) {
	s.TxId.Value++
	s.PieceKind.Value = pieceKind
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

func (w *Warehouse) OpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&w.Quantity,
	}
}
