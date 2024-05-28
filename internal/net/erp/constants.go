package erp

import "time"

const (
	ENDPOINT_DATE              = "/date"
	ENDPOINT_WAREHOUSE         = "/warehouse"
	ENDPOINT_SHIPMENT_ARRIVAL  = "/materials/arrivals"
	ENDPOINT_EXPECTED_SHIPMENT = "/materials/expected"
	ENDPOINT_TRANSFORMATION    = "/transformations"
	ENDPOINT_PRODUCTION        = "/production"
	ENDPOINT_DELIVERY          = "/deliveries"
	ENDPOINT_DELIVERY_STATS    = "/deliveries/statistics"

	ENDPOINT_DEFAULT_BASE_URL = "http://localhost:8080"
	DEFAULT_HTTP_TIMEOUT      = 5 * time.Second
)
