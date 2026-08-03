package testhelpers

import (
	"fmt"
	"strings"
)

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
