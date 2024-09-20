package board

import (
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/images"
	"achan.moe/utils/queue"
	"achan.moe/utils/sitemap"
	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"
	"golang.org/x/time/rate"
)

type Board struct {
	BoardID     string `gob:"id" gorm:"column:board_id"`
	Name        string `gob:"name"`
	Description string `gob:"description"`
	PostCount   int64  `gob:"post_count"`
	ImageOnly   bool   `gob:"image_only"`
	Locked      bool   `gob:"locked"`
	Archived    bool   `gob:"archived"` //todo
	LatestPosts bool   `gob:"latest_posts"`
	Pages       int    `gob:"pages"` //todo
}

type Post struct {
	BoardID        string `gob:"BoardID"`
	ThreadID       string `gob:"ThreadID"`
	PostID         string `gob:"PostID"`
	Content        string `gob:"Content"`
	PartialContent string `gob:"PartialContent"`
	ImageURL       string `gob:"ImageURL"`
	ThumbURL       string `gob:"ThumbURL"`
	Subject        string `gob:"Subject"`
	Author         string `gob:"Author"`
	ParentID       string `gob:"ParentID"`
	Timestamp      string `gob:"Timestamp"`
	IP             string `gob:"IP"`
	Sticky         bool   `gob:"Sticky"`
	Locked         bool   `gob:"Locked"`
	Page           int    `gob:"Page"`
	Upvotes        int    `gob:"Upvotes"`
	Downvotes      int    `gob:"Downvotes"`
	ReportCount    int    `gob:"ReportCount"`
}

type RecentPosts struct {
	ID             int64  `gob:"ID"`
	BoardID        string `gob:"BoardID"`
	ThreadID       string `gob:"ThreadID"`
	PostID         string `gob:"PostID"`
	Content        string `gob:"Content"`
	PartialContent string `gob:"PartialContent"`
	ImageURL       string `gob:"ImageURL"`
	ThumbURL       string `gob:"ThumbURL"`
	Subject        string `gob:"Subject"`
	Author         string `gob:"Author"`
	ParentID       string `gob:"ParentID"`
	Timestamp      string `gob:"Timestamp"`
}

type Recents struct {
	ID     int64  `gob:"ID" gorm:"primaryKey"`
	PostID string `gob:"PostID"`
}

type PostCounter struct {
	ID        int   `gob:"ID" gorm:"primaryKey"`
	PostCount int64 `gob:"PostCount" gorm:"default:0"`
}

var manager = queue.NewQueueManager()
var q = manager.GetQueue("thread", 1000)

func init() {
	db := database.DB

	db.AutoMigrate(&Board{})
	db.AutoMigrate(&RecentPosts{})
	db.AutoMigrate(&Recents{})
	db.AutoMigrate(&PostCounter{})
	manager.ProcessQueuesWithPrefix("thread")
}

func extractThreadData(c echo.Context) (string, string, string, string, string, *multipart.FileHeader, error) {
	boardID := c.Param("b")
	content := c.FormValue("content")
	subject := c.FormValue("subject")
	author := c.FormValue("author")
	image, err := c.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		return "", "", "", "", "", nil, err
	}

	return boardID, content, subject, author, c.FormValue("isSticky"), image, nil
}

