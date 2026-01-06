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
	fmt.Fprintf(os.Stderr, "\x1b[36m%s$ "+format+"\x1b[0m\n", append([]any{output.Indent}, args...)...)
}
