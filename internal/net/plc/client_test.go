package plc

import (
	"context"
	"flag"
	"testing"
)

//! NEEDS CODESYS TO BE RUNNING
func TestConnection(t *testing.T) {
	endpoint := flag.String(
		"endpoint",
		"opc.tcp://192.168.1.74:4840",
		"OPC UA Server Endpoint URL")

	config := ClientConfig{
		OpcuaEndpoint: *endpoint,
	}

	client := config.ConnectOpcua()
	if client == nil {
		t.Errorf("Error connecting to server")
	}

	defer client.Close(context.Background())

	// populates the

	t.Logf("Client connected successfully")
}


//! NEEDS CODESYS TO BE RUNNING
func TestReadAndWrite(t *testing.T) {
	endpoint := flag.String(
		"endpoint",
		"opc.tcp://10.227.157.49:4840",
		"OPC UA Server Endpoint URL")

	config := ClientConfig{
		OpcuaEndpoint: *endpoint,
	}

	client := config.ConnectOpcua()
	if client == nil {
		t.Errorf("Error connecting to server")
	}

	defer client.Close(context.Background())

	cellControl := make([]*CellCommand, 1)
	cellControl[0] = &CellCommand{
		TxId:       NewOpcuaInt16(NODE_ID_CELLS+"1"+CELL_ID_POSTFIX, 1),
		PieceKind:  NewOpcuaInt16(NODE_ID_CELLS+"1"+CELL_PIECE_POSTFIX, 1),
		ProcessBot: NewOpcuaBool(NODE_ID_CELLS+"1"+CELL_PROCESSBOT_POSTFIX, true),
		ProcessTop: NewOpcuaBool(NODE_ID_CELLS+"1"+CELL_PROCESSTOP_POSTFIX, false),
		ToolBot:    NewOpcuaInt16(NODE_ID_CELLS+"1"+CELL_TOOLBOT_POSTFIX, 1),
		ToolTop:    NewOpcuaInt16(NODE_ID_CELLS+"1"+CELL_TOOLTOP_POSTFIX, 1),
	}

	inputWarehouses := make([]*InputWarehouses, 1)
	inputWarehouses[0] = &InputWarehouses{
		TxId:      NewOpcuaInt16(NODE_ID_INPUT_WAREHOUSES+"1"+INPUT_WAREHOUSE_ID_POSTFIX, 2),
		PieceKind: NewOpcuaInt16(NODE_ID_INPUT_WAREHOUSES+"1"+INPUT_WAREHOUSE_PIECE_POSTFIX, 2),
	}

	cellState := make([]*CellState, 1)
	cellState[0] = &CellState{
		TxIdPieceIN:  NewOpcuaInt16(NODE_ID_CELLS_CONTROL+"1"+CELLS_CONTROL_IN_POSTFIX, 1),
		TxIdPieceOut: NewOpcuaInt16(NODE_ID_CELLS_CONTROL+"1"+CELLS_CONTROL_OUT_POSTFIX, 1),
	}

	warerhouses := make([]*Warehouses, 1)
	warerhouses[0] = &Warehouses{
		Quantity: NewOpcuaInt16(NODE_ID_WAREHOUSE_1_TOTAL, 1),
	}

	// inserts all the variables of the cell control read form into a apcuavariable array
	cellControlVar := []opcuaVariable{
		cellControl[0].TxId,
		cellControl[0].PieceKind,
		cellControl[0].ProcessBot,
		cellControl[0].ProcessTop,
		cellControl[0].ToolBot,
		cellControl[0].ToolTop,
	}

	inputWarehousesVar := []opcuaVariable{
		inputWarehouses[0].TxId,
		inputWarehouses[0].PieceKind,
	}

	cellStateVar := []opcuaVariable{
		cellState[0].TxIdPieceIN,
		cellState[0].TxIdPieceOut,
	}

	totalWarehouseVar := []opcuaVariable{
		warerhouses[0].Quantity,
	}

	// writes all the variables to teh server
	_, err := Write(cellControlVar, client)
	if err != nil {
		t.Errorf("Error writing variables: %s", err)
	}

	_, err = Write(inputWarehousesVar, client)
	if err != nil {
		t.Errorf("Error writing variables: %s", err)
	}

	_, err = Write(cellStateVar, client)
	if err != nil {
		t.Errorf("Error writing variables: %s", err)
	}

	//reads from the server and compares with the expected values ,values written in the declaration of the variables

	readResponse, err := Read(cellControlVar, client)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[0].Value.Value() != cellControl[0].TxId.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[1].Value.Value() != cellControl[0].PieceKind.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[2].Value.Value() != cellControl[0].ProcessBot.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[3].Value.Value() != cellControl[0].ProcessTop.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[4].Value.Value() != cellControl[0].ToolBot.Value {

		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[5].Value.Value() != cellControl[0].ToolTop.Value {
		t.Errorf("Error reading variables: %s", err)
	}

	readResponse, err = Read(inputWarehousesVar, client)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[0].Value.Value() != inputWarehouses[0].TxId.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[1].Value.Value() != inputWarehouses[0].PieceKind.Value {
		t.Errorf("Error reading variables: %s", err)
	}

	readResponse, err = Read(cellStateVar, client)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[0].Value.Value() != cellState[0].TxIdPieceIN.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[1].Value.Value() != cellState[0].TxIdPieceOut.Value {
		t.Errorf("Error reading variables: %s", err)
	}

	readResponse, err = Read(totalWarehouseVar, client)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}

	for _, v := range readResponse.Results {
		t.Logf("Response: %s", v.Value.Value())
	}
}