func processThread(c echo.Context, boardID, content, subject, author, stickyValue string, image *multipart.FileHeader) error {
	if boardID == "" {
		return errors.New("Board ID cannot be empty")
	}
	if CheckIfLocked(boardID) {
		return errors.New("Board is locked")
	}
	if CheckIfArchived(boardID) {
		return errors.New("Board is archived")
	}
	imgonly := CheckIfImageOnly(boardID)
	if imgonly && content != "" {
		return errors.New("This board only allows image posts")
	}
	files, err := ioutil.ReadDir("boards/" + boardID)
	if err != nil {
		return err
	}
	threadID := 1
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".gob") {
			threadID++
		}
	}
	if threadID > 30 {
		DeleteLastThread(boardID)
	}
	if content == "" && !imgonly {
		if image == nil {
			return errors.New("Content cannot be empty")
		} else {
			content = ""
		}
	}
	if subject == "" {
		subject = "No Subject"
	}
	if author == "" {
		return errors.New("Author cannot be empty")
	}
	if image == nil {
		return errors.New("Image is required for threads")
	}
	if image.Size > 11<<20 {
		return errors.New("File is too large")
	}
	ext := filepath.Ext(image.Filename)
	if !isValidImageExtension(ext) {
		return errors.New("Invalid image extension")
	}
	imageURL, err := saveImage(boardID, image)
	if err != nil {
		return err
	}
	go images.GenerateThumbnail("boards/"+boardID+"/"+imageURL, "thumbs/"+imageURL, 200, 200)
	AddGlobalPostCount()
	AddBoardPostCount(boardID)
	sticky := stickyValue == "on"
	locked := false
	if !auth.AdminCheck(c) {
		lockedValue := c.FormValue("isLocked")
		locked = lockedValue == "on"
	}
	// limit the subject length to 30 characters
	if len(subject) > 30 {
		subject = subject[:30]
	}
	post := Post{
		BoardID:        boardID,
		ThreadID:       strconv.Itoa(threadID),
		PostID:         GenUUID(),
		Content:        content,
		PartialContent: content[:min(len(content), 20)],
		ImageURL:       imageURL,
		ThumbURL:       "thumbs/" + imageURL,
		Subject:        subject,
		Author:         author,
		Timestamp:      time.Now().Format("01-02-2006 15:04:05"),
		IP:             "IP Placeholder", // Replace with actual IP retrieval
		Sticky:         sticky,
		Locked:         locked,
	}

	addToRecents(post.PostID)

	boardDir := "boards/" + boardID
	jsonFilePath := boardDir + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.OpenFile(jsonFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	var posts []Post
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		return err
	}
	if err := gob.NewEncoder(file).Encode(append(posts, post)); err != nil {
		return err
	}
	if LatestPostsCheck(c, boardID) {
		AddRecentPost(post)
	}
	sitemap := sitemap.Sitemap{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	sitemap.AddURL("https://achan.moe/board/"+url.PathEscape(boardID)+"/"+strconv.Itoa(threadID), "daily", "0.5")
	return nil
}

func CreateThread(c echo.Context) error {
	if auth.PremiumCheck(c) {
		limiter := rate.NewLimiter(rate.Every(5*time.Minute), 1)
		if !limiter.Allow() {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "5 Minute cooldown"})
		}
	}

	boardID, content, subject, author, stickyValue, image, err := extractThreadData(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	q.Enqueue(func() {
		if err := processThread(c, boardID, content, subject, author, stickyValue, image); err != nil {
			fmt.Println("Error processing thread:", err)
		}
	})

	return nil
}
func addToRecents(postID string) {
	db := database.DB
	db.Create(&Recents{PostID: postID})
}
func extractPostData(c echo.Context) (string, string, string, string, *multipart.FileHeader, error) {
	boardID := c.Param("b")
	content := c.FormValue("content")
	author := c.FormValue("author")
	image, err := c.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		return "", "", "", "", nil, err
	}

	if image == nil {
		image = nil
	}

	return boardID, content, c.FormValue("replyto"), author, image, nil
}

