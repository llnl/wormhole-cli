package args

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func findFlag(t *testing.T, flags []cli.Flag, name string) cli.Flag {
	t.Helper()
	for _, f := range flags {
		for _, n := range f.Names() {
			if n == name {
				return f
			}
		}
	}
	t.Fatalf("flag %q not found", name)
	return nil
}

func TestGlobalFlags(t *testing.T) {
	a := &CLIArgs{}
	flags := GlobalFlags(a)

	assert.Len(t, flags, 3)

	type flagCheck struct {
		name        string
		value       string
		required    bool
		trimSpace   bool
		category    string
		destination any
	}
	checks := []flagCheck{
		{"endpoint", DefaultEndpoint, true, true, categoryGeneral, &a.Global.Endpoint},
		{"token", DefaultToken, true, true, categoryGeneral, &a.Global.Token},
		{"verbose", "false", false, false, categoryGeneral, &a.Global.Verbose},
	}
	for _, c := range checks {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := findFlag(t, flags, c.name)
			switch f := f.(type) {
			case *cli.StringFlag:
				assert.Equal(t, c.value, f.Value)
				assert.Equal(t, c.required, f.Required)
				assert.Equal(t, c.trimSpace, f.Config.TrimSpace)
				assert.Equal(t, c.category, f.Category)
				assert.Equal(t, c.destination, f.Destination)
			case *cli.BoolFlag:
				assert.Equal(t, c.required, f.Required)
				assert.Equal(t, c.category, f.Category)
				assert.Equal(t, c.destination, f.Destination)
			}
		})
	}

	t.Run("all flags have ValueSourceChain sources", func(t *testing.T) {
		for _, f := range flags {
			switch f := f.(type) {
			case *cli.StringFlag:
				assert.NotNil(t, f.Sources)
				assert.IsType(t, cli.ValueSourceChain{}, f.Sources)
			case *cli.BoolFlag:
				assert.NotNil(t, f.Sources)
				assert.IsType(t, cli.ValueSourceChain{}, f.Sources)
			}
		}
	})
}

func TestOpenFlags(t *testing.T) {
	a := &CLIArgs{}
	flags := OpenFlags(a)

	t.Run("all flags have Sources", func(t *testing.T) {
		for _, f := range flags {
			switch f := f.(type) {
			case *cli.StringFlag:
				assert.NotNil(t, f.Sources, "flag %q should have Sources", f.Name)
			case *cli.BoolFlag:
				assert.NotNil(t, f.Sources, "flag %q should have Sources", f.Name)
			}
		}
	})

	t.Run("all flags have Category", func(t *testing.T) {
		for _, f := range flags {
			switch f := f.(type) {
			case *cli.StringFlag:
				assert.NotEmpty(t, f.Category, "flag %q should have a Category", f.Name)
			case *cli.BoolFlag:
				assert.NotEmpty(t, f.Category, "flag %q should have a Category", f.Name)
			}
		}
	})

	t.Run("string flags have TrimSpace", func(t *testing.T) {
		for _, f := range flags {
			if sf, ok := f.(*cli.StringFlag); ok {
				assert.True(t, sf.Config.TrimSpace, "string flag %q should have TrimSpace", sf.Name)
			}
		}
	})

	t.Run("name flag validator", func(t *testing.T) {
		var nameFlag *cli.StringFlag
		for _, f := range flags {
			if sf, ok := f.(*cli.StringFlag); ok && sf.Name == nameName {
				nameFlag = sf
				break
			}
		}
		assert.NotNil(t, nameFlag)
		assert.NotNil(t, nameFlag.Validator)
		assert.Error(t, nameFlag.Validator("my_app"))
		assert.NoError(t, nameFlag.Validator("my-app"))
	})
}
