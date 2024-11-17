package news

import (
	"context"
	"net/http"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/models"
	"github.com/labstack/echo/v4"
)

// News represents a news article with a title, content, and date.

// init function creates the file if it doesn't exist and adds dummy news if needed.
func NewNews(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusForbidden, "You are not authorized to add news")
	}
	title := c.FormValue("title")
	content := c.FormValue("content")
	date := time.Now().Format("2006-01-02")

	news := models.News{
		Title:   title,
		Content: content,
		Date:    date,
		Author:  auth.LoggedInUser(c).Username,
	}
	if err := c.Bind(&news); err != nil {
		return c.String(http.StatusBadRequest, "Invalid input")
	}
	AddNews(news)
	return c.String(http.StatusOK, "News added successfully")
}

// AddNews adds a news article to the file.
func AddNews(news models.News) {
	db := database.DB_Main
	db.Collection("news").InsertOne(context.Background(), news)
}

// GetNews retrieves the last 10 news articles from the file.
func GetNews() []models.News {
	allNews := GetAllNews()
	var recentNews []models.News

	if len(allNews) > 10 {
		allNews = allNews[len(allNews)-10:]
	}

	for i := len(allNews) - 1; i >= 0; i-- {
		recentNews = append(recentNews, allNews[i])
	}

	return recentNews
}

// ClearNews clears all news articles from the file.
func ClearNews() {
	db := database.DB_Main
	db.Collection("news").Drop(context.Background())
}

// GetAllNews retrieves all news articles from the file.
func GetAllNews() []models.News {
	db := database.DB_Main
	cursor, err := db.Collection("news").Find(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer cursor.Close(context.Background())

	var allNews []models.News
	for cursor.Next(context.Background()) {
		var news models.News
		if err := cursor.Decode(&news); err != nil {
			continue
		}
		allNews = append(allNews, news)
	}

	return allNews
}

func DummyData() {
	ClearNews()
	AddNews(models.News{
		Title:   "Welcome to the News Page",
		Content: "This is the first news article. It will be displayed on the news page.",
		Date:    "2021-01-01",
	})
	AddNews(models.News{
		Title:   "New Feature Added",
		Content: "We have added a new feature to the website. Check it out now!",
		Date:    "2021-01-02",
	})
	AddNews(models.News{
		Title:   "Important Announcement",
		Content: "There will be a scheduled maintenance on the website. Please bear with us.",
		Date:    "2021-01-03",
	})
	AddNews(models.News{
		Title:   "Holiday Closure",
		Content: "The website will be closed for the holidays. We will be back soon!",
		Date:    "2021-01-04",
	})
	AddNews(models.News{
		Title:   "Thank You for Your Support",
		Content: "We appreciate all the support from our users. Thank you!",
		Date:    "2021-01-05",
	})
}

func Migratenewsfromsql() {
	if database.MySQL == nil {
		return
	}

	var news []models.News
	result := database.MySQL.Table("news").Find(&news)
	if result.Error != nil {
		return
	}

	for _, n := range news {
		AddNews(n)
	}
}
