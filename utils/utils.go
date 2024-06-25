package utils

import (
	"os"
	"path/filepath"
)

// GetProjectSize calculates and returns the size of the project directory in megabytes.
func GetProjectSize(projectPath string) int {
	var totalSize int64

	// Walk through all files in the project directory
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0
	}

	// Convert bytes to megabytes
	sizeInMB := totalSize / (1024 * 1024)

	return int(sizeInMB)
}
