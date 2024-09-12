package minecraft

// query minecraft server status

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"achan.moe/utils/config"

	"github.com/dreamscached/minequery/v2"
	"github.com/labstack/echo/v4"
)

var globalConfig config.GlobalConfig
var currentStatus *ServerStatus // Use a pointer to handle nil checks

func init() {
	// Initialize globalConfig by reading the global configuration
	globalConfig = config.ReadGlobalConfig()

	// Ensure globalConfig is initialized
	if globalConfig.MinecraftIP == "" {
		fmt.Println("Minecraft server address is not configured")
		return
	}

	// Fetch the server status and update currentStatus
	status, err := GetServerStatus()
	if err != nil {
		fmt.Println("Error fetching server status:", err)
		return
	}
	currentStatus = status
}

type ServerStatus struct {
	Version     string
	Players     int
	MaxPlayers  int
	Description string
}

func GetServerStatus() (*ServerStatus, error) {
	globalConfig := config.ReadGlobalConfig()

	// Set a timeout for the ping operation
	timeout := 10 * time.Second
	pinger := minequery.NewPinger(
		minequery.WithTimeout(timeout),
		minequery.WithUseStrict(true),
		minequery.WithProtocolVersion17(minequery.Ping17ProtocolVersion172),
	)

	res, err := pinger.Ping17(globalConfig.MinecraftIP, globalConfig.MinecraftPort)
	if err != nil {
		return nil, fmt.Errorf("error querying server: %w", err)
	}

	// Populate the ServerStatus struct with the response data
	status := &ServerStatus{
		Version:     res.VersionName,
		Players:     res.OnlinePlayers,
		MaxPlayers:  res.MaxPlayers,
		Description: res.Description.String(),
	}
	fmt.Println("Fetched MC server status")
	return status, nil
}

func JSONStatus(c echo.Context) error {
	// Check if currentStatus is nil
	if currentStatus == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Server status not available"})
	}

	// Use the already loaded ServerStatus struct
	status := currentStatus

	jsonData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return c.JSONBlob(http.StatusOK, jsonData)
}
