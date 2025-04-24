package actions

import (
	"context"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var db = database.DB_Actions

func Routes(e *echo.Echo) {
	e.GET("/actions/announcements", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.Redirect(302, "/")
		}
		return GetAnnouncementActionsHandler(c)
	})
	e.GET("/actions/bans", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.Redirect(302, "/")
		}
		return GetBanActionsHandler(c)
	})
	e.GET("/actions/unbans", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.Redirect(302, "/")
		}
		return GetUnbanActionsHandler(c)
	})
	e.GET("/actions/boards", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.Redirect(302, "/")
		}
		return GetBoardActionsHandler(c)
	})
	e.GET("/actions/threads", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.Redirect(302, "/")
		}
		return GetThreadActionsHandler(c)
	})
	e.GET("/actions/posts", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.Redirect(302, "/")
		}
		return GetPostActionsHandler(c)
	})
}
func AddAnnouncementAction(c echo.Context, announcement models.Announcement) error {
	announcement = models.Announcement{
		BoardID:   announcement.BoardID,
		Content:   announcement.Content,
		Timestamp: announcement.Timestamp,
		UserID:    announcement.UserID,
	}

	user := auth.UserSession(c)
	action := models.AnnouncementActions{
		Action:    "add",
		BoardID:   announcement.BoardID,
		Content:   announcement.Content,
		Timestamp: announcement.Timestamp,
		UserID:    user.ID,
		IP:        c.RealIP(),
	}

	_, err := db.Collection("announcements").InsertOne(context.Background(), &models.AnnouncementActions{
		Action:    action.Action,
		BoardID:   action.BoardID,
		Content:   action.Content,
		Timestamp: action.Timestamp,
		UserID:    action.UserID,
		IP:        action.IP,
	})
	if err != nil {
		logs.Error("Failed to add announcement action", err)
		return err
	}
	logs.Info("Added announcement action", action)
	return nil
}

