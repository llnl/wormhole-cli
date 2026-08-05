package testutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/BurntSushi/toml"
)

const (
	TestToken       = "4745a904-81a0-49ca-b764-37bb4db9bb2c.Y1a2Y3ZTZ18BDaKqm2YUwmIAe78r1D2Fp-jO1gOsVao"
	TestEndpointUrl = "http://routeregistry.test"
	DefaultTable    = "test"
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
	Table string
}

// TOMLConfig generates TOML for a table with the given name and key-value pairs.
func TOMLConfig(table string, kvs ...KV) string {
	for i := range kvs {
		if kvs[i].Table == "" {
			kvs[i].Table = table
		}
	}
	tables := buildTableMap(kvs...)
	data, err := toml.Marshal(tables)
	if err != nil {
		panic(fmt.Sprintf("toml.Marshal failed: %v", err))
	}
	return string(data)
}

// buildTableMap groups entries by table, applying DefaultTable fallback.
func buildTableMap(entries ...KV) map[string]map[string]any {
	tables := make(map[string]map[string]any)
	for _, e := range entries {
		table := e.Table
		if table == "" {
			table = DefaultTable
		}
		if tables[table] == nil {
			tables[table] = make(map[string]any)
		}
		tables[table][e.Key] = e.Value
	}
	return tables
}
