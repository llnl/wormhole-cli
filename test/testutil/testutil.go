package testutil

import (
	"bytes"
	"io"
	"net/http"
)

const (
	TestToken       = "4745a904-81a0-49ca-b764-37bb4db9bb2c.Y1a2Y3ZTZ18BDaKqm2YUwmIAe78r1D2Fp-jO1gOsVao"
	TestEndpointUrl = "http://routeregistry.test"
)

// NewMockResponse creates a simple http.Response for testing purposes.
func NewMockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     make(http.Header),
	}
}
