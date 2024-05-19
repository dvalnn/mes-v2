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

type Client struct {
	opcua *opcua.Client
}

func NewClient(opcuaEndpoint string) (client *Client) {
	var opcuaClient *opcua.Client
	var err error

	opcuaClient, err = opcua.NewClient(opcuaEndpoint)
	if err != nil {
		log.Printf("[plc] Error creating client: %s", err)
		return nil
	}
	if opcuaClient == nil {
		log.Fatal("[plc] opcuaClient is nil")
		return nil
	}

	return &Client{opcua: opcuaClient}
}

func (c *Client) Connect(ctx context.Context) error {
	return c.opcua.Connect(ctx)
}

func (c *Client) Read(vars []opcuaVariable, ctx context.Context) (*ua.ReadResponse, error) {
	rvs := make([]*ua.ReadValueID, len(vars))

	for i, v := range vars {
		rv, err := v.asReadValue()
		if err != nil {
			return nil, fmt.Errorf("[plc.WriteBatch] %s", err.Error())
		}
		rvs[i] = rv
	}

	request := ua.ReadRequest{NodesToRead: rvs}

	response, err := c.opcua.Read(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("error writing to server: %s", err)
	}

	return response, nil
}

func (c *Client) Write(vars []opcuaVariable, ctx context.Context) (*ua.WriteResponse, error) {
	wvs := make([]*ua.WriteValue, len(vars))

	for i, v := range vars {
		wv, err := v.asWriteValue()
		if err != nil {
			return nil, fmt.Errorf("[plc.WriteBatch] %s", err.Error())
		}
		wvs[i] = wv
	}

	request := ua.WriteRequest{NodesToWrite: wvs}
	response, err := c.opcua.Write(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("error writing to server: %s", err)
	}

	return response, nil
}

func (c *Client) Close(ctx context.Context) {
	c.opcua.Close(ctx)
}
