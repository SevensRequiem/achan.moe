package stats

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"

	"achan.moe/bans"
	"achan.moe/board"
	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type Stats struct {
	HDDFree            uint64  `json:"hdd_free"`
	HDDTotal           uint64  `json:"hdd_total"`
	RAMFree            uint64  `json:"ram_free"`
	RAMTotal           uint64  `json:"ram_total"`
	RAMUsage           float64 `json:"ram_usage"`
	CPUUsage           string  `json:"cpu_usage"`
	BinarySum          string  `json:"binary_sum"`
	BinarySize         string  `json:"binary_size"`
	PostCount          string  `json:"post_count"`
	ThreadCount        string  `json:"thread_count"`
	AllTimePostCount   string  `json:"all_time_post_count"`
	AllTimeThreadCount string  `json:"all_time_thread_count"`
	LiveBanCount       string  `json:"live_ban_count"`
	TotalBanCount      string  `json:"total_ban_count"`
}

func GetStats(c echo.Context) error {
	var stats Stats

	// Get free and total HDD space
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}
	diskStat, err := disk.Usage(wd)
	if err != nil {
		return fmt.Errorf("error getting disk usage: %w", err)
	}

	// Get free and total RAM space
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error getting virtual memory: %w", err)
	}

	// Get CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return fmt.Errorf("error getting CPU usage: %w", err)
	}

	// Get binary checksum and size
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %w", err)
	}
	binaryInfo, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("error getting binary info: %w", err)
	}
	binarySize := binaryInfo.Size()

	binaryData, err := ioutil.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("error reading binary file: %w", err)
	}
	hash := sha256.Sum256(binaryData)
	binarySum := hex.EncodeToString(hash[:])

	// Assuming these functions exist and return the required values
	allTimePostCount := board.GetTotalPostCount()
	liveBanCount := bans.GetActiveBanCount()
	totalBanCount := bans.GetTotalBanCount()

	stats.HDDFree = diskStat.Free / (1024 * 1024 * 1024)
	stats.HDDTotal = diskStat.Total / (1024 * 1024 * 1024)
	stats.RAMFree = vmStat.Free / (1024 * 1024 * 1024)
	stats.RAMTotal = vmStat.Total / (1024 * 1024 * 1024)
	stats.RAMUsage = math.Round(vmStat.UsedPercent)
	stats.CPUUsage = fmt.Sprintf("%.2f", cpuPercent[0])
	stats.BinarySum = binarySum
	stats.BinarySize = fmt.Sprintf("%d", binarySize)
	stats.PostCount = "TODO"   // Replace with actual value
	stats.ThreadCount = "TODO" // Replace with actual value
	stats.AllTimePostCount = fmt.Sprintf("%d", allTimePostCount)
	stats.AllTimeThreadCount = "TODO" // Replace with actual value
	stats.LiveBanCount = fmt.Sprintf("%d", liveBanCount)
	stats.TotalBanCount = fmt.Sprintf("%d", totalBanCount)

	return c.JSON(http.StatusOK, stats)
}

func GetBinarySize(c echo.Context) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %w", err)
	}

	binaryInfo, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("error getting binary info: %w", err)
	}

	return c.JSON(http.StatusOK, binaryInfo.Size())
}

func GetBinarySum(c echo.Context) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable path: %w", err)
	}

	binaryData, err := ioutil.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("error reading binary file: %w", err)
	}

	hash := sha256.Sum256(binaryData)
	return c.JSON(http.StatusOK, hex.EncodeToString(hash[:]))
}

func GetContentSize() float64 {
	wd, err := os.Getwd()
	if err != nil {
		return 0
	}

	dir := filepath.Join(wd, "boards")
	var size int64
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		return 0
	}

	sizemb := float64(size) / (1024 * 1024)
	// round to 2 decimal places
	sizemb = math.Round(sizemb*100) / 100
	return sizemb
}
