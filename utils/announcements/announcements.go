package announcements

import (
	"context"
	"encoding/json"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/cache"
	"github.com/labstack/echo/v4"
)

var ctx = context.Background()

func AddAnnouncement(c echo.Context) error {
	announcement := models.Announcement{}
	if err := c.Bind(&announcement); err != nil {
		return err
	}

	announcement.Timestamp = time.Now().Unix()

	if c.FormValue("board_id") != "" {
		announcement.BoardID = c.FormValue("board_id")
	} else {
		announcement.BoardID = "global"
	}
	processAnnouncement(c, announcement)
	return c.JSON(200, "announcement added")
}

func processAnnouncement(c echo.Context, announcement models.Announcement) {
	// Marshal the announcement struct into JSON
	announcementJSON, err := json.Marshal(announcement)
	if err != nil {
		logs.Error("Failed to marshal announcement: ", err)
		return
	}

	// Use the marshaled JSON string as the value
	err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("announcement_"+announcement.BoardID).Value(string(announcementJSON)).Build()).Error()
	if err != nil {
		logs.Error("Failed to set announcement in Redis: ", err)
		return
	}

	announcementModel := models.Announcement{
		BoardID:   announcement.BoardID,
		Content:   announcement.Content,
		Timestamp: announcement.Timestamp,
		User:      announcement.User,
	}
	announcementModel.ID = announcement.ID
	user := auth.UserSession(c)
	announcementModel.User = user.Username
	announcementModel.Timestamp = time.Now().Unix()
	announcementModel.BoardID = c.FormValue("board_id")

	// Save the announcement to the database
	_, err = database.DB_Actions.Collection("announcements").InsertOne(context.Background(), announcementModel)
	if err != nil {
		logs.Error("Failed to save announcement to database: ", err)
		return
	}
	logs.Info("Announcement added: ", announcement)
}
func GetAnnouncement(c echo.Context) error {
	boardID := c.Param("board_id")
	announcement, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Get().Key("announcement_"+boardID).Build()).AsStrMap()
	if err != nil {
		logs.Error("Failed to get announcement from Redis: ", err)
		return c.JSON(500, "Failed to get announcement")
	}
	return c.JSON(200, announcement)
}
func DeleteAnnouncement(c echo.Context) error {
	boardID := c.FormValue("board_id")
	announcementID := c.FormValue("announcement_id")
	_, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Del().Key("announcement_"+boardID).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to delete announcement from Redis: ", err)
		return c.JSON(500, "Failed to delete announcement")
	}
	// Delete the announcement from the database
	_, err = database.DB_Actions.Collection("announcements").DeleteOne(context.Background(), models.Announcement{BoardID: boardID, ID: announcementID})
	if err != nil {
		logs.Error("Failed to delete announcement from database: ", err)
		return c.JSON(500, "Failed to delete announcement from database")
	}
	return c.JSON(200, "announcement deleted")
}