func processPost(c echo.Context, boardID, content, replyto, author string, image *multipart.FileHeader, postid string) error {
	if boardID == "" {
		return errors.New("Board ID cannot be empty")
	}
	if CheckIfThreadLocked(c, boardID, c.Param("t")) {
		return errors.New("Thread is locked")
	}
	if CheckIfLocked(boardID) {
		return errors.New("Board is locked")
	}
	if CheckIfArchived(boardID) {
		return errors.New("Board is archived")
	}
	imgonly := CheckIfImageOnly(boardID)
	if imgonly && content != "" {
		return errors.New("This board only allows image posts")
	}
	threadID, err := strconv.Atoi(c.Param("t"))
	if err != nil {
		return errors.New("Invalid thread ID")
	}
	if content == "" && !imgonly {
		if image == nil {
			return errors.New("Content cannot be empty")
		} else {
			content = ""
		}
	}
	if author == "" {
		return errors.New("Author cannot be empty")
	}
	var imageURL string
	if image != nil {
		if image.Size > 11<<20 {
			return errors.New("File is too large")
		}
		ext := filepath.Ext(image.Filename)
		if !isValidImageExtension(ext) {
			return errors.New("Invalid image extension")
		}
		imageURL, err = saveImage(boardID, image)
		if err != nil {
			return err
		}
		go images.GenerateThumbnail("boards/"+boardID+"/"+imageURL, "thumbs/"+imageURL, 200, 200)
	}
	AddGlobalPostCount()
	AddBoardPostCount(boardID)
	post := Post{
		BoardID:        boardID,
		ThreadID:       strconv.Itoa(threadID),
		PostID:         postid,
		Content:        content,
		PartialContent: content[:min(len(content), 20)],
		ImageURL:       imageURL,
		ThumbURL:       "thumbs/" + imageURL,
		Author:         author,
		Timestamp:      time.Now().Format("01-02-2006 15:04:05"),
		IP:             c.RealIP(),
		ParentID:       replyto,
	}
	addToRecents(post.PostID)
	if err := addPostToFile(boardID, threadID, post); err != nil {
		return err
	}
	SetSessionSelfPostID(c, post.PostID)
	return nil
}
func CreatePost(c echo.Context) error {
	boardID, content, replyto, author, image, err := extractPostData(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if auth.PremiumCheck(c) {
		limiter := rate.NewLimiter(rate.Every(5*time.Minute), 1)
		if !limiter.Allow() {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "5 Minute cooldown"})
		}
	}
	postID := GenUUID()

	// Save session synchronously before enqueuing the task
	updatedIDs, err := SetSessionSelfPostID(c, postID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update session"})
	}
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get session"})
	}
	sess.Values["self_post_id"] = updatedIDs
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save session"})
	}
	if replyto != "" && !checkReplyID(replyto) {
		return c.JSON(http.StatusBadRequest, "Invalid reply ID")
	}

	q.Enqueue(func() {
		if err := processPost(c, boardID, content, replyto, author, image, postID); err != nil {
			fmt.Println("Error processing post:", err)
		}
	})

	return nil
}

func CreateThreadPost(c echo.Context) error {
	boardID, content, replyto, author, image, err := extractPostData(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if auth.PremiumCheck(c) {
		limiter := rate.NewLimiter(rate.Every(5*time.Minute), 1)
		if !limiter.Allow() {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "5 Minute cooldown"})
		}
	}

	if replyto != "" && !checkReplyID(replyto) {
		return c.JSON(http.StatusBadRequest, "Invalid reply ID")
	}

	// Generate postID separately
	postID := GenUUID()

	// Save session synchronously before enqueuing the task
	updatedIDs, err := SetSessionSelfPostID(c, postID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update session"})
	}

	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get session"})
	}
	sess.Values["self_post_id"] = updatedIDs
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save session"})
	}

	q.Enqueue(func() {
		if err := processPost(c, boardID, content, replyto, author, image, postID); err != nil {
			fmt.Println("Error processing post:", err)
		}
	})

	return nil
}
func checkReplyID(replyID string) bool {
	db := database.DB
	if db.Where("post_id = ?", replyID).First(&Post{}).RowsAffected > 0 {
		return true
	}
	return false
}
func isValidImageExtension(ext string) bool {
	validExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".webm"}
	for _, v := range validExtensions {
		if ext == v {
			return true
		}
	}
	return false
}

func GenUUID() string {
	db := database.DB
	for {
		b := make([]byte, 4)
		_, err := rand.Read(b)
		if err != nil {
			log.Fatalf("Failed to generate random bytes: %v", err)
		}
		id := hex.EncodeToString(b)

		if db.Where("post_id = ?", id).First(&Recents{}).RowsAffected == 0 {
			return id
		}

		log.Printf("UUID collision: %s", id)
		GenUUID()
	}
}

