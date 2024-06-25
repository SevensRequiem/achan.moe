package board

import (
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Board struct {
	BoardID     string `json:"id" gorm:"column:board_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PostCount   int64  `json:"post_count"`
	ImageOnly   bool   `json:"image_only"`   //todo
	Locked      bool   `json:"locked"`       //todo
	Archived    bool   `json:"archived"`     //todo
	LatestPosts bool   `json:"latest_posts"` //todo
}

type Post struct {
	BoardID   string `json:"BoardID"`
	ThreadID  string `json:"ThreadID"`
	PostID    int64  `json:"PostID"`
	Content   string `json:"Content"`
	ImageURL  string `json:"ImageURL"`
	Subject   string `json:"Subject"`
	Author    string `json:"Author"`
	ParentID  string `json:"ParentID"`
	Timestamp string `json:"Timestamp"`
	IP        string `json:"IP"`
	Sticky    bool   `json:"Sticky"`
	Locked    bool   `json:"Locked"`
}

type RecentPosts struct {
	ID        int64  `json:"ID"`
	BoardID   string `json:"BoardID"`
	ThreadID  string `json:"ThreadID"`
	PostID    int64  `json:"PostID"`
	Content   string `json:"Content"`
	ImageURL  string `json:"ImageURL"`
	Subject   string `json:"Subject"`
	Author    string `json:"Author"`
	ParentID  string `json:"ParentID"`
	Timestamp string `json:"Timestamp"`
}

type RateLimit struct {
	IP    string `json:"IP"`
	Count int    `json:"Count"`
}

type PostCounter struct {
	ID        int   `json:"ID"`
	PostCount int64 `json:"PostCount"`
}

func init() {
	db := database.Connect()
	defer database.Close()
	db.AutoMigrate(&Board{})
	db.AutoMigrate(&RecentPosts{})
	db.AutoMigrate(&PostCounter{})
}

func CreateThread(c echo.Context) error {

	if CheckIfLocked(c.Param("b")) {
		return c.JSON(http.StatusForbidden, "Board is locked")
	}
	if CheckIfArchived(c.Param("b")) {
		return c.JSON(http.StatusForbidden, "Board is archived")
	}
	err := c.Request().ParseMultipartForm(10 << 20)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error1")
	}
	// get board id
	boardID := c.Param("b")
	if boardID == "" {
		return c.JSON(http.StatusBadRequest, "Board ID cannot be empty")
	}
	// get thread id
	boardDir := "boards/" + boardID
	// scan all files in the board directory each json file is a thread, titled with an integer
	files, err := ioutil.ReadDir(boardDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error2")
	}

	// get the thread id by counting the number of json files in the board directory
	threadID := 1
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			threadID++
		}
	}
	// if 30 threads already exist, return an error
	if threadID > 30 {
		DeleteLastThread(boardID)
	}
	// get post content
	content := c.FormValue("content")
	if content == "" {
		return c.JSON(http.StatusBadRequest, "Content cannot be empty")
	}
	// get post subject
	subject := c.FormValue("subject")
	if subject == "" {
		return c.JSON(http.StatusBadRequest, "Subject cannot be empty")
	}
	// get post author
	author := c.FormValue("author")
	if author == "" {
		return c.JSON(http.StatusBadRequest, "Author cannot be empty")
	}

	// get image
	image, err := c.FormFile("image")
	if err != nil {
		if err == http.ErrMissingFile {
			image = nil // This is somewhat redundant as image would be nil if err != nil
		} else {
			return c.JSON(http.StatusBadRequest, "There was an error retrieving the file")
		}
	}

	imageURL, err := saveImage(boardID, image)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	// save the image to the board directory
	// /img/:b/:f
	sticky := false
	if !auth.AdminCheck(c) {
		stickyValue := c.FormValue("sticky")
		if stickyValue == "true" {
			sticky = true
		}
	}
	if !auth.JannyCheck(c, boardID) {
		stickyValue := c.FormValue("sticky")
		if stickyValue == "true" {
			sticky = true
		}
	}
	locked := false
	if !auth.AdminCheck(c) {
		lockedValue := c.FormValue("locked")
		if lockedValue == "true" {
			locked = true
		}
	}
	if !auth.JannyCheck(c, boardID) {
		lockedValue := c.FormValue("locked")
		if lockedValue == "true" {
			locked = true
		}
	}
	AddGlobalPostCount()
	AddBoardPostCount(boardID)
	var postCount int64
	postCount = GetGlobalPostCount()
	post := Post{
		BoardID:   boardID,
		ThreadID:  strconv.Itoa(threadID),
		PostID:    int64(postCount),
		Content:   content,
		ImageURL:  imageURL,
		Subject:   subject,
		Author:    author,
		Timestamp: time.Now().Format("01-02-2006 15:04:05"),
		IP:        c.RealIP(),
		Sticky:    sticky,
		Locked:    locked,
	}
	// create a json file for the thread
	jsonFilePath := boardDir + "/" + strconv.Itoa(threadID) + ".json"
	file, err := os.OpenFile(jsonFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error6")
	}
	defer file.Close()

	// Since it's a new thread, we start with an empty slice of Post and append the new post
	var posts []Post
	posts = append(posts, post)

	// Truncate the file to remove old content before writing new content
	if err := file.Truncate(0); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error truncating file7")
	}
	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error seeking file")
	}
	if err := json.NewEncoder(file).Encode(posts); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error encoding JSON")
	}

	boardName := url.PathEscape(c.Param("b"))
	threadIDStr := strconv.Itoa(threadID)
	redirectURL := "/board/" + boardName + "/" + threadIDStr
	AddRecentPost(post)
	return c.Redirect(http.StatusFound, redirectURL)
}

func CreateThreadPost(c echo.Context) error {
	boardID := c.Param("b")
	if boardID == "" {
		return c.JSON(http.StatusBadRequest, "Board ID cannot be empty")
	}

	if CheckIfLocked(boardID) {
		return c.JSON(http.StatusForbidden, "Board is locked")
	}

	if CheckIfArchived(boardID) {
		return c.JSON(http.StatusForbidden, "Board is archived")
	}

	threadID, err := strconv.Atoi(c.Param("t"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid thread ID")
	}

	content := c.FormValue("content")
	if content == "" {
		return c.JSON(http.StatusBadRequest, "Content cannot be empty")
	}

	author := c.FormValue("author")
	if author == "" {
		return c.JSON(http.StatusBadRequest, "Author cannot be empty")
	}

	var image *multipart.FileHeader

	image, err = c.FormFile("image")
	if err != nil {
		if err == http.ErrMissingFile {
			image = nil // This is somewhat redundant as image would be nil if err != nil
		} else {
			return c.JSON(http.StatusBadRequest, "There was an error retrieving the file")
		}
	}

	if threadIsFull(boardID, threadID) {
		return c.JSON(http.StatusForbidden, "Thread is full")
	}

	imageURL, err := saveImage(boardID, image)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	postCount := AddGlobalPostCount()
	AddBoardPostCount(boardID)

	post := Post{
		BoardID:   boardID,
		ThreadID:  strconv.Itoa(threadID),
		PostID:    postCount,
		Content:   content,
		ImageURL:  imageURL,
		Author:    author,
		Timestamp: time.Now().Format("01-02-2006 15:04:05"),
		IP:        c.RealIP(),
	}

	if err := addPostToFile(boardID, threadID, post); err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to add post to thread")
	}

	boardName := url.PathEscape(boardID)
	threadIDStr := strconv.Itoa(threadID)
	redirectURL := "/board/" + boardName + "/" + threadIDStr
	AddRecentPost(post)

	return c.Redirect(http.StatusFound, redirectURL)
}

func addPostToFile(boardID string, threadID int, post Post) error {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".json"
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.New("Error opening file")
	}
	defer file.Close()

	var posts []Post
	if err := json.NewDecoder(file).Decode(&posts); err != nil {
		return errors.New("Error decoding JSON")
	}

	posts = append(posts, post)

	if err := file.Truncate(0); err != nil {
		return errors.New("Error truncating file")
	}
	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		return errors.New("Error seeking file")
	}
	if err := json.NewEncoder(file).Encode(posts); err != nil {
		return errors.New("Error encoding JSON")
	}

	return nil
}

func saveImage(boardID string, image *multipart.FileHeader) (string, error) {
	if image == nil {
		return "", nil
	}
	imagename := uuid.New().String()
	imageExt := filepath.Ext(image.Filename)
	if imageExt == "" {
		return "", errors.New("Invalid image extension")
	}

	imageFile, err := image.Open()
	if err != nil {
		return "", errors.New("Error opening image file")
	}
	defer imageFile.Close()

	imageData, err := ioutil.ReadAll(imageFile)
	if err != nil {
		return "", errors.New("Error reading image file")
	}

	imageURL := imagename + imageExt
	baseImgDir := "boards/" + boardID + "/" + imagename + imageExt
	if err := ioutil.WriteFile(baseImgDir, imageData, 0644); err != nil {
		return "", errors.New("Error writing image file")
	}

	return imageURL, nil
}

func threadIsFull(boardID string, threadID int) bool {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".json"
	file, err := os.Open(filepath)
	if err != nil {
		return true
	}
	defer file.Close()
	var posts []Post
	json.NewDecoder(file).Decode(&posts)
	return len(posts) >= 300
}

func DeleteLastThread(boardID string) {
	dir := "boards/" + boardID
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	var oldestThread string
	oldestTime := time.Now()
	for _, file := range files {
		if !file.IsDir() {
			filepath := dir + "/" + file.Name()
			f, err := os.Open(filepath)
			if err != nil {
				continue
			}
			var posts []Post
			if err := json.NewDecoder(f).Decode(&posts); err != nil || len(posts) == 0 {
				f.Close()
				continue
			}
			f.Close()
			timestamp, err := time.Parse("01-02-2006 15:04:05", posts[0].Timestamp)
			if err != nil {
				continue
			}
			if timestamp.Before(oldestTime) {
				oldestTime = timestamp
				oldestThread = file.Name()
			}
		}
	}
	if oldestThread != "" {
		os.Remove(dir + "/" + oldestThread)
	}
}

func GetBoards() []Board {
	db := database.Connect()
	defer database.Close()
	var boards []Board
	db.Find(&boards)

	return boards
}

func GetLatestPosts(n int) ([]RecentPosts, error) {
	db := database.Connect()
	defer database.Close()
	var posts []RecentPosts
	db.Order("timestamp DESC").Limit(n).Find(&posts)

	return posts, nil
}

func GetBoardName(boardID string) string {
	db := database.Connect()
	defer database.Close()
	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.Name
}

func GetBoard(boardID string) Board {
	db := database.Connect()
	defer database.Close()
	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board
}

func GetBoardID(boardID string) string {
	db := database.Connect()
	defer database.Close()
	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.BoardID
}

func GetThreads(boardID string) []Post {
	dir := "boards/" + boardID
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil
	}
	var threads []Post
	for _, file := range files {
		if !file.IsDir() {
			filepath := dir + "/" + file.Name()
			f, err := os.Open(filepath)
			if err != nil {
				return nil
			}
			var posts []Post
			if err := json.NewDecoder(f).Decode(&posts); err != nil || len(posts) == 0 {
				f.Close() // Close the file before returning the error
				continue  // Skip to the next file if an error occurs or the file is empty
			}
			f.Close()                           // Ensure file is closed after processing
			threads = append(threads, posts[0]) // Append only the first post
		}
	}
	return threads
}

func GetPosts(boardID string, threadID int) []Post {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".json"
	file, err := os.Open(filepath)
	if err != nil {
		return nil
	}
	defer file.Close()
	var posts []Post
	json.NewDecoder(file).Decode(&posts)

	// Sanitize the Content field of each Post
	for i := range posts {
		posts[i].Content = template.HTMLEscapeString(posts[i].Content)
	}

	// Assuming you still want to ignore the first post if there are more than one
	if len(posts) > 1 {
		posts = posts[1:]
	} else {
		return []Post{}
	}

	return posts
}

func GetThread(boardID string, threadID int) Post {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".json"
	file, err := os.Open(filepath)
	if err != nil {
		return Post{}
	}
	defer file.Close()
	var posts []Post
	err = json.NewDecoder(file).Decode(&posts)
	if err != nil || len(posts) == 0 {
		return Post{} // Return an empty Post if there's an error or the array is empty
	}
	return posts[0] // Return the first post
}

func AddRecentPost(post Post) {
	db := database.Connect()
	defer database.Close()
	var count int64
	db.Model(&RecentPosts{}).Count(&count)
	if count >= 10 {
		var oldestPost RecentPosts
		db.Order("timestamp ASC").First(&oldestPost)
		// Assuming RecentPosts has an ID field that serves as the primary key
		db.Where("id = ?", oldestPost.ID).Delete(&RecentPosts{})
	}
	recentPost := RecentPosts{
		BoardID:   post.BoardID,
		ThreadID:  post.ThreadID,
		PostID:    post.PostID,
		Content:   post.Content,
		ImageURL:  post.ImageURL,
		Subject:   post.Subject,
		Timestamp: post.Timestamp,
	}
	db.Create(&recentPost)
}

func CheckIfLocked(boardID string) bool {
	db := database.Connect()
	defer database.Close()
	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	defer database.Close() // Ensure the database is closed after all operations are done
	return board.Locked
}

func CheckIfArchived(boardID string) bool {
	db := database.Connect()
	defer database.Close()
	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	defer database.Close() // Ensure the database is closed after all operations are done
	return board.Archived
}

func CheckIfImageOnly(boardID string) bool {
	db := database.Connect()
	defer database.Close()
	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	defer database.Close() // Ensure the database is closed after all operations are done
	return board.ImageOnly
}

func ThreadCheckLocked(c echo.Context, boardid string, threadid string) bool {
	filepath := "boards/" + boardid + "/" + threadid + ".json"
	file, err := os.Open(filepath)
	if err != nil {
		return false
	}
	defer file.Close()
	var posts []Post
	json.NewDecoder(file).Decode(&posts)
	return posts[0].Locked
}

func AddGlobalPostCount() int64 {
	db := database.Connect()
	defer database.Close()

	var postCounter PostCounter
	db.First(&postCounter)
	postCounter.PostCount++
	db.Save(&postCounter)
	return postCounter.PostCount
}
func GetGlobalPostCount() int64 {
	db := database.Connect()
	defer database.Close()

	var postCounter PostCounter
	db.First(&postCounter)
	return postCounter.PostCount
}

func AddBoardPostCount(boardID string) {
	db := database.Connect()
	defer database.Close()

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	board.PostCount++
	db.Where("board_id = ?", boardID).Save(&board)
}

func GetBoardPostCount(boardID string) int64 {
	db := database.Connect()
	defer database.Close()

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.PostCount
}
