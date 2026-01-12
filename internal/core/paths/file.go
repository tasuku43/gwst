package paths

import (
	"fmt"
	"os"
)

// FileExists reports whether the path exists and is a file.
func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("path is a directory: %s", path)
	}
	return true, nil
}
