package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
)

type Client interface {
	// GetJSON sends a GET request to the specified URL, unmarshal the JSON response into the
	// provided reference variable.
	GetJSON(url string, v any) error
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type manager struct {
	client httpClient
}

// DefaultClient returns a Client with the default configuration.
func DefaultClient() Client {
	return manager{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (m manager) GetJSON(url string, v any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer closeResponseBody(resp)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return handleInvalidResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err = json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return nil
}

//

// handleInvalidResponse builds an error message that includes the status code and
// response body (if available).
func handleInvalidResponse(resp *http.Response) error {
	errorMessage := fmt.Sprintf("unexpected HTTP status code %d", resp.StatusCode)

	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			errorMessage = fmt.Sprintf("%s: %s", errorMessage, string(body))
		}
	}

	return errors.New(errorMessage)
}

func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}
