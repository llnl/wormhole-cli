// Package version maintains requirements for --version output.
package version

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

var (
	version   string
	gitCommit string
	gitBranch string
	goVersion string
	buildDate string
)

func GetVersion() string {
	return version
}

func Printer(cCmd *cli.Command) {
	var sb strings.Builder

	sb.WriteString("Version: " + cCmd.Version + "\n")

	if gitCommit != "" {
		sb.WriteString("Git Commit: " + gitCommit + "\n")
	}

	if gitBranch != "" {
		sb.WriteString("Git Branch: " + gitBranch + "\n")
	}

	sb.WriteString("Go Version: " + goVersion + "\n")
	sb.WriteString("Built: " + buildDate)

	fmt.Println(sb.String())
}
