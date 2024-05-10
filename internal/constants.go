package mes

import "time"

type ctxKey string

const (
	// Context keys.

	KEY_SIM_TIME     ctxKey = "simTime"
	KEY_HTTP_TIMEOUT ctxKey = "httpTimeout"
	KEY_ERP_URL      ctxKey = "erpUrl"

	// Default Context values.

	DEFAULT_SIM_TIME     = 1 * time.Minute
	DEFAULT_HTTP_TIMEOUT = 500 * time.Millisecond
	DEFAULT_ERP_URL      = "http://localhost:8080"

	// Endpoints in the ERP system.

	ENDPOINT_BASE_URL          = "http://localhost:8080"
	ENDPOINT_NEW_DATE          = "/date"
	ENDPOINT_WAREHOUSE         = "/warehouse"
	ENDPOINT_SHIPMENT_ARRIVAL  = "/materials/arrivals"
	ENDPOINT_EXPECTED_SHIPMENT = "/materials/expected"
	ENDPOINT_TRANSFORMATION    = "/transformations"
	ENDPOINT_PRODUCTION        = "/production"
	ENDPOINT_DELIVERY          = "/deliveries"

	// ERP system warehouse IDs.

	ID_W1 string = "W1" // Warehouse 1 ID
	ID_W2 string = "W2" // Warehouse 2 ID

	// ERP system line IDs.

	ID_L0 string = "L0" // Line 0 ID
	ID_L1 string = "L1" // Line 1 ID
	ID_L2 string = "L2" // Line 2 ID
	ID_L3 string = "L3" // Line 3 ID
	ID_L4 string = "L4" // Line 4 ID
	ID_L5 string = "L5" // Line 5 ID
	ID_L6 string = "L6" // Line 6 ID

	// Factory system piece kinds.

	P_KIND_1 string = "P1" // piece Kind 1
	P_KIND_2 string = "P2" // piece Kind 2
	P_KIND_3 string = "P3" // piece Kind 3
	P_KIND_4 string = "P4" // piece Kind 4
	P_KIND_5 string = "P5" // piece Kind 5
	P_KIND_7 string = "P7" // piece Kind 7
	P_KIND_8 string = "P8" // piece Kind 8
	P_KIND_9 string = "P9" // piece Kind 9

	// Factory tool Types

	TOOL_1 string = "T1"
	TOOL_2 string = "T2"
	TOOL_3 string = "T3"
	TOOL_4 string = "T4"
	TOOL_5 string = "T5"
	TOOL_6 string = "T6"

	// Factory constants
	LINE_CONVEYOR_SIZE  = 5
	LINE_DEFAULT_M1_POS = 1
	LINE_DEFAULT_M2_POS = 3
)
