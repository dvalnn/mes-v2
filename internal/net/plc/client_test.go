package plc

import (
	"context"
	"flag"
	"strconv"
	"testing"
)



// ! NEEDS CODESYS TO BE RUNNING
func TestConnection(t *testing.T) {
	// endpoint := flag.String(
	// 	"endpoint",
	// 	"opc.tcp://192.168.1.74:4840",
	// 	"OPC UA Server Endpoint URL")

	client := NewClient(OPCUA_ENDPOINT)
	if client == nil {
		t.Errorf("Error connecting to server")
	}
	err := client.Connect(context.Background())
	if err != nil {
		t.Errorf("Error connecting to server")
	}
	defer client.Close(context.Background())

	t.Logf("Client connected successfully")
}

// ! NEEDS CODESYS TO BE RUNNING
func TestReadAndWrite(t *testing.T) {
	endpoint := flag.String(
		"endpoint",
		"opc.tcp://192.168.1.74:4840",
		"OPC UA Server Endpoint URL")

	client := NewClient(*endpoint)
	if client == nil {
		t.Errorf("Error connecting to server")
	}
	err := client.Connect(context.Background())
	if err != nil {
		t.Errorf("Error connecting to server")
	}
	defer client.Close(context.Background())

	ctx := context.Background()

	cellControl := make([]*CellCommand, 1)
	cellControl[0] = &CellCommand{
		TxId:       NewOpcuaInt16(NODE_ID_CELL+"1"+CELL_ID_POSTFIX, 1),
		PieceKind:  NewOpcuaInt16(NODE_ID_CELL+"1"+CELL_PIECE_POSTFIX, 1),
		ProcessBot: NewOpcuaBool(NODE_ID_CELL+"1"+CELL_PROCESSBOT_POSTFIX, true),
		ProcessTop: NewOpcuaBool(NODE_ID_CELL+"1"+CELL_PROCESSTOP_POSTFIX, false),
		ToolBot:    NewOpcuaInt16(NODE_ID_CELL+"1"+CELL_TOOLBOT_POSTFIX, 1),
		ToolTop:    NewOpcuaInt16(NODE_ID_CELL+"1"+CELL_TOOLTOP_POSTFIX, 1),
	}

	inputWarehouses := make([]*SupplyLine, 1)
	inputWarehouses[0] = &SupplyLine{
		TxId:      NewOpcuaInt16(NODE_ID_SUPPLY_LINE+"1"+SUPPLY_LINE_ID_POSTFIX, 2),
		PieceKind: NewOpcuaInt16(NODE_ID_SUPPLY_LINE+"1"+SUPPLY_LINE_PIECE_POSTFIX, 2),
	}

	cellState := make([]*CellState, 1)
	cellState[0] = &CellState{
		TxIdPieceIN:  NewOpcuaInt16(NODE_ID_CELL_CONTROL+"1"+CELL_CONTROL_IN_POSTFIX, 1),
		TxIdPieceOut: NewOpcuaInt16(NODE_ID_CELL_CONTROL+"1"+CELL_CONTROL_OUT_POSTFIX, 1),
	}

	warerhouses := make([]*Warehouse, 1)
	warerhouses[0] = &Warehouse{
		Quantity: NewOpcuaInt16(NODE_ID_WAREHOUSE_TOTAL+strconv.Itoa(1), 1),
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
	_, err = client.Write(cellControlVar, ctx)
	if err != nil {
		t.Errorf("Error writing variables: %s", err)
	}

	_, err = client.Write(inputWarehousesVar, ctx)
	if err != nil {
		t.Errorf("Error writing variables: %s", err)
	}

	_, err = client.Write(cellStateVar, ctx)
	if err != nil {
		t.Errorf("Error writing variables: %s", err)
	}

	// reads from the server and compares with the expected values ,values written in the declaration of the variables

	readResponse, err := client.Read(cellControlVar, ctx)
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

	readResponse, err = client.Read(inputWarehousesVar, ctx)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[0].Value.Value() != inputWarehouses[0].TxId.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[1].Value.Value() != inputWarehouses[0].PieceKind.Value {
		t.Errorf("Error reading variables: %s", err)
	}

	readResponse, err = client.Read(cellStateVar, ctx)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[0].Value.Value() != cellState[0].TxIdPieceIN.Value {
		t.Errorf("Error reading variables: %s", err)
	}
	if readResponse.Results[1].Value.Value() != cellState[0].TxIdPieceOut.Value {
		t.Errorf("Error reading variables: %s", err)
	}

	readResponse, err = client.Read(totalWarehouseVar, ctx)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}

	for _, v := range readResponse.Results {
		t.Logf("Response: %s", v.Value.Value())
	}
}


func TestInitCells(t *testing.T) {
	cells := InitCells()
	if len(cells) != NUMBER_OF_CELLS {
		t.Errorf("Error creating cells: %v", cells)
	}
	for _, cell := range cells {
		if cell.command.TxId.Value != 0 {
			t.Errorf("Error creating cells: %v", cells)
		}
		if cell.command.PieceKind.Value != 0 {
			t.Errorf("Error creating cells: %v", cells)
		}
		if cell.command.ProcessBot.Value != false {
			t.Errorf("Error creating cells: %v", cells)
		}
		if cell.command.ProcessTop.Value != false {
			t.Errorf("Error creating cells: %v", cells)
		}
		if cell.command.ToolBot.Value != 0 {
			t.Errorf("Error creating cells: %v", cells)
		}
		if cell.command.ToolTop.Value != 0 {
			t.Errorf("Error creating cells: %v", cells)
		}
		if cell.state.TxIdPieceIN.Value != 0 {
			t.Errorf("Error creating cells: %v", cells)
		}
		if cell.state.TxIdPieceOut.Value != 0 {
			t.Errorf("Error creating cells: %v", cells)
		}
	}
	t.Logf("Cells created successfully")
}


func TestInitSupplyLines(t *testing.T) {

	supplyLines := InitSupplyLines()
	if len(supplyLines) != NUMBER_OF_SUPPLY_LINES {
		t.Errorf("Error creating supply lines: %v", supplyLines)
	}
	for _, supplyLine := range supplyLines {
		if supplyLine.TxId.Value != 0 {
			t.Errorf("Error creating supply lines: %v", supplyLines)
		}
		if supplyLine.PieceKind.Value != 0 {
			t.Errorf("Error creating supply lines: %v", supplyLines)
		}
	}
	t.Logf("Supply lines created successfully")
}
func TestInitWarehouse(t *testing.T) {
	warehouses := InitWarehouses()
	if len(warehouses) != NUMBER_OF_WAREHOUSES {
		t.Errorf("Error creating warehouses: %v", warehouses)
	}
	for _, warehouse := range warehouses {
		if warehouse.Quantity.Value != 0 {
			t.Errorf("Error creating warehouses: %v", warehouses)
		}
	}
	t.Logf("Warehouses created successfully")
}


