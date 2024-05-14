package plc

const (
	NUMBER_OF_CELLS = 6

	// PATHS
	CODESYS_PATH = "ns=4;s=|var|CODESYS Control Win V3 x64.Application."
	GVL_PATH     = CODESYS_PATH + "GVL."
	POU_PATH     = CODESYS_PATH + "POU."

	// INPUT WAREHOUSES DATA
	NODE_ID_INPUT_WAREHOUSES     = GVL_PATH + "cin"
	INPUT_WAREHOUSE_ID_POSTFIX    = ".id"
	INPUT_WAREHOUSE_PIECE_POSTFIX = ".piece"

	// OUTPUT WAREHOUSES DATA
	TOTAL_WAREHOUSES_PATH     = GVL_PATH + "total"
	NODE_ID_WAREHOUSE_1_TOTAL = TOTAL_WAREHOUSES_PATH + "W1"
	NODE_ID_WAREHOUSE_2_TOTAL = TOTAL_WAREHOUSES_PATH + "W2"

	// CELL DATA
	NODE_ID_CELLS           = GVL_PATH + "cell"
	CELL_ID_POSTFIX         = ".id"
	CELL_PIECE_POSTFIX      = ".piece"
	CELL_PROCESSBOT_POSTFIX = ".processBot"
	CELL_PROCESSTOP_POSTFIX = ".processTop"
	CELL_TOOLBOT_POSTFIX    = ".tool_MBot"
	CELL_TOOLTOP_POSTFIX    = ".tool_MTop"

	// CELL CONTROL DATA
	NODE_ID_CELLS_CONTROL     = POU_PATH + "id"
	CELLS_CONTROL_IN_POSTFIX  = "_i"
	CELLS_CONTROL_OUT_POSTFIX = "_o"
)
