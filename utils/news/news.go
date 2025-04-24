package news

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/models"
	"achan.moe/utils/cache"
	"github.com/labstack/echo/v4"
)

func NewNews(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusForbidden, "You are not authorized to add news")
	}
	title := c.FormValue("title")
	content := c.FormValue("content")
	date := time.Now().Unix()
	user := auth.LoggedInUser(c)
	author := "Admin"
	if user.DisplayName == "" {
		author = user.Username
	} else {
		author = user.DisplayName
	}
	news := models.News{
		ID:      strconv.Itoa(int(date)),
		Title:   title,
		Content: content,
		Date:    date,
		Author:  author,
	}
	if err := c.Bind(&news); err != nil {
		return c.String(http.StatusBadRequest, "Invalid input")
	}
	AddNews(c, news)
	cache.CacheNews(news)
	return c.JSON(http.StatusOK, "News added successfully")
}

func AddNews(c echo.Context, news models.News) error {
	_, err := database.DB_Main.Collection("news").InsertOne(context.Background(), news)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to add news: "+err.Error())
	}
	return nil
}

func ClearNews() {
	database.DB_Main.Collection("news").Drop(context.Background())
	cache.ClearNews()
}

func DeleteNews(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusForbidden, "You are not authorized to delete news")
	}
	newsID := c.FormValue("news_id")
	if newsID == "" {
		return c.String(http.StatusBadRequest, "Invalid news ID")
	}

	_, err := database.DB_Main.Collection("news").DeleteOne(context.Background(), models.News{ID: newsID})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete news: "+err.Error())
	}
	cache.DeleteNews(newsID)
	return c.JSON(http.StatusOK, "News deleted successfully")
}
