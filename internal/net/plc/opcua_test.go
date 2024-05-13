package plc

import (
	"flag"
	"context"
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


	//populates the 

	t.Logf("Client connected successfully")
}

func TestRead(t *testing.T) {
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

	//populates the controller with the variables

	//read the variables
	
	cellControlReadForm := make([]*Place_holder_cell_struct, 1)

	cellControlReadForm[0] = &Place_holder_cell_struct{
		Index: NewOpcuaInt16(NODE_ID_CELL1_ID, 0),
		Piece: NewOpcuaInt16(NODE_ID_CELL1_PIECE, 0),
		ProcessBot: NewOpcuaBool(NODE_ID_CELL1_PROCESSBOT, false),
		ProcessTop: NewOpcuaBool(NODE_ID_CELL1_PROCESSTOP, false),
		ToolBot: NewOpcuaInt16(NODE_ID_CELL1_TOOLBOT, 0),
		ToolTop: NewOpcuaInt16(NODE_ID_CELL1_TOOLTOP, 0),
	}

	//prints the cell control Read Form
	//t.Logf("Cell Control Read Form: %v", cellControlReadForm[0].Index)

	//creates the OPCUA variables




	//read the variables
	readResponse, err := Read([]opcuaVariable{cellControlReadForm[0].Index}, client)
	if err != nil {
		t.Errorf("Error reading variables: %s", err)
	}
	readResponse.Results[0].Value.Value()

	t.Logf("Response: %v", 	readResponse.Results[0].Value.Value())
	
}