func GetAnnouncementActions(c echo.Context) ([]models.AnnouncementActions, error) {
	actions := []models.AnnouncementActions{}
	cursor, err := db.Collection("announcements").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Failed to get announcement actions", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var action models.AnnouncementActions
		if err := cursor.Decode(&action); err != nil {
			logs.Error("Failed to decode announcement action", err)
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}

func GetAnnouncementActionsHandler(c echo.Context) error {
	actions, err := GetAnnouncementActions(c)
	if err != nil {
		logs.Error("Failed to get announcement actions", err)
		return c.JSON(500, "Failed to get announcement actions")
	}
	return c.JSON(200, actions)
}

func AddBanAction(c echo.Context, ban models.Bans) error {
	ban = models.Bans{
		ID:        ban.ID,
		Status:    ban.Status,
		IP:        ban.IP,
		Reason:    ban.Reason,
		Timestamp: ban.Timestamp,
		Expires:   ban.Expires,
	}

	action := models.BanActions{}
	user := auth.UserSession(c)
	_, err := db.Collection("bans").InsertOne(context.Background(), &models.BanActions{
		ID:        primitive.NewObjectID(),
		Status:    ban.Status,
		IP:        ban.IP,
		Reason:    ban.Reason,
		Timestamp: ban.Timestamp,
		Expires:   ban.Expires,
		Username:  user.Username,
		UserID:    user.ID,
	})
	if err != nil {
		logs.Error("Failed to add ban action", err)
		return err
	}
	logs.Info("Added ban action", action)
	return nil
}

func GetBanActions(c echo.Context) ([]models.BanActions, error) {
	actions := []models.BanActions{}
	cursor, err := db.Collection("bans").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Failed to get ban actions", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var action models.BanActions
		if err := cursor.Decode(&action); err != nil {
			logs.Error("Failed to decode ban action", err)
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}
func AddUnbanAction(c echo.Context, ban models.Bans) error {
	ban = models.Bans{
		ID:        ban.ID,
		Status:    ban.Status,
		IP:        ban.IP,
		Reason:    ban.Reason,
		Timestamp: ban.Timestamp,
		Expires:   ban.Expires,
	}

	action := models.UnbanActions{}
	user := auth.UserSession(c)
	_, err := db.Collection("unbans").InsertOne(context.Background(), &models.UnbanActions{
		ID:        primitive.NewObjectID(),
		Status:    ban.Status,
		IP:        ban.IP,
		Username:  user.Username,
		UserID:    user.ID,
		Reason:    ban.Reason,
		Timestamp: ban.Timestamp,
	})
	if err != nil {
		logs.Error("Failed to add unban action", err)
		return err
	}
	logs.Info("Added unban action", action)
	return nil
}

func GetUnbanActions(c echo.Context) ([]models.UnbanActions, error) {
	actions := []models.UnbanActions{}
	cursor, err := db.Collection("unbans").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Failed to get unban actions", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var action models.UnbanActions
		if err := cursor.Decode(&action); err != nil {
			logs.Error("Failed to decode unban action", err)
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}

func GetBanActionsHandler(c echo.Context) error {
	actions, err := GetBanActions(c)
	if err != nil {
		logs.Error("Failed to get ban actions", err)
		return c.JSON(500, "Failed to get ban actions")
	}
	return c.JSON(200, actions)
}
func GetUnbanActionsHandler(c echo.Context) error {
	actions, err := GetUnbanActions(c)
	if err != nil {
		logs.Error("Failed to get unban actions", err)
		return c.JSON(500, "Failed to get unban actions")
	}
	return c.JSON(200, actions)
}

func AddBoardAction(c echo.Context, board models.Board) error {
	action := models.BoardActions{}
	user := auth.UserSession(c)
	_, err := db.Collection("boards").InsertOne(context.Background(), &models.BoardActions{
		ID:        primitive.NewObjectID(),
		BoardID:   board.BoardID,
		BoardName: board.Name,
		Timestamp: time.Now().Unix(),
		UserID:    user.ID,
		Action:    "add",
		IP:        c.RealIP(),
	})
	if err != nil {
		logs.Error("Failed to add board action", err)
		return err
	}
	logs.Info("Added board action", action)
	return nil
}
func DeleteBoardAction(c echo.Context, board models.Board) error {
	action := models.BoardActions{}
	user := auth.UserSession(c)
	_, err := db.Collection("boards").InsertOne(context.Background(), &models.BoardActions{
		ID:        primitive.NewObjectID(),
		BoardID:   board.BoardID,
		BoardName: board.Name,
		Timestamp: time.Now().Unix(),
		UserID:    user.ID,
		Action:    "del",
		IP:        c.RealIP(),
	})
	if err != nil {
		logs.Error("Failed to delete board action", err)
		return err
	}
	logs.Info("Deleted board action", action)
	return nil
}
func GetBoardActions(c echo.Context) ([]models.BoardActions, error) {
	actions := []models.BoardActions{}
	cursor, err := db.Collection("boards").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Failed to get board actions", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var action models.BoardActions
		if err := cursor.Decode(&action); err != nil {
			logs.Error("Failed to decode board action", err)
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}
func GetBoardActionsHandler(c echo.Context) error {
	actions, err := GetBoardActions(c)
	if err != nil {
		logs.Error("Failed to get board actions", err)
		return c.JSON(500, "Failed to get board actions")
	}
	return c.JSON(200, actions)
}

func AddThreadAction(c echo.Context, thread models.ThreadPost) error {
	action := models.ThreadActions{}
	user := auth.UserSession(c)
	if user == nil {
		user = &models.User{
			ID:       0,
			Username: "Anonymous",
		}
	}
	_, err := db.Collection("threads").InsertOne(context.Background(), &models.ThreadActions{
		ID:             primitive.NewObjectID(),
		ThreadID:       thread.ThreadID,
		Timestamp:      time.Now().Unix(),
		UserID:         user.ID,
		Action:         "add",
		IP:             c.RealIP(),
		Subject:        thread.Subject,
		BoardID:        thread.BoardID,
		Username:       user.Username,
		Image:          thread.Image,
		Thumbnail:      thread.Thumbnail,
		PartialContent: thread.PartialContent,
	})
	if err != nil {
		logs.Error("Failed to add thread action", err)
		return err
	}
	logs.Info("Added thread action", action)
	return nil
}

func GetThreadActions(c echo.Context) ([]models.ThreadActions, error) {
	actions := []models.ThreadActions{}
	cursor, err := db.Collection("threads").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Failed to get thread actions", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var action models.ThreadActions
		if err := cursor.Decode(&action); err != nil {
			logs.Error("Failed to decode thread action", err)
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}
func GetThreadActionsHandler(c echo.Context) error {
	actions, err := GetThreadActions(c)
	if err != nil {
		logs.Error("Failed to get thread actions", err)
		return c.JSON(500, "Failed to get thread actions")
	}
	return c.JSON(200, actions)
}

func AddPostAction(c echo.Context, post models.Posts) error {
	action := models.PostActions{}
	user := auth.UserSession(c)
	if user == nil {
		user = &models.User{
			ID:       0,
			Username: "Anonymous",
		}
	}
	_, err := db.Collection("posts").InsertOne(context.Background(), &models.PostActions{
		ID:             primitive.NewObjectID(),
		ThreadID:       post.ParentID,
		Timestamp:      time.Now().Unix(),
		UserID:         user.ID,
		Action:         "add",
		IP:             c.RealIP(),
		Subject:        post.Subject,
		BoardID:        post.BoardID,
		Username:       user.Username,
		Image:          post.Image,
		Thumbnail:      post.Thumbnail,
		PartialContent: post.PartialContent,
		PostID:         post.PostID,
	})
	if err != nil {
		logs.Error("Failed to add post action", err)
		return err
	}
	logs.Info("Added post action", action)
	return nil
}

func GetPostActions(c echo.Context) ([]models.PostActions, error) {
	actions := []models.PostActions{}
	cursor, err := db.Collection("posts").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Failed to get post actions", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var action models.PostActions
		if err := cursor.Decode(&action); err != nil {
			logs.Error("Failed to decode post action", err)
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}
func GetPostActionsHandler(c echo.Context) error {
	actions, err := GetPostActions(c)
	if err != nil {
		logs.Error("Failed to get post actions", err)
		return c.JSON(500, "Failed to get post actions")
	}
	return c.JSON(200, actions)
}
