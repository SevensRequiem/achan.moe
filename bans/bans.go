package bans

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/cache"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/valkey-io/valkey-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var Client = cache.ClientBans
var ctx = context.Background()

func GetBansFromDB() []models.Bans {
	db := database.DB_Main.Collection("bans")
	var bans []models.Bans
	cur, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		return nil
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban models.Bans
		err := cur.Decode(&ban)
		if err != nil {
			return nil
		}
		bans = append(bans, ban)
	}
	return bans
}

func ManualBanIP(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return nil
	}

	userSessionValue, ok := sess.Values["user"]
	if !ok {
		return nil
	}

	user, ok := userSessionValue.(models.User)
	if !ok {
		return nil
	}
	ip := c.FormValue("ip")
	reason := c.FormValue("reason")
	userID := user.ID
	timestamp := time.Now().Format(time.RFC3339)
	expires := c.FormValue("expires")

	expiresTime, err := time.Parse("2006-01-02", expires)
	if err != nil {
		logs.Error("Error parsing expires date: %v", err)
		expiresTime = time.Now().Add(24 * time.Hour)
	}
	expires = expiresTime.Format(time.RFC3339)
	ID := primitive.NewObjectID()
	bannedIP := models.Bans{
		ID:        ID,
		IP:        ip,
		Status:    "active",
		Reason:    reason,
		Username:  user.Username,
		UserID:    userID,
		Timestamp: timestamp,
		Expires:   expires,
	}
	db := database.DB_Main.Collection("bans")
	db.InsertOne(context.Background(), bannedIP)
	// Set the ban in Redis
	bannedIPdata, err := json.Marshal(bannedIP)
	if err != nil {
		logs.Error("Error marshalling banned IP data: %v", err)
		return nil
	}
	err = cache.ClientBans.Do(ctx, cache.ClientBans.B().Set().Key(ip).Value(string(bannedIPdata)).Build()).Error()
	if err != nil {
		logs.Error("Error setting banned IP in Redis: %v", err)
		return nil
	}
	logs.Info("Banned IP %s for %s by %s", ip, reason, userID)

	return nil
}

func UnbanIP(c echo.Context) error {
	ip := c.QueryParam("ip")
	var ban models.Bans
	err := database.DB_Main.Collection("bans").FindOne(context.Background(), bson.M{"ip": ip}).Decode(&ban)
	if err != nil {
		logs.Error("Error finding ban: %v", err)
		return nil
	}
	_, err = database.DB_Main.Collection("bans").DeleteOne(context.TODO(), bson.M{"ip": ip})
	if err != nil {
		logs.Error("Error deleting ban: %v", err)
		return nil
	}
	sess, err := session.Get("session", c)
	if err != nil {
		return nil
	}
	user, ok := sess.Values["user"].(models.User)
	if !ok {
		logs.Error("Error getting user from session: %v", err)
		return nil
	}
	oldBan := models.Bans{
		ID:        ban.ID,
		Status:    "deleted",
		IP:        ban.IP,
		Reason:    ban.Reason,
		Username:  user.Username,
		UserID:    user.ID,
		Timestamp: ban.Timestamp,
		Expires:   ban.Expires,
	}
	database.DB_Main.Collection("old_bans").InsertOne(context.Background(), oldBan)
	logs.Info("Unbanned IP %s", ban.IP)
	return nil
}

func GetTotalBans(c echo.Context) error {
	totalBans := GetTotalBanCount()
	return c.JSON(http.StatusOK, totalBans)
}

func GetBans(c echo.Context) error {
	db := database.DB_Main.Collection("bans")
	var bans []models.Bans
	cur, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error getting bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban models.Bans
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
	var bans []models.Bans
	cur, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error getting old bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban models.Bans
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
	var bans []models.Bans
	cur, err := db.Find(context.Background(), bson.M{"status": "active"})
	if err != nil {
		logs.Error("Error getting active bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban models.Bans
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
	var bans []models.Bans
	cur, err := db.Find(context.Background(), bson.M{"status": "expired"})
	if err != nil {
		logs.Error("Error getting expired bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban models.Bans
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
	var bans []models.Bans
	cur, err := db.Find(context.Background(), bson.M{"status": "deleted"})
	if err != nil {
		logs.Error("Error getting deleted bans: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban models.Bans
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
	var ban models.Bans
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
		clientIP := c.RealIP()
		logs.Info("Checking ban for IP: %s", clientIP)
		banBool, err := cache.ClientBans.Do(ctx, cache.ClientBans.B().Get().Key(clientIP).Build()).AsBool()
		if err != valkey.Nil {
			logs.Error("Error checking ban in Redis: %v", err)
			return c.String(http.StatusInternalServerError, "Internal server error")
		}
		logs.Info("Ban check result for IP %s: %v", clientIP, banBool)
		if banBool == true {
			logs.Info("IP %s is banned", clientIP)

			// Fetch ban data from Redis
			banDataBytes, err := cache.ClientBans.Do(ctx, cache.ClientBans.B().Get().Key(clientIP).Build()).AsBytes()
			if err != nil || len(banDataBytes) == 0 {
				logs.Error("Error fetching ban data from Redis or data is empty: %v", err)
				return c.String(http.StatusInternalServerError, "Internal server error")
			}

			var banData models.Bans
			err = json.Unmarshal(banDataBytes, &banData)
			if err != nil {
				logs.Error("Error unmarshalling ban data: %v", err)
				return c.String(http.StatusInternalServerError, "Internal server error")
			}

			logs.Info("Rendering ban page for IP: %s", clientIP)
			tmpl, err := template.ParseFiles("views/util/banned.html")
			if err != nil {
				logs.Error("Error parsing template: %v", err)
				return c.String(http.StatusInternalServerError, "Internal server error")
			}

			data := map[string]interface{}{
				"Pagename":   "Banned",
				"BanReason":  banData.Reason,
				"BanExpires": banData.Expires,
			}
			err = tmpl.Execute(c.Response().Writer, data)
			if err != nil {
				logs.Error("Error executing template: %v", err)
				return c.String(http.StatusInternalServerError, "Internal server error")
			}
		} else {
			logs.Info("IP %s is not banned", clientIP)
			return next(c)
		}
		return nil
	}
}
func ExpireCheck() {
	fmt.Println("Checking for expired bans...")
	db := database.DB_Main.Collection("bans")
	var bans []models.Bans
	cur, err := db.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error getting bans: %v", err)
		return
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		var ban models.Bans
		err := cur.Decode(&ban)
		if err != nil {
			logs.Error("Error decoding ban: %v", err)
			return
		}
		bans = append(bans, ban)
	}

	currentTime := time.Now()
	for _, ban := range bans {
		expiresTime, err := time.Parse(time.RFC3339, ban.Expires)
		if err != nil {
			logs.Error("Error parsing time:", err)
			continue
		}
		if expiresTime.Before(currentTime) && ban.Status == "active" {
			fmt.Println("Ban expired:", ban)
			db.DeleteOne(context.Background(), bson.M{"id": ban.ID})
			// Move to OldBans table
			oldBan := models.Bans{
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

func DeleteBan(c echo.Context) models.Bans {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	id := c.Param("id")
	db := database.DB_Main.Collection("bans")
	var ban models.Bans
	db.FindOne(context.Background(), bson.M{"id": id}).Decode(&ban)
	// Move to OldBans table
	oldBan := models.Bans{
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
