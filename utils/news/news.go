package news

import (
	"net/http"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/labstack/echo/v4"
)

// News represents a news article with a title, content, and date.
type News struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Date    string `json:"date"`
	Author  string `json:"author"`
}

// init function creates the file if it doesn't exist and adds dummy news if needed.
func init() {
	db := database.DB
	db.AutoMigrate(&News{})
}
func NewNews(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusForbidden, "You are not authorized to add news")
	}
	title := c.FormValue("title")
	content := c.FormValue("content")
	date := time.Now().Format("2006-01-02")

	news := News{
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
func AddNews(news News) {
	db := database.DB
	db.Create(&news)
}

// GetNews retrieves the last 10 news articles from the file.
func GetNews() []News {
	allNews := GetAllNews()
	var recentNews []News

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
	db := database.DB
	db.Exec("DELETE FROM news")
}

// GetAllNews retrieves all news articles from the file.
func GetAllNews() []News {
	db := database.DB
	var news []News
	db.Find(&news)
	return news
}

func DummyData() {
	ClearNews()
	AddNews(News{
		Title:   "Welcome to the News Page",
		Content: "This is the first news article. It will be displayed on the news page.",
		Date:    "2021-01-01",
	})
	AddNews(News{
		Title:   "New Feature Added",
		Content: "We have added a new feature to the website. Check it out now!",
		Date:    "2021-01-02",
	})
	AddNews(News{
		Title:   "Important Announcement",
		Content: "There will be a scheduled maintenance on the website. Please bear with us.",
		Date:    "2021-01-03",
	})
	AddNews(News{
		Title:   "Holiday Closure",
		Content: "The website will be closed for the holidays. We will be back soon!",
		Date:    "2021-01-04",
	})
	AddNews(News{
		Title:   "Thank You for Your Support",
		Content: "We appreciate all the support from our users. Thank you!",
		Date:    "2021-01-05",
	})
}
