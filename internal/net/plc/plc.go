package plc

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

type InputWarehouses struct {
	TxId      OpcuaInt16
	PieceKind OpcuaInt16
}

type Warehouses struct {
	Quantity   OpcuaInt16
	QuantityP1 OpcuaInt16
	QuantityP2 OpcuaInt16
	QuantityP3 OpcuaInt16
	QuantityP4 OpcuaInt16
	QuantityP5 OpcuaInt16
	QuantityP6 OpcuaInt16
	QuantityP7 OpcuaInt16
	QuantityP8 OpcuaInt16
	QuantityP9 OpcuaInt16
}
