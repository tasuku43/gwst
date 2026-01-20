package gitcmd

import "github.com/tasuku43/gwst/internal/infra/output"

func Logf(format string, args ...any) {
	output.Logf("$ "+format, args...)
}
