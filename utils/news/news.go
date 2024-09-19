package news

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"achan.moe/auth"
	"github.com/labstack/echo/v4"
)

// News represents a news article with a title, content, and date.
type News struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Date    string `json:"date"`
}

// init function creates the file if it doesn't exist and adds dummy news if needed.
func init() {
	if _, err := os.Stat("news.json"); os.IsNotExist(err) {
		file, err := os.Create("news.json")
		if err != nil {
			fmt.Printf("Failed to create file: %v\n", err)
			return
		}
		defer file.Close()

		DummyNews()
	}
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
	}
	if err := c.Bind(&news); err != nil {
		return c.String(http.StatusBadRequest, "Invalid input")
	}
	AddNews(news)
	return c.String(http.StatusOK, "News added successfully")
}

// AddNews adds a news article to the file.
func AddNews(news News) {
	news.Date = time.Now().Format("2006-01-02")
	allNews := GetAllNews()
	allNews = append(allNews, news)

	data, err := json.Marshal(allNews)
	if err != nil {
		fmt.Printf("Failed to encode news: %v\n", err)
		return
	}

	err = ioutil.WriteFile("news.json", data, 0666)
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
	}
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
	err := os.Remove("news.json")
	if err != nil {
		fmt.Printf("Failed to clear file: %v\n", err)
	}
	// Create a new empty file
	_, err = os.Create("news.json")
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
	}
}

// GetAllNews retrieves all news articles from the file.
func GetAllNews() []News {
	file, err := os.ReadFile("news.json")
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return nil
	}

	var news []News
	err = json.Unmarshal(file, &news)
	if err != nil {
		fmt.Printf("Failed to decode news: %v\n", err)
		return nil
	}

	return news
}

// DummyNews adds some initial dummy news articles.
func DummyNews() {
	if len(GetAllNews()) == 0 { // Only add if no news exists
		AddNews(News{Title: "Dummy News 1", Content: "This is a dummy news article."})
		AddNews(News{Title: "Dummy News 2", Content: "This is another dummy news article."})
		AddNews(News{Title: "Dummy News 3", Content: "This is yet another dummy news article."})
		AddNews(News{Title: "Dummy News 4", Content: "This is a fourth dummy news article."})
		AddNews(News{Title: "Dummy News 5", Content: "This is a fifth dummy news article."})
		AddNews(News{Title: "Dummy News 6", Content: "This is a sixth dummy news article."})
		AddNews(News{Title: "Dummy News 7", Content: "This is a seventh dummy news article."})
		AddNews(News{Title: "Dummy News 8", Content: "This is an eighth dummy news article."})
		AddNews(News{Title: "Dummy News 9", Content: "This is a ninth dummy news article."})
		AddNews(News{Title: "Dummy News 10", Content: "This is a tenth dummy news article."})
		AddNews(News{Title: "Dummy News 11", Content: "This is an eleventh dummy news article."})
	}
}

// TestNews tests the news functions.
func TestNews() {
	ClearNews() // Start fresh
	DummyNews() // Add dummy articles

	allNews := GetAllNews()
	fmt.Printf("All News Articles: %+v\n", allNews)

	lastNews := GetNews()
	fmt.Printf("Last 10 News Articles: %+v\n", lastNews)
}
