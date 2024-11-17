package bans

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

type Bans struct {
	ID        string `bson:"_id,omitempty"`        // MongoDB uses _id as the primary key
	Status    string `bson:"status" json:"status"` // active, expired, deleted
	IP        string `bson:"ip" json:"ip"`
	Reason    string `bson:"reason" json:"reason"`
	Username  string `bson:"username" json:"username"`
	Timestamp string `bson:"timestamp" json:"timestamp"`
	Expires   string `bson:"expires" json:"expires"`
}

type OldBans struct {
	ID        string `bson:"_id,omitempty"`        // MongoDB uses _id as the primary key
	Status    string `bson:"status" json:"status"` // active, expired, deleted
	IP        string `bson:"ip" json:"ip"`
	Reason    string `bson:"reason" json:"reason"`
	Username  string `bson:"username" json:"username"`
	Timestamp string `bson:"timestamp" json:"timestamp"`
	Expires   string `bson:"expires" json:"expires"`
}

func BanIP(c echo.Context) Bans {
	ip := c.FormValue("ip")
	reason := c.FormValue("reason")
	username := c.FormValue("username")
	timestamp := time.Now().Format(time.RFC3339)
	expires := c.FormValue("expires")

	// Parse the expires date to ensure it's in the correct format
	expiresTime, err := time.Parse("2006-01-02", expires)
	if err != nil {
		logs.Error("Error parsing expires date: %v", err)
		expiresTime = time.Now().Add(24 * time.Hour) // Default to 24 hours if parsing fails
	}
	expires = expiresTime.Format(time.RFC3339)

	bannedIP := Bans{
		IP:        ip,
		Status:    "active",
		Reason:    reason,
		Username:  username,
		Timestamp: timestamp,
		Expires:   expires,
	}
	db := database.DB_Main.Collection("bans")
	db.InsertOne(context.Background(), bannedIP)

	logs.Info("Banned IP %s for %s by %s", ip, reason, username)

	return bannedIP
}

func UnbanIP(c echo.Context) Bans {
	id := c.Param("id")
	db := database.DB_Main.Collection("bans")
	var ban Bans
	db.FindOne(context.Background(), bson.M{"id": id}).Decode(&ban)
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
	// Delete from Bans table
	db.DeleteOne(context.Background(), bson.M{"id": id})

	// add to old bans
	olddb := database.DB_Main.Collection("old_bans")
	olddb.InsertOne(context.Background(), oldBan)
	logs.Info("Unbanned IP %s", ban.IP)
	return ban
}

func GetTotalBans(c echo.Context) error {
	totalBans := GetTotalBanCount()
	return c.JSON(http.StatusOK, totalBans)
}

