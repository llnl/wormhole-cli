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
			switch s := f.(type) {
			case *cli.StringFlag:
				assert.NotNil(t, s.Sources)
				assert.IsType(t, cli.ValueSourceChain{}, s.Sources)
			case *cli.BoolFlag:
				assert.NotNil(t, s.Sources)
				assert.IsType(t, cli.ValueSourceChain{}, s.Sources)
			}
		})
	}
}

func TestOpenFlags(t *testing.T) {
	a := &CLIArgs{}
	flags := OpenFlags(a)

	t.Run("count", func(t *testing.T) {
		assert.Len(t, flags, 11)
	})

	t.Run("names", func(t *testing.T) {
		var names []string
		for _, f := range flags {
			names = append(names, f.Names()...)
		}
		expected := []string{
			"name", "community", "app-port", "podman-compat",
			"allowed-users", "allowed-groups", "forbidden-users", "forbidden-groups",
			"forwarded-header-user", "forwarded-header-groups", "auth-bearer-header",
		}
		for _, n := range expected {
			assert.Contains(t, names, n, "missing flag: %s", n)
		}
	})

	t.Run("properties", func(t *testing.T) {
		flags := OpenFlags(a)

		name := findFlag(t, flags, "name").(*cli.StringFlag)
		community := findFlag(t, flags, "community").(*cli.StringFlag)
		appPort := findFlag(t, flags, "app-port").(*cli.StringFlag)
		podmanCompat := findFlag(t, flags, "podman-compat").(*cli.BoolFlag)
		authBearer := findFlag(t, flags, "auth-bearer-header").(*cli.BoolFlag)

		assert.True(t, name.Required)
		assert.False(t, community.Required)
		assert.False(t, appPort.Required)
		assert.Equal(t, DefaultAppPort, appPort.Value)
		assert.True(t, podmanCompat.Hidden)
		assert.True(t, authBearer.Hidden)
		assert.Equal(t, categoryTunnel, name.Category)
		assert.Equal(t, categoryTunnel, podmanCompat.Category)
	})

	t.Run("destinations", func(t *testing.T) {
		flags := OpenFlags(a)

		checkDest := func(name string, expected any) {
			f := findFlag(t, flags, name)
			switch f := f.(type) {
			case *cli.StringFlag:
				assert.Equal(t, expected, f.Destination, "destination for %s", name)
			case *cli.BoolFlag:
				assert.Equal(t, expected, f.Destination, "destination for %s", name)
			}
		}

		checkDest("name", &a.Open.Name)
		checkDest("community", &a.Open.Community)
		checkDest("app-port", &a.Open.AppPort)
		checkDest("podman-compat", &a.Open.PodmanCompat)
		checkDest("allowed-users", &a.Open.AllowedUsers)
		checkDest("allowed-groups", &a.Open.AllowedGroups)
		checkDest("forbidden-users", &a.Open.ForbiddenUsers)
		checkDest("forbidden-groups", &a.Open.ForbiddenGroups)
		checkDest("forwarded-header-user", &a.Open.ForwardedHeaderUser)
		checkDest("forwarded-header-groups", &a.Open.ForwardedHeaderGroups)
		checkDest("auth-bearer-header", &a.Open.AuthBearerHeader)
	})

	t.Run("nameValidator", func(t *testing.T) {
		flags := OpenFlags(a)
		nameFlag := findFlag(t, flags, "name").(*cli.StringFlag)

		assert.NotNil(t, nameFlag.Validator)
		assert.Error(t, nameFlag.Validator("my_app"))
		assert.NoError(t, nameFlag.Validator("my-app"))
	})
}
