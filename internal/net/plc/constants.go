package plc

const (
	OPCUA_ENDPOINT = "opc.tcp://192.168.1.4:4840"

	CODESYS_PATH = "ns=4;s=|var|CODESYS Control Win V3 x64.Application."
	GVL_PATH     = CODESYS_PATH + "GVL."
	POU_PATH     = CODESYS_PATH + "POU."

	NUMBER_OF_SUPPLY_LINES    = 4
	NODE_ID_SUPPLY_LINE       = GVL_PATH + "cin"
	SUPPLY_LINE_ID_POSTFIX    = ".id"
	SUPPLY_LINE_PIECE_POSTFIX = ".piece"
	NODE_ID_IDX_SUPPLY_LINE   = POU_PATH + "id_in"

	NUMBER_OF_WAREHOUSES    = 2
	NODE_ID_WAREHOUSE_TOTAL = GVL_PATH + "totalW"

	// Cell command opcua node data
	NUMBER_OF_CELLS         = 7
	NODE_ID_CELL            = GVL_PATH + "cell"
	CELL_ID_POSTFIX         = ".id"
	CELL_PIECE_POSTFIX      = ".piece"
	CELL_PROCESSBOT_POSTFIX = ".processBot"
	CELL_PROCESSTOP_POSTFIX = ".processTop"
	CELL_TOOLBOT_POSTFIX    = ".tool_MBot"
	CELL_TOOLTOP_POSTFIX    = ".tool_MTop"
	CELL_REPEATTOP_POSTFIX  = ".repeatTop"
	CELL_REPEATBOT_POSTFIX  = ".repeatBot"

	// Warehouse entry Ack
	NODE_ID_WAREHOUSE_ACK = POU_PATH + "mes"

	// Cell state opcua node data
	NODE_ID_CELL_CONTROL     = POU_PATH + "id"
	CELL_CONTROL_IN_POSTFIX  = "_i"
	CELL_CONTROL_OUT_POSTFIX = "_o"

	// Delivery line opcua node data
	NUMBER_OF_OUTPUTS    = 4
	NODE_ID_OUTPUTS      = GVL_PATH + "roller"
	OUTPUT_ID_POSTFIX    = ".id"
	OUTPUT_NP_POSTFIX    = ".np"
	OUTPUT_PIECE_POSTFIX = ".piece"

	NODE_ID_OUTPUT_ACK = POU_PATH + "idr"
)