func LatestPostsCheck(c echo.Context, boardID string) bool {
	if CheckIfArchived(boardID) {
		return false
	}
	if CheckIfLocked(boardID) {
		return false
	}
	if CheckIfImageOnly(boardID) {
		return false
	}
	if CheckLatestPosts(boardID) {
		return false
	}
	return true

}

func CheckIfThreadLocked(c echo.Context, boardID string, threadID string) bool {
	filepath := "boards/" + boardID + "/" + threadID + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		return false
	}
	defer file.Close()
	var posts []Post
	gob.NewDecoder(file).Decode(&posts)
	return posts[0].Locked
}

func CheckLatestPosts(boardID string) bool {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.LatestPosts
}

func addPostToFile(boardID string, threadID int, post Post) error {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.New("Error opening file")
	}
	defer file.Close()

	var posts []Post
	if err := gob.NewDecoder(file).Decode(&posts); err != nil {
		return errors.New("Error decoding JSON")
	}

	posts = append(posts, post)

	if err := file.Truncate(0); err != nil {
		return errors.New("Error truncating file")
	}
	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		return errors.New("Error seeking file")
	}
	if err := gob.NewEncoder(file).Encode(posts); err != nil {
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
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		return true
	}
	defer file.Close()
	var posts []Post
	gob.NewDecoder(file).Decode(&posts)
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
			if err := gob.NewDecoder(f).Decode(&posts); err != nil || len(posts) == 0 {
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
func PurgeBoard(boardID string) {
	dir := "boards/" + boardID
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	for _, file := range files {
		if !file.IsDir() {
			filepath := dir + "/" + file.Name()
			os.Remove(filepath)
		}
	}
}
func GetBoards() []Board {
	db := database.DB

	var boards []Board
	db.Find(&boards)

	return boards
}

func GetLatestPosts(n int) ([]RecentPosts, error) {
	db := database.DB

	var posts []RecentPosts
	db.Order("timestamp DESC").Limit(n).Find(&posts)

	return posts, nil
}

func GetBoardName(boardID string) string {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.Name
}

func GetBoard(boardID string) Board {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board
}

func GetBoardID(boardID string) string {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.BoardID
}

// need to fix ai hallucination here
func GetThreads(boardID string) []Post {
	dir := "boards/" + boardID
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Error reading directory %s: %v", dir, err)
		return nil
	}

	type ThreadInfo struct {
		FirstPost     Post
		LastTimestamp time.Time
	}

	var threadInfos []ThreadInfo

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".gob" {
			filePath := filepath.Join(dir, file.Name())
			f, err := os.Open(filePath)
			if err != nil {
				log.Printf("Error opening file %s: %v", filePath, err)
				continue
			}

			var posts []Post
			if err := gob.NewDecoder(f).Decode(&posts); err != nil || len(posts) == 0 {
				f.Close()
				if err != nil {
					log.Printf("Error decoding JSON in file %s: %v", filePath, err)
				}
				continue
			}
			f.Close()

			// Parse the last post's timestamp
			lastPost := posts[len(posts)-1]
			lastTimestamp, err := time.Parse("01-02-2006 15:04:05", lastPost.Timestamp)
			if err != nil {
				log.Printf("Error parsing timestamp for file %s: %v", filePath, err)
				continue
			}

			// Append ThreadInfo with the first post and the last timestamp
			threadInfos = append(threadInfos, ThreadInfo{
				FirstPost:     posts[0],
				LastTimestamp: lastTimestamp,
			})
		}
	}

	// Sort threadInfos slice based on lastTimestamp in descending order
	sort.Slice(threadInfos, func(i, j int) bool {
		return threadInfos[i].LastTimestamp.After(threadInfos[j].LastTimestamp)
	})

	// Extract the sorted first posts
	var sortedPosts []Post
	for _, info := range threadInfos {
		sortedPosts = append(sortedPosts, info.FirstPost)
	}

	return sortedPosts
}

