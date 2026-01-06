package gitcmd

import (
	"fmt"
	"os"

	"github.com/tasuku43/gws/internal/output"
)

var verbose bool

func SetVerbose(v bool) {
	verbose = v
}

func IsVerbose() bool {
	return verbose
}

func Logf(format string, args ...any) {
	if verbose {
		return
	}
	if output.HasStepLogger() {
		output.Logf("$ "+format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, "%s$ "+format+"\n", append([]any{output.Indent}, args...)...)
}
