package testutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// KV is a key-value pair for TOML entry construction.
type KV struct {
	Key   string
	Value any
}

// TOMLEntries formats key-value pairs as a TOML [defaults] section.
// Order is preserved as specified.
func TOMLEntries(entries ...KV) string {
	var b strings.Builder
	b.WriteString("[defaults]\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "%s = %v\n", e.Key, formatTomlValue(e.Value))
	}
	return b.String()
}

// TOMLEntry formats a single key-value pair as a TOML [defaults] entry.
func TOMLEntry(key string, value any) string {
	return TOMLEntries(KV{key, value})
}

// formatTomlValue formats a value for TOML output.
// Strings are quoted; other types use %v.
func formatTomlValue(v any) string {
	switch v := v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
