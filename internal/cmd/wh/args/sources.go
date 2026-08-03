package args

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli/v3"
)

// flagSources builds the ValueSourceChain for a flag:
//   1. Environment variable: WORMHOLE_APP_PORT
//   2. One MapSource per TOML file: defaults.app-port
//
// The chain is evaluated in order; the first source that returns a value wins.
// urfave/cli evaluates this chain in PostParse() for flags not set by CLI.
func flagSources(flagName string, srcs ...cli.MapSource) cli.ValueSourceChain {
	envName := strings.ReplaceAll(strings.ToUpper(flagName), "-", "_")
	chain := cli.NewValueSourceChain(cli.EnvVar(envPrefix + envName))
	for _, ms := range srcs {
		chain.Chain = append(chain.Chain,
			cli.NewMapValueSource("defaults."+flagName, ms))
	}
	return chain
}

// TOMLMapSource reads a TOML file and returns a MapSource for use in a
// ValueSourceChain. toml.Unmarshal returns map[string]any; the top-level
// map is converted to map[any]any since MapSource requires it. Nested
// maps are handled transparently by MapSource.Lookup().
func TOMLMapSource(path string) (cli.MapSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseTOML(data, path)
}

// ParseTOML parses TOML data and returns a MapSource for use in a
// ValueSourceChain. The name parameter is used as the MapSource name.
func ParseTOML(data []byte, name string) (cli.MapSource, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}
	return cli.NewMapSource(name, convertTopLevel(raw)), nil
}

// convertTopLevel converts map[string]any to map[any]any. This is required
// by cli.MapSource. Nested maps are handled transparently by
// MapSource.Lookup(), but our single-level [defaults] TOML format does not
// produce nested values at the top level, so recursion is not needed.
func convertTopLevel(m map[string]any) map[any]any {
	r := make(map[any]any, len(m))
	for k, v := range m {
		r[k] = v
	}
	return r
}
