package plc

import (
	"context"
	"flag"
	"testing"
)

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

func TestReadAndWrite(t *testing.T) {
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

	cellControlReadForm := make([]*CellCommand, 6)

	cellControlReadForm[0] = &CellCommand{
		TxId:       NewOpcuaInt16(NODE_ID_CELL1_ID, 1),
		PieceKind:  NewOpcuaInt16(NODE_ID_CELL1_PIECE, 1),
		ProcessBot: NewOpcuaBool(NODE_ID_CELL1_PROCESSBOT, true),
		ProcessTop: NewOpcuaBool(NODE_ID_CELL1_PROCESSTOP, false),
		ToolBot:    NewOpcuaInt16(NODE_ID_CELL1_TOOLBOT, 2),
		ToolTop:    NewOpcuaInt16(NODE_ID_CELL1_TOOLTOP, 3),
	}

	// inserts all the variablles of the cell control read form into a apcuavariable array
	vars := []opcuaVariable{
		cellControlReadForm[0].TxId,
		cellControlReadForm[0].PieceKind,
		cellControlReadForm[0].ProcessBot,
		cellControlReadForm[0].ProcessTop,
		cellControlReadForm[0].ToolBot,
		cellControlReadForm[0].ToolTop,
	}

	// prints the variables
	readResponse, err := Read(vars, client)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}

	for _, v := range readResponse.Results {
		t.Logf("Response: %s", v.Value.Value())
	}
	// write the variables
	_, err = Write(vars, client)
	if err != nil {
		t.Errorf("Error writing variables: %s", err)
	}
}
