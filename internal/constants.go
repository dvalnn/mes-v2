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

	// ERP system product kinds.

	P_KIND_1 string = "P1" // Product Kind 1
	P_KIND_2 string = "P2" // Product Kind 2
	P_KIND_3 string = "P3" // Product Kind 3
	P_KIND_4 string = "P4" // Product Kind 4
	P_KIND_5 string = "P5" // Product Kind 5
	P_KIND_7 string = "P7" // Product Kind 7
	P_KIND_8 string = "P8" // Product Kind 8
	P_KIND_9 string = "P9" // Product Kind 9
)
