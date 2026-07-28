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

func readTOMLFile(path string) (map[any]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return toAnyMap(m), nil
}

// BurntSushi/toml produces map[string]any for all tables, but altsrc.NestedVal
// and deepMerge expect map[any]any. Convert at the boundary so the rest of the
// code only has to reason about one map key type.
//
// Note: only map[string]any values are recursively converted. Slice values
// (e.g. TOML arrays of tables) are passed through unchanged.
func toAnyMap(m map[string]any) map[any]any {
	r := make(map[any]any, len(m))
	for k, v := range m {
		if n, ok := v.(map[string]any); ok {
			r[k] = toAnyMap(n)
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
	mergedConfig := make(map[any]any)

	if !nodefaults {
		m, err := readTOMLFile(systemConfigPath)
		if err != nil {
			// System config is optional
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("system config %s: %w", systemConfigPath, err)
			}
		} else {
			deepMerge(mergedConfig, m)
		}
	}

	for _, path := range confFiles {
		m, err := readTOMLFile(path)
		if err != nil {
			return nil, fmt.Errorf("config %s: %w", path, err)
		}
		deepMerge(mergedConfig, m)
	}

	return mergedConfig, nil
}

var configErrOnce sync.Once

type configValueSource struct {
	key string
}

func (vs *configValueSource) Lookup() (string, bool) {
	merged, err := loadConfigs()
	if err != nil {
		// Deduplicate: Lookup() is called per-flag, but we only want one warning.
		configErrOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "warn: %v\n", err)
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

func confWrapper(varName string) cli.ValueSourceChain {
	key := "defaults." + strings.ToLower(varName)
	tomlSource := &configValueSource{key: key}
	return cli.NewValueSourceChain(cli.EnvVar(envPrefix+varName), tomlSource)
}
