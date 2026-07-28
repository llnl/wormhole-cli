package wh

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"
)

var (
	confFiles        []string
	nodefaults       bool
	systemConfigPath = "/etc/wormhole/cli/wh.toml"
)

func confWrapper(varName string) cli.ValueSourceChain {
	key := "defaults." + strings.ToLower(varName)
	tomlSource := &configValueSource{key: key}
	return cli.NewValueSourceChain(cli.EnvVar(envPrefix+varName), tomlSource)
}

func readTOMLFile(path string) (map[any]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	// Convert map[string]any -> map[any]any at the boundary.
	// Only map values are recursively converted; slices pass through unchanged.
	return convertMap(raw), nil
}

// convertMap recursively converts map[string]any to map[any]any.
func convertMap(m map[string]any) map[any]any {
	r := make(map[any]any, len(m))
	for k, v := range m {
		if n, ok := v.(map[string]any); ok {
			r[k] = convertMap(n)
		} else {
			r[k] = v
		}
	}
	return r
}

// Map values are merged recursively; all other values (including slices) in
// src overwrite those in dst.
func deepMerge(dst, src map[any]any) {
	for k, v := range src {
		if srcVal, ok := v.(map[any]any); ok {
			if dstVal, ok := dst[k].(map[any]any); ok {
				deepMerge(dstVal, srcVal)
				continue
			}
		}
		dst[k] = v
	}
}

func loadConfigs() (map[any]any, error) {
	merged := make(map[any]any)

	if !nodefaults {
		if m, err := readTOMLFile(systemConfigPath); err != nil {
			// System config is optional: missing file is not an error,
			// but parse errors and permission failures are fatal.
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("system config %s: %w", systemConfigPath, err)
			}
		} else {
			deepMerge(merged, m)
		}
	}

	for _, path := range confFiles {
		m, err := readTOMLFile(path)
		if err != nil {
			return nil, fmt.Errorf("config %s: %w", path, err)
		}
		deepMerge(merged, m)
	}

	return merged, nil
}

var configErrOnce sync.Once

type configValueSource struct {
	key string
}

func (vs *configValueSource) Lookup() (string, bool) {
	merged, err := loadConfigs()
	if err != nil {
		// Deduplicate: Lookup() is called per-flag, but we only want one error printed.
		configErrOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		})
		return "", false
	}
	if v, ok := altsrc.NestedVal(vs.key, merged); ok {
		return fmt.Sprint(v), true
	}
	return "", false
}

func (vs *configValueSource) String() string {
	return fmt.Sprintf("config file key %[1]q", vs.key)
}

func (vs *configValueSource) GoString() string {
	return fmt.Sprintf("&configValueSource{keyPath:%[1]q}", vs.key)
}
