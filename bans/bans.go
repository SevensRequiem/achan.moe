package bans

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
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

type OldBans struct {
	ID        uint   `gorm:"primary_key"`
	Status    string `gorm:"default:'inactive'"` // active, expired, deleted
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
	Expires   string `json:"expires"`
}

func init() {
	db := database.DB

	db.AutoMigrate(&Bans{})
	db.AutoMigrate(&OldBans{})
}

func BanIP(c echo.Context) Bans {
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

	logs.Info("Banned IP %s for %s by %s", ip, reason, username)

	return bannedIP
}

func UnbanIP(c echo.Context) Bans {
	id := c.Param("id")
	db := database.DB
	var ban Bans
	db.First(&ban, id)
	db.Where("ID = ?", id).Update("Status", "deleted")
	// Move to OldBans table
	oldBan := OldBans{
		ID:        ban.ID,
		Status:    "deleted",
		IP:        ban.IP,
		Reason:    ban.Reason,
		Username:  ban.Username,
		Timestamp: ban.Timestamp,
		Expires:   ban.Expires,
	}
	db.Create(&oldBan)
	// Delete from Bans table
	db.Delete(&ban)
	logs.Info("Unbanned IP %s", ban.IP)
	return ban
}

func GetTotalBans(c echo.Context) error {
	db := database.DB
	var count int64
	if err := db.Model(&Bans{}).Count(&count).Error; err != nil {
		logs.Error("Error getting total ban count: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, count)
}

func GetBans(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Find(&bans).Error; err != nil {
		logs.Error("Error getting bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansOld(c echo.Context) error {
	db := database.DB
	var oldBans []OldBans
	if err := db.Find(&oldBans).Error; err != nil {
		logs.Error("Error getting old bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, oldBans)
}

func GetBansActive(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Where("Status = ?", "active").Find(&bans).Error; err != nil {
		logs.Error("Error getting active bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansExpired(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Where("Status = ?", "expired").Find(&bans).Error; err != nil {
		logs.Error("Error getting expired bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansDeleted(c echo.Context) error {
	db := database.DB
	var bans []Bans
	if err := db.Where("Status = ?", "deleted").Find(&bans).Error; err != nil {
		logs.Error("Error getting deleted bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBanByIP(c echo.Context) error {
	db := database.DB
	ip := c.Param("ip")
	var bans []Bans
	if err := db.Where("IP = ?", ip).Find(&bans).Error; err != nil {
		logs.Error("Error getting bans by IP: %v", err)
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
					logs.Error("Error parsing time:", err)
					continue
				}
				if expiresTime.After(currentTime) {
					tmpl, err := template.ParseFiles("views/util/banned.html")
					if err != nil {
						logs.Error("Error parsing template:", err)
						return c.String(http.StatusInternalServerError, err.Error())
					}
					data := map[string]interface{}{
						"Pagename":   "Banned",
						"BanReason":  ban.Reason,
						"BanExpires": ban.Expires,
					}
					err = tmpl.Execute(c.Response().Writer, data)
					if err != nil {
						logs.Error("Error executing template:", err)
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
			logs.Error("Error parsing time:", err)
			continue
		}
		if expiresTime.Before(currentTime) && ban.Status == "active" {
			fmt.Println("Ban expired:", ban)
			result := db.Model(&ban).Where("ID = ?", ban.ID).Update("Status", "expired")
			// Move to OldBans table
			oldBan := OldBans{
				ID:        ban.ID,
				Status:    "expired",
				IP:        ban.IP,
				Reason:    ban.Reason,
				Username:  ban.Username,
				Timestamp: ban.Timestamp,
				Expires:   ban.Expires,
			}
			db.Create(&oldBan)
			// Delete from Bans table
			db.Delete(&ban)
			logs.Info("Ban expired: %v", ban)
			// Check for errors

			if result.Error != nil {
				logs.Error("Error expiring ban: %v", result.Error)
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
	logs.Info("Deleted ban: %v", ban)
	return ban
}
