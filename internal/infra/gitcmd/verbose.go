package gitcmd

import "github.com/tasuku43/gion/internal/infra/output"

func Logf(format string, args ...any) {
	output.Logf("$ "+format, args...)
}
