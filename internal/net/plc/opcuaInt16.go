package plc

import (
	"fmt"

	"github.com/gopcua/opcua/ua"
)

type OpcuaInt16 struct {
	nodeID string
	Value  int16
}

func NewOpcuaInt16(nodeID string, value int16) OpcuaInt16 {
	return OpcuaInt16{
		nodeID: nodeID,
		Value:  value,
	}
}

func (i OpcuaInt16) asReadValue() (*ua.ReadValueID, error) {
	nodeID, err := ua.ParseNodeID(i.nodeID)
	if err != nil {
		return nil, fmt.Errorf("[opcuaInt16.asReadValue] error parsing nodeID: %s", err)
	}

	return &ua.ReadValueID{
		NodeID: nodeID,
	}, err
}

func (i OpcuaInt16) asWriteValue() (*ua.WriteValue, error) {
	nodeID, err := ua.ParseNodeID(i.nodeID)
	if err != nil {
		return nil, fmt.Errorf("[opcuaInt16.asReadValue] error parsing nodeID: %s", err)
	}

	value, err := ua.NewVariant(i.Value)
	if err != nil {
		return nil, fmt.Errorf("[opcuaInt16.asWriteValue] error creating variant: %s", err)
	}

	return &ua.WriteValue{
		NodeID:      nodeID,
		AttributeID: ua.AttributeIDValue,
		Value: &ua.DataValue{
			EncodingMask: ua.DataValueValue,
			Value:        value,
		},
	}, nil
}
