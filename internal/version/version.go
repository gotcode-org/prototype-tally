package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Branch  = "unknown"
)

func GetVersionString() string {
	return fmt.Sprintf("Tally %s (branch: %s, commit: %s)", Version, Branch, Commit)
}

func GetDepsString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "Dependency info not available."
	}
	
	var sb strings.Builder
	sb.WriteString("Dependencies:\n")
	for _, dep := range info.Deps {
		sb.WriteString(fmt.Sprintf("  - %s @ %s\n", dep.Path, dep.Version))
	}
	return sb.String()
}
