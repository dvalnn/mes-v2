package mes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// PostToErp sends a POST request to the ERP system at the given endpoint
// with the provided form data. It returns an error if the request fails.
// Form data is sent as x-www-form-urlencoded.
//
// Context must be provided with values for
// - KEY_ERP_URL (erp base url - string)
// - KEY_HTTP_TIMEOUT (timeout for client request - time.Duration)
func PostToErp(ctx context.Context, endpoint string, formData url.Values) error {
	timeout, exists := ctx.Value(KEY_HTTP_TIMEOUT).(time.Duration)
	assert(exists, "[PostToErp] http timeout not found in context")

	baseUrl, exists := ctx.Value(KEY_ERP_URL).(string)
	assert(exists, "[PostToErp] erp url not found in context")

	client := http.Client{
		Timeout: timeout,
	}

	url := fmt.Sprintf("%s%s", baseUrl, endpoint)
	resp, err := client.PostForm(url, formData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("[PostToErp] unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func GetFromErp(ctx context.Context, endpoint string) (*http.Response, error) {
	timeout, exists := ctx.Value(KEY_HTTP_TIMEOUT).(time.Duration)
	assert(exists, "[GetFromErp] http timeout not found in context")
	baseUrl, exists := ctx.Value(KEY_ERP_URL).(string)
	assert(exists, "[GetFromErp] erp url not found in context")

	client := http.Client{
		Timeout: timeout,
	}

	url := fmt.Sprintf("%s%s", baseUrl, endpoint)
	return client.Get(url)
}
