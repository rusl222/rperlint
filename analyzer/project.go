package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot searches upward from start for go.mod.
func FindProjectRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("get absolute path: %w", err)
	}

	info, err := os.Stat(dir)
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	for {
		goMod := filepath.Join(dir, "go.mod")

		if _, err := os.Stat(goMod); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			break
		}

		dir = parent
	}

	return "", fmt.Errorf(
		"go.mod not found from %q",
		start,
	)
}
