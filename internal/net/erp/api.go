package erp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type HttpRequestConfig struct {
	BaseUrl  string
	Endpoint string
	Timeout  time.Duration
}

func ConfigDefaultWithEndpoint(endpoint string) HttpRequestConfig {
	return HttpRequestConfig{
		Endpoint: endpoint,
		BaseUrl:  ENDPOINT_DEFAULT_BASE_URL,
		Timeout:  DEFAULT_HTTP_TIMEOUT,
	}
}

// Post sends a POST request to the ERP system at the given endpoint
// with the provided form data. It returns an error if the request fails.
// Form data is sent as x-www-form-urlencoded.
//
// Context must be provided with values for
// - KEY_ERP_URL (erp base url - string)
// - KEY_HTTP_TIMEOUT (timeout for client request - time.Duration)
func Post(ctx context.Context, config HttpRequestConfig, formData url.Values) error {
	client := http.Client{
		Timeout: config.Timeout,
	}

	url := fmt.Sprintf("%s%s", config.BaseUrl, config.Endpoint)
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

// Get sends a GET request to the ERP system at the given endpoint.
// It returns the response and an error if the request fails.
func Get(ctx context.Context, config HttpRequestConfig) (*http.Response, error) {
	client := http.Client{
		Timeout: config.Timeout,
	}

	url := fmt.Sprintf("%s%s", config.BaseUrl, config.Endpoint)
	return client.Get(url)
}
