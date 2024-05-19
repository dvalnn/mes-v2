package plc

import (
	"fmt"

	"github.com/gopcua/opcua/ua"
)

type OpcuaBool struct {
	nodeID string
	Value  bool
}

func NewOpcuaBool(nodeID string, value bool) OpcuaBool {
	return OpcuaBool{
		nodeID: nodeID,
		Value:  value,
	}
}

func (b OpcuaBool) asReadValue() (*ua.ReadValueID, error) {
	nodeID, err := ua.ParseNodeID(b.nodeID)
	if err != nil {
		return nil, fmt.Errorf("[opcuaInt16.asReadValue] error parsing nodeID: %s", err)
	}

	return &ua.ReadValueID{
		NodeID: nodeID,
	}, err
}

func (b OpcuaBool) asWriteValue() (*ua.WriteValue, error) {
	nodeID, err := ua.ParseNodeID(b.nodeID)
	if err != nil {
		return nil, fmt.Errorf("[opcuaInt16.asReadValue] error parsing nodeID: %s", err)
	}

	value, err := ua.NewVariant(b.Value)
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
