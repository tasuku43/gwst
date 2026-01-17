package gitcmd

import "github.com/tasuku43/gws/internal/core/output"

func Logf(format string, args ...any) {
	output.Logf("$ "+format, args...)
}
