package filer

import (
	"fmt"
	"os"
	"path/filepath"
)

func WriteFile(dir, filename string, data []byte) error {
	//nolint: gosec // recorded files must be accessible to unprivileged users
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("make directory: %w", err)
	}

	//nolint: gosec // recorded files must be accessible to unprivileged users
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return fmt.Errorf("write bytes: %w", err)
	}

	return nil
}
