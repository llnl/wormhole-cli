package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func Test_Version(t *testing.T) {
	version = "1.1"
	goVersion = "go1"
	buildDate = "2020"
	gitCommit = "abc"
	gitBranch = "test"

	cCmd := &cli.Command{
		Version: version,
	}

	t.Run("verify Version()", func(t *testing.T) {
		assert.Equal(t, "1.1", GetVersion())
	})

	t.Run("run Printer()", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Printer(cCmd)
		})
	})
}