func SetSessionSelfPostID(c echo.Context, postID string) ([]string, error) {
	sess, err := session.Get("session", c)
	if err != nil {
		return nil, err
	}

	var updatedIDs []string
	if ids, ok := sess.Values["self_post_id"].([]string); ok {
		updatedIDs = append(ids, postID)
	} else {
		updatedIDs = []string{postID}
	}

	return updatedIDs, nil
}

func GetPosts(boardID string, threadID int) []Post {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		return nil
	}
	defer file.Close()
	var posts []Post
	gob.NewDecoder(file).Decode(&posts)

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
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		return Post{}
	}
	defer file.Close()
	var posts []Post
	err = gob.NewDecoder(file).Decode(&posts)
	if err != nil || len(posts) == 0 {
		return Post{} // Return an empty Post if there's an error or the array is empty
	}
	return posts[0] // Return the first post
}

func AddRecentPost(post Post) {
	db := database.DB

	var count int64
	db.Model(&RecentPosts{}).Count(&count)
	if count >= 10 {
		var oldestPost RecentPosts
		db.Order("timestamp ASC").First(&oldestPost)
		// Assuming RecentPosts has an ID field that serves as the primary key
		db.Where("id = ?", oldestPost.ID).Delete(&RecentPosts{})
	}

	if len(post.Subject) > 30 {
		post.Subject = post.Subject[:30]
	}
	recentPost := RecentPosts{
		BoardID:        post.BoardID,
		ThreadID:       post.ThreadID,
		PostID:         post.PostID,
		Content:        post.Content,
		PartialContent: post.PartialContent,
		ImageURL:       post.ImageURL,
		Subject:        post.Subject,
		Timestamp:      post.Timestamp,
	}
	db.Create(&recentPost)
}

func CheckIfLocked(boardID string) bool {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	// Ensure the database is closed after all operations are done
	return board.Locked
}

func CheckIfArchived(boardID string) bool {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	// Ensure the database is closed after all operations are done
	return board.Archived
}

func CheckIfImageOnly(boardID string) bool {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	// Ensure the database is closed after all operations are done
	return board.ImageOnly
}

func ThreadCheckLocked(c echo.Context, boardid string, threadid string) bool {
	filepath := "boards/" + boardid + "/" + threadid + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		return false
	}
	defer file.Close()
	var posts []Post
	gob.NewDecoder(file).Decode(&posts)
	return posts[0].Locked
}

func AddGlobalPostCount() int64 {
	db := database.DB

	var postCounter PostCounter
	db.First(&postCounter)
	postCounter.PostCount++
	db.Save(&postCounter)
	return postCounter.PostCount
}
func GetGlobalPostCount() int64 {
	db := database.DB

	var postCounter PostCounter
	db.First(&postCounter)
	return postCounter.PostCount
}

func AddBoardPostCount(boardID string) {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	board.PostCount++
	db.Where("board_id = ?", boardID).Save(&board)
}

func GetBoardPostCount(boardID string) int64 {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.PostCount
}

func GetPartialPosts(boardID string, threadID int, postid int) []Post {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		return nil
	}
	defer file.Close()
	var posts []Post
	gob.NewDecoder(file).Decode(&posts)

	// get partial post
	var partialPosts []Post
	for i := range posts {
		if posts[i].PostID > string(postid) {
			// Check if the post content is longer than 20 characters
			if len(posts[i].Content) > 20 {
				// Truncate the content to the first 20 characters
				posts[i].Content = posts[i].Content[:20]
			}
			partialPosts = append(partialPosts, posts[i])
		}
	}
	return partialPosts
}

func GetBoardDescription(boardID string) string {
	db := database.DB

	var board Board
	db.Where("board_id = ?", boardID).First(&board)
	return board.Description
}

func GetTotalPostCount() int64 {
	db := database.DB

	var PostCount PostCounter
	db.First(&PostCount)
	return PostCount.PostCount
}
