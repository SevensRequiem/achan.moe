package bans

import (
	"fmt"
	"net/http"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/labstack/echo/v4"
)

type Bans struct {
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
	Expires   string `json:"expires"`
}

func init() {
	db := database.DB

	db.AutoMigrate(&Bans{})
}

func BanIP(c echo.Context) Bans {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	ip := c.FormValue("ip")
	reason := c.FormValue("reason")
	username := c.FormValue("username")
	timestamp := time.Now().Format(time.RFC3339)
	expires := c.FormValue("expires")

	bannedIP := Bans{
		IP:        ip,
		Reason:    reason,
		Username:  username,
		Timestamp: timestamp,
		Expires:   expires,
	}
	db := database.DB

	db.Create(&bannedIP)

	return bannedIP
}

func GetBans(c echo.Context) []Bans {
	db := database.DB

	var bans []Bans
	db.Find(&bans)

	return bans
}

func BanMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		db := database.DB

		var bans []Bans
		db.Find(&bans)
		currentTime := time.Now()
		for _, ban := range bans {
			if ban.IP == c.RealIP() {
				expiresTime, err := time.Parse("2006-01-02", ban.Expires)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					continue
				}
				if expiresTime.After(currentTime) {
					return c.String(http.StatusForbidden, "You are banned :3, reason: "+ban.Reason+" expires: "+ban.Expires)
				}
			}
		}
		return next(c)
	}
}