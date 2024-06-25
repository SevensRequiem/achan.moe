package bans

import (
	"net/http"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/labstack/echo/v4"
)

type Bans struct {
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	Username  string `json:"username"`
	Admin     string `json:"admin"`
	Timestamp string `json:"timestamp"`
}

func init() {
	db := database.Connect()
	defer database.Close()
	db.AutoMigrate(&Bans{})
}

func BanIP(c echo.Context, ip string, reason string, username string, admin string, timestamp string) Bans {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	bannedIP := Bans{
		IP:        ip,
		Reason:    reason,
		Username:  username,
		Admin:     admin,
		Timestamp: timestamp,
	}
	return bannedIP
}
