package plc

import (
	"log"
	"mes/internal/utils"
	"strconv"

	"github.com/gopcua/opcua/ua"
)

type CellCommand struct {
	TxId      OpcuaInt16
	PieceKind OpcuaInt16

	ProcessTop OpcuaBool
	ToolTop    OpcuaInt16
	RepeatTop  OpcuaInt16

	ProcessBot OpcuaBool
	ToolBot    OpcuaInt16
	RepeatBot  OpcuaInt16
}

func (cc *CellCommand) OpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&cc.TxId,
		&cc.PieceKind,

		&cc.ProcessTop,
		&cc.ToolTop,
		&cc.RepeatTop,

		&cc.ProcessBot,
		&cc.ToolBot,
		&cc.RepeatBot,
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
	command     *CellCommand
	state       *CellState
	oldState    *CellState
	cellExitAck OpcuaInt16
}

func (c *Cell) StateOpcuaVars() []opcuaVariable {
	return c.state.OpcuaVars()
}

func (c *Cell) AckOpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&c.cellExitAck,
	}
}

func (c *Cell) AckPiece(txId int16) {
	utils.Assert(txId >= 0, "[AckPiece] Invalid txId must be greater than 0")
	if txId != c.cellExitAck.Value+1 {
		log.Printf("[AckPiece - WARNING] txId: %d, cellExitAck: %d\n", txId, c.cellExitAck.Value)
	}
	c.cellExitAck.Value = txId
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

	c.command.ProcessTop.Value = pcf.ProcessTop.Value
	c.command.ToolTop.Value = pcf.ToolTop.Value
	c.command.RepeatTop.Value = pcf.RepeatTop.Value

	c.command.ProcessBot.Value = pcf.ProcessBot.Value
	c.command.ToolBot.Value = pcf.ToolBot.Value
	c.command.RepeatBot.Value = pcf.RepeatBot.Value
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
		ackID := NODE_ID_WAREHOUSE_ACK + strconv.Itoa(i)

		cells[i] = &Cell{
			command: &CellCommand{
				TxId:      OpcuaInt16{nodeID: commandPrefix + CELL_ID_POSTFIX},
				PieceKind: OpcuaInt16{nodeID: commandPrefix + CELL_PIECE_POSTFIX},

				ProcessTop: OpcuaBool{nodeID: commandPrefix + CELL_PROCESSTOP_POSTFIX},
				ToolTop:    OpcuaInt16{nodeID: commandPrefix + CELL_TOOLTOP_POSTFIX},
				RepeatTop:  OpcuaInt16{nodeID: commandPrefix + CELL_REPEATTOP_POSTFIX},

				ProcessBot: OpcuaBool{nodeID: commandPrefix + CELL_PROCESSBOT_POSTFIX},
				ToolBot:    OpcuaInt16{nodeID: commandPrefix + CELL_TOOLBOT_POSTFIX},
				RepeatBot:  OpcuaInt16{nodeID: commandPrefix + CELL_REPEATBOT_POSTFIX},
			},
			state: &CellState{
				TxIdPieceIN:  OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_OUT_POSTFIX},
				TxIdPieceOut: OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_IN_POSTFIX},
			},
			oldState: &CellState{
				TxIdPieceIN:  OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_OUT_POSTFIX},
				TxIdPieceOut: OpcuaInt16{nodeID: controlPrefix + CELL_CONTROL_IN_POSTFIX},
			},
			cellExitAck: OpcuaInt16{nodeID: ackID},
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

type DeliveryCommand struct {
	TxId  OpcuaInt16
	Np    OpcuaInt16
	Piece OpcuaInt16
}

type DeliveryState struct {
	TxAckId OpcuaInt16
}

type DeliveryLine struct {
	command  *DeliveryCommand
	state    *DeliveryState
	oldState *DeliveryState
}

func InitDeliveryLines() []*DeliveryLine {
	lines := make([]*DeliveryLine, NUMBER_OF_OUTPUTS)

	for i := range NUMBER_OF_OUTPUTS {
		nodeIDPrefix := NODE_ID_OUTPUTS + strconv.Itoa(i+1)
		nodeIDOoutputConfirm := NODE_ID_OUTPUT_ACK + strconv.Itoa(i+1)

		lines[i] = &DeliveryLine{
			command: &DeliveryCommand{
				TxId:  OpcuaInt16{nodeID: nodeIDPrefix + OUTPUT_ID_POSTFIX},
				Np:    OpcuaInt16{nodeID: nodeIDPrefix + OUTPUT_NP_POSTFIX},
				Piece: OpcuaInt16{nodeID: nodeIDPrefix + OUTPUT_PIECE_POSTFIX},
			},
			state:    &DeliveryState{TxAckId: OpcuaInt16{nodeID: nodeIDOoutputConfirm}},
			oldState: &DeliveryState{TxAckId: OpcuaInt16{nodeID: nodeIDOoutputConfirm}},
		}
	}

	return lines
}

func (dl *DeliveryLine) CommandOpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&dl.command.TxId,
		&dl.command.Np,
		&dl.command.Piece,
	}
}

func (dl *DeliveryLine) StateOpcuaVars() []opcuaVariable {
	return []opcuaVariable{
		&dl.state.TxAckId,
	}
}

func (dl *DeliveryLine) UpdateState(response *ua.ReadResponse) {
	utils.Assert(response != nil, "Response is nil")
	utils.Assert(len(response.Results) == 1,
		"Delivery line state response has wrong number of results")
	utils.Assert(response.Results[0].Value.Type() == ua.TypeIDInt16,
		"Delivery line state response has wrong type")

	dl.oldState.TxAckId.Value = dl.state.TxAckId.Value // save old state before updating
	dl.state.TxAckId.Value = response.Results[0].Value.Value().(int16)
}

func (dl *DeliveryLine) SetDelivery(quantity int16, pieceKind int16) {
	dl.command.TxId.Value++
	dl.command.Np.Value = quantity
	dl.command.Piece.Value = pieceKind
}

func (dl *DeliveryLine) LastCommandTxId() int16 {
	return dl.command.TxId.Value
}

func (dl *DeliveryLine) PieceAcked() bool {
	return dl.state.TxAckId.Value == dl.command.TxId.Value &&
		dl.state.TxAckId.Value != dl.oldState.TxAckId.Value
}
