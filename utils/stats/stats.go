package stats

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"achan.moe/bans"
	"achan.moe/board"
	"achan.moe/database"
	"achan.moe/models"
	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"go.mongodb.org/mongo-driver/bson"
)

var startTime = time.Now()

func init() {
	startTime = time.Now()
	SetContentSizetoDB()
}

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
	TotalSize          string  `json:"total_size"`
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

func ReturnContentSizeFromDB(c echo.Context) error {
	db := database.DB_Main.Collection("stats")
	cursor, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())

	var totalSize float64
	for cursor.Next(context.Background()) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Fatal(err)
		}
		totalSize = doc["total_size"].(float64)
	}
	return c.JSON(http.StatusOK, totalSize)
}

func SetContentSizetoDB() {
	db := database.DB_Main.Collection("stats")
	_, err := db.InsertOne(context.Background(), bson.M{"total_size": GetContentSize()})
	if err != nil {
		log.Fatal(err)
	}
}

func GetContentSize() float64 {
	boardsdb := database.DB_Main.Collection("boards")
	cursor, err := boardsdb.Find(context.Background(), bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())

	var totalSize float64
	for cursor.Next(context.Background()) {
		var board models.Board
		if err := cursor.Decode(&board); err != nil {
			log.Fatal(err)
		}

		boardDB := database.Client.Database(board.BoardID)
		collections, err := boardDB.ListCollectionNames(context.Background(), bson.M{})
		if err != nil {
			log.Fatal(err)
		}

		for _, collection := range collections {
			if collection == "banners" || collection == "thumbs" || collection == "images" {
				continue
			}
			col := boardDB.Collection(collection)
			docsCursor, err := col.Find(context.Background(), bson.M{})
			if err != nil {
				log.Fatal(err)
			}
			defer docsCursor.Close(context.Background())

			for docsCursor.Next(context.Background()) {
				var doc bson.M
				if err := docsCursor.Decode(&doc); err != nil {
					log.Fatal(err)
				}
				docBytes, err := bson.Marshal(doc)
				if err != nil {
					log.Fatal(err)
				}
				docSize := len(docBytes)
				totalSize += float64(docSize)
			}
		}
	}
	totalSize = totalSize / 1024 / 1024         // Convert to MB
	totalSize = math.Round(totalSize*100) / 100 // Round to 2 decimal places
	return totalSize
}

func ServerStatus(c echo.Context) error {
	status := "OK"
	servertime := time.Now().Format("15:04:05")
	uptime := uptime() // No need to call String() on uptime

	response := map[string]string{
		"status":      status,
		"server_time": servertime,
		"uptime":      uptime,
	}

	return c.JSON(http.StatusOK, response)
}

func uptime() string {
	duration := time.Since(startTime).Round(time.Second)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
