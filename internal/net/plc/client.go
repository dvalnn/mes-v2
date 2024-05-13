package plc

import (
	"context"
	"fmt"
	"log"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type opcuaVariable interface {
	asReadValue() (*ua.ReadValueID, error)
	asWriteValue() (*ua.WriteValue, error)
}

type ClientConfig struct {
	OpcuaEndpoint string
	ErpEndpoint   string
}

func (config ClientConfig) ConnectOpcua() (client *opcua.Client) {
	var new_client *opcua.Client
	var err error

	new_client, err = opcua.NewClient(config.OpcuaEndpoint)
	if err != nil {
		log.Printf("Error creating client: %s", err)
	}

	log.Print("Connecting...")
	
	err = new_client.Connect(context.Background())
	if err != nil {
		log.Printf("Error connecting to server: %s", err)
	}
	log.Println("Client connected successfully.")

	return new_client
}

func Read(vars []opcuaVariable, client *opcua.Client) (*ua.ReadResponse, error) {
	rvs := make([]*ua.ReadValueID, len(vars))

	for _, v := range vars {
		rv, err := v.asReadValue()
		if err != nil {
			return nil, fmt.Errorf("[plc.WriteBatch] %s", err.Error())
		}
		rvs = append(rvs, rv)
	}

	request := ua.ReadRequest{NodesToRead: rvs}
	response, err := client.Read(context.Background(), &request)
	if err != nil {
		return nil, fmt.Errorf("error writing to server: %s", err)
	}

	return response, nil
}

func Write(vars []opcuaVariable, client *opcua.Client) (*ua.WriteResponse, error) {
	wvs := make([]*ua.WriteValue, len(vars))

	for _, v := range vars {
		wv, err := v.asWriteValue()
		if err != nil {
			return nil, fmt.Errorf("[plc.WriteBatch] %s", err.Error())
		}
		wvs = append(wvs, wv)
	}

	request := ua.WriteRequest{NodesToWrite: wvs}
	response, err := client.Write(context.Background(), &request)
	if err != nil {
		return nil, fmt.Errorf("error writing to server: %s", err)
	}

	return response, nil
}
