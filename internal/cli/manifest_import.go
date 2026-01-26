package cli

import (
	"context"

	"github.com/tasuku43/gion/internal/app/manifestimport"
)

func rebuildManifest(ctx context.Context, rootDir string) error {
	_, err := manifestimport.Import(ctx, rootDir)
	return err
}
