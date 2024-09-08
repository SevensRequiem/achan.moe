package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type SystemInfo struct {
	CurrentWorkingDirectory string
	UserHomeDirectory       string
	Hostname                string
	SystemPageSize          int
	SystemArchitecture      string
	NumCPU                  int
	Memory                  uint64
	HDD                     uint64
	OperatingSystem         string
	GoVersion               string
	GoRootDirectory         string
}

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

func GetSystemInfo() (SystemInfo, error) {
	var info SystemInfo
	var err error

	// Get the current working directory
	info.CurrentWorkingDirectory, err = os.Getwd()
	if err != nil {
		return info, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Get the user's home directory
	info.UserHomeDirectory, err = os.UserHomeDir()
	if err != nil {
		return info, fmt.Errorf("failed to get user's home directory: %w", err)
	}

	// Get the hostname of the system
	info.Hostname, err = os.Hostname()
	if err != nil {
		return info, fmt.Errorf("failed to get hostname: %w", err)
	}

	// Get the system page size
	info.SystemPageSize = os.Getpagesize()

	// Get the system architecture
	info.SystemArchitecture = runtime.GOARCH

	// Get the number of CPUs
	info.NumCPU = runtime.NumCPU()

	// Get the memory
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return info, fmt.Errorf("error getting virtual memory: %w", err)
	}

	info.Memory = vmStat.Total / (1024 * 1024 * 1024)

	// Get the HDD
	diskStat, err := disk.Usage("/")
	if err != nil {
		return info, fmt.Errorf("error getting disk usage: %w", err)
	}

	info.HDD = diskStat.Total / (1024 * 1024 * 1024)

	// Get the operating system
	info.OperatingSystem = runtime.GOOS

	// Get the Go version
	info.GoVersion = runtime.Version()

	// Get the Go root directory
	info.GoRootDirectory = runtime.GOROOT()

	return info, nil
}