func GetBans(c echo.Context) error {
	db := database.DB_Main.Collection("bans")
	var bans []Bans
	cur, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error getting bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban Bans
		err := cur.Decode(&ban)
		if err != nil {
			logs.Error("Error decoding ban: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		bans = append(bans, ban)
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansOld(c echo.Context) error {
	db := database.DB_Main.Collection("old_bans")
	var bans []OldBans
	cur, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error getting old bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban OldBans
		err := cur.Decode(&ban)
		if err != nil {
			logs.Error("Error decoding old ban: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		bans = append(bans, ban)
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansActive(c echo.Context) error {
	db := database.DB_Main.Collection("bans")
	var bans []Bans
	cur, err := db.Find(context.Background(), bson.M{"status": "active"})
	if err != nil {
		logs.Error("Error getting active bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban Bans
		err := cur.Decode(&ban)
		if err != nil {
			logs.Error("Error decoding active ban: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		bans = append(bans, ban)
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansExpired(c echo.Context) error {
	db := database.DB_Main.Collection("old_bans")
	var bans []Bans
	cur, err := db.Find(context.Background(), bson.M{"status": "expired"})
	if err != nil {
		logs.Error("Error getting expired bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban Bans
		err := cur.Decode(&ban)
		if err != nil {
			logs.Error("Error decoding expired ban: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		bans = append(bans, ban)
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBansDeleted(c echo.Context) error {
	db := database.DB_Main.Collection("old_bans")
	var bans []Bans
	cur, err := db.Find(context.Background(), bson.M{"status": "deleted"})
	if err != nil {
		logs.Error("Error getting deleted bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban Bans
		err := cur.Decode(&ban)
		if err != nil {
			logs.Error("Error decoding deleted ban: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		bans = append(bans, ban)
	}
	return c.JSON(http.StatusOK, bans)
}

func GetBanByIP(c echo.Context) error {
	ip := c.Param("ip")
	db := database.DB_Main.Collection("bans")
	var ban Bans
	err := db.FindOne(context.Background(), bson.M{"ip": ip}).Decode(&ban)
	if err != nil {
		logs.Error("Error getting ban by IP: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, ban)
}

func GetTotalBanCount() int64 {
	db := database.DB_Main.Collection("bans")
	count, err := db.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error counting bans: %v", err)
	}
	return count
}

func GetActiveBanCount() int64 {
	db := database.DB_Main.Collection("bans")
	count, err := db.CountDocuments(context.Background(), bson.M{"status": "active"})
	if err != nil {
		logs.Error("Error counting active bans: %v", err)
	}
	return count
}

func BanMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		db := database.DB_Main.Collection("bans")
		var bans []Bans
		cur, err := db.Find(context.Background(), bson.M{"status": "active"})
		if err != nil {
			logs.Error("Error getting active bans: %v", err)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		defer cur.Close(context.Background())
		for cur.Next(context.Background()) {
			var ban Bans
			err := cur.Decode(&ban)
			if err != nil {
				logs.Error("Error decoding ban: %v", err)
				return c.String(http.StatusInternalServerError, err.Error())
			}
			bans = append(bans, ban)
		}

		currentTime := time.Now()
		clientIP := c.RealIP()
		cfConnectingIP := c.Request().Header.Get("CF-Connecting-IP")
		logs.Info("Client IP: %s, CF-Connecting-IP: %s", clientIP, cfConnectingIP)

		for _, ban := range bans {
			if ban.IP == clientIP || ban.IP == cfConnectingIP {
				expiresTime, err := time.Parse(time.RFC3339, ban.Expires)
				if err != nil {
					logs.Error("Error parsing time: %v", err)
					continue
				}
				if expiresTime.After(currentTime) {
					tmpl, err := template.ParseFiles("views/util/banned.html")
					if err != nil {
						logs.Error("Error parsing template: %v", err)
						return c.String(http.StatusInternalServerError, err.Error())
					}
					data := map[string]interface{}{
						"Pagename":   "Banned",
						"BanReason":  ban.Reason,
						"BanExpires": ban.Expires,
					}
					err = tmpl.Execute(c.Response().Writer, data)
					if err != nil {
						logs.Error("Error executing template: %v", err)
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
	db := database.DB_Main.Collection("bans")
	var bans []Bans
	cur, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error getting bans: %v", err)
		return
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban Bans
		err := cur.Decode(&ban)
		if err != nil {
			logs.Error("Error decoding ban: %v", err)
			return
		}
		bans = append(bans, ban)
	}

	currentTime := time.Now()
	for _, ban := range bans {
		expiresTime, err := time.Parse("2006-01-02", ban.Expires)
		if err != nil {
			logs.Error("Error parsing time:", err)
			continue
		}
		if expiresTime.Before(currentTime) && ban.Status == "active" {
			fmt.Println("Ban expired:", ban)
			db.DeleteOne(context.Background(), bson.M{"id": ban.ID})
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
			// add to old bans
			olddb := database.DB_Main.Collection("old_bans")
			olddb.InsertOne(context.Background(), oldBan)
			// Delete from Bans table
			db.DeleteOne(context.Background(), bson.M{"id": ban.ID})
			logs.Info("Ban expired: %v", ban)

		}
	}
}

func DeleteBan(c echo.Context) Bans {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	id := c.Param("id")
	db := database.DB_Main.Collection("bans")
	var ban Bans
	db.FindOne(context.Background(), bson.M{"id": id}).Decode(&ban)
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
	// Delete from Bans table
	db.DeleteOne(context.Background(), bson.M{"id": id})

	// add to old bans
	olddb := database.DB_Main.Collection("old_bans")
	olddb.InsertOne(context.Background(), oldBan)
	logs.Info("Deleted ban %s", ban.IP)
	return ban
}
