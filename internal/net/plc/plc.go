package plc

import (
	"mes/internal/utils"
	"strconv"

	"github.com/gopcua/opcua/ua"
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
	return c.state.OpcuaVars()
}

func (c *Cell) UpdateState(response *ua.ReadResponse) {
	utils.Assert(response != nil, "Response is nil")
	utils.Assert(len(response.Results) == 2, "Cell state response has wrong number of results")
	utils.Assert(response.Results[0].Value.Type() == ua.TypeIDInt16, "Cell state response has wrong type")
	utils.Assert(response.Results[1].Value.Type() == ua.TypeIDInt16, "Cell state response has wrong type")

	*c.oldState = *c.state // save old state before updating
	c.state.TxIdPieceIN.Value = response.Results[0].Value.Value().(int16)
	c.state.TxIdPieceOut.Value = response.Results[1].Value.Value().(int16)
}

func (c *Cell) UpdateCommandOpcuaVars(pcf *CellCommand) {
	c.command.TxId.Value = pcf.TxId.Value
	c.command.PieceKind.Value = pcf.PieceKind.Value
	c.command.ProcessBot.Value = pcf.ProcessBot.Value
	c.command.ProcessTop.Value = pcf.ProcessTop.Value
	c.command.ToolBot.Value = pcf.ToolBot.Value
	c.command.ToolTop.Value = pcf.ToolTop.Value
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

// Returns true if a command was started (piece entered the cell)
func (c *Cell) PieceEnteredM1() bool {
	return c.state.TxIdPieceIN.Value == c.command.TxId.Value &&
		c.state.TxIdPieceIN.Value != c.oldState.TxIdPieceIN.Value
}

// Returns true if a command was completed (piece left the cell)
func (c *Cell) PieceLeft() bool {
	return c.state.TxIdPieceOut.Value != c.oldState.TxIdPieceOut.Value
}

func InitCells() []*Cell {
	cells := make([]*Cell, NUMBER_OF_CELLS)

	for i := range NUMBER_OF_CELLS {

		commandPrefix := NODE_ID_CELL + strconv.Itoa(i)
		controlPrefix := NODE_ID_CELL_CONTROL + strconv.Itoa(i)

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
				TxIdPieceIN:  OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_OUT_POSTFIX},
				TxIdPieceOut: OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_IN_POSTFIX},
			},
			oldState: &CellState{
				TxIdPieceIN:  OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_OUT_POSTFIX},
				TxIdPieceOut: OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_IN_POSTFIX},
			},
		}
	}

	return cells
}

type SupplyLineCommand struct {
	TxId      OpcuaInt16
	PieceKind OpcuaInt16
}

type SupplyLineState struct {
	TxAckId OpcuaInt16
}

type SupplyLine struct {
	command  *SupplyLineCommand
	state    *SupplyLineState
	oldState *SupplyLineState
}

func InitSupplyLines() []*SupplyLine {
	supplyLines := make([]*SupplyLine, NUMBER_OF_SUPPLY_LINES)

	for i := range NUMBER_OF_SUPPLY_LINES {
		commandNodeID := NODE_ID_SUPPLY_LINE + strconv.Itoa(i+1)
		stateNodeID := NODE_ID_IDX_SUPPLY_LINE + strconv.Itoa(i+1)

		supplyLines[i] = &SupplyLine{
			command: &SupplyLineCommand{
				TxId: OpcuaInt16{
					nodeID: commandNodeID + SUPPLY_LINE_ID_POSTFIX,
					Value:  0,
				},
				PieceKind: OpcuaInt16{
					nodeID: commandNodeID + SUPPLY_LINE_PIECE_POSTFIX,
					Value:  0,
				},
			},
			state: &SupplyLineState{
				TxAckId: OpcuaInt16{
					nodeID: stateNodeID,
					Value:  0,
				},
			},
			oldState: &SupplyLineState{
				TxAckId: OpcuaInt16{
					nodeID: stateNodeID,
					Value:  0,
				},
			},
		}
	}

	return supplyLines
}

func (s *SupplyLine) CommandOpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&s.command.TxId,
		&s.command.PieceKind,
	}
}

func (s *SupplyLine) StateOpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&s.state.TxAckId,
	}
}

func (s *SupplyLine) NewShipment(pieceKind int16) {
	s.command.TxId.Value++
	s.command.PieceKind.Value = pieceKind
}

func (s *SupplyLine) LastCommandTxId() int16 {
	return s.command.TxId.Value
}

func (s *SupplyLine) UpdateState(response *ua.ReadResponse) {
	utils.Assert(response != nil, "Response is nil")
	utils.Assert(len(response.Results) == 1, "Supply line state response has wrong number of results")
	utils.Assert(response.Results[0].Value.Type() == ua.TypeIDInt16, "Supply line state response has wrong type")
	s.oldState.TxAckId.Value = s.state.TxAckId.Value // save old state before updating
	s.state.TxAckId.Value = response.Results[0].Value.Value().(int16)
}

func (s *SupplyLine) PieceAcked() bool {
	return s.state.TxAckId.Value == s.command.TxId.Value &&
		s.state.TxAckId.Value != s.oldState.TxAckId.Value
}

type Warehouse struct {
	Quantity OpcuaInt16
}

func InitWarehouses() []*Warehouse {
	warehouses := make([]*Warehouse, NUMBER_OF_WAREHOUSES)

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

type FactoryOutput struct {
	TxId  OpcuaInt16
	Np    OpcuaInt16
	Piece OpcuaInt16
}

func InitOutputs() []*FactoryOutput {
	outputs := make([]*FactoryOutput, NUMBER_OF_OUTPUTS)

	for i := range NUMBER_OF_OUTPUTS {
		nodeIDPrefix := NODE_ID_OUTPUTS + strconv.Itoa(i+1)

		outputs[i] = &FactoryOutput{
			TxId: OpcuaInt16{
				nodeID: nodeIDPrefix + OUTPUT_ID_POSTFIX,
				Value:  0,
			},
			Np: OpcuaInt16{
				nodeID: nodeIDPrefix + OUTPUT_NP_POSTFIX,
				Value:  0,
			},
			Piece: OpcuaInt16{
				nodeID: nodeIDPrefix + OUTPUT_PIECE_POSTFIX,
				Value:  0,
			},
		}
	}

	return outputs
}

func (fo *FactoryOutput) OpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&fo.TxId,
		&fo.Np,
		&fo.Piece,
	}
}
