package bans

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/labstack/echo/v4"
)

type Bans struct {
	ID        uint   `gorm:"primary_key"`
	Status    string `gorm:"default:'active'"` // active, expired, deleted
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

func GetTotalBans(c echo.Context) error {
	db := database.DB
	var count int64
	if err := db.Model(&Bans{}).Count(&count).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, count)
}

func GetBans(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Find(&bans).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansActive(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Where("Status = ?", "active").Find(&bans).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansExpired(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Where("Status = ?", "expired").Find(&bans).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansDeleted(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Where("Status = ?", "deleted").Find(&bans).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBanByIP(c echo.Context) error {
	db := database.DB
	ip := c.Param("ip")
	var bans []Bans
	if err := db.Where("IP = ?", ip).Find(&bans).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetTotalBanCount() int64 {
	db := database.DB
	var count int64
	db.Model(&Bans{}).Count(&count)
	return count
}

func GetActiveBanCount() int64 {
	db := database.DB
	var count int64
	db.Model(&Bans{}).Where("Status = ?", "active").Count(&count)
	return count
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
					tmpl, err := template.ParseFiles("views/util/banned.html")
					if err != nil {
						return c.String(http.StatusInternalServerError, err.Error())
					}
					data := map[string]interface{}{
						"Pagename":   "Banned",
						"BanReason":  ban.Reason,
						"BanExpires": ban.Expires,
					}
					err = tmpl.Execute(c.Response().Writer, data)
					if err != nil {
						fmt.Println("Error executing template:", err)
						return c.String(http.StatusInternalServerError, err.Error())
					}
					return nil
				}
			}
		}
		return next(c)
	}
}

func ExpireCheck() {
	fmt.Println("Checking for expired bans...")
	db := database.DB
	var bans []Bans
	db.Find(&bans)
	currentTime := time.Now()
	for _, ban := range bans {
		expiresTime, err := time.Parse("2006-01-02", ban.Expires)
		if err != nil {
			fmt.Println("Error parsing time:", err)
			continue
		}
		if expiresTime.Before(currentTime) && ban.Status == "active" {
			fmt.Println("Ban expired:", ban)
			result := db.Model(&ban).Where("ID = ?", ban.ID).Update("Status", "expired")
			if result.Error != nil {
				fmt.Println("Error updating status:", result.Error)
			} else if result.RowsAffected == 0 {
				fmt.Println("No rows were updated. Check if the ID is correct:", ban.ID)
			} else {
				fmt.Println("Status updated to 'expired' for ban ID:", ban.ID)
			}
		}
	}
}

func DeleteBan(c echo.Context) Bans {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	id := c.Param("id")
	db := database.DB
	var ban Bans
	db.First(&ban, id)
	db.Where("ID = ?", id).Update("Status", "deleted")
	return ban
}
