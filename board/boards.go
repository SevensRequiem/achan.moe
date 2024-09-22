package board

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
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
	"achan.moe/logs"
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
	TrueUser       string `gob:"TrueUser"`
	ParentID       string `gob:"ParentID"`
	Timestamp      string `gob:"Timestamp"`
	IP             string `gob:"IP"`
	Sticky         bool   `gob:"Sticky"`
	Locked         bool   `gob:"Locked"`
	PostCount      int    `gob:"PostCount"`
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
	TrueUser       string `gob:"TrueUser"`
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

var User = auth.User{}
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

func extractThreadData(c echo.Context) (string, string, string, string, string, string, *multipart.FileHeader, error) {
	boardID := c.Param("b")
	content := c.FormValue("content")
	subject := c.FormValue("subject")
	author := c.FormValue("author")
	image, err := c.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		logs.Error("Error getting image file: %v", err)
		return "", "", "", "", "", "", nil, err
	}

	trueuser := "Anonymous"
	if auth.AuthCheck(c) {
		trueuser = auth.LoggedInUser(c).UUID
	}

	sanitize(content, subject, author)
	return boardID, content, subject, author, trueuser, c.FormValue("isSticky"), image, nil
}

func processThread(c echo.Context, boardID, content, subject, author, trueuser, stickyValue string, image *multipart.FileHeader) error {
	if boardID == "" {
		return c.JSON(http.StatusBadRequest, "Board ID cannot be empty")
	}
	if CheckIfLocked(boardID) {
		return c.JSON(http.StatusBadRequest, "Board is locked")
	}
	if CheckIfArchived(boardID) {
		return c.JSON(http.StatusBadRequest, "Board is archived")
	}
	imgonly := CheckIfImageOnly(boardID)
	if imgonly && content != "" {
		return c.JSON(http.StatusBadRequest, "This board only allows image posts")
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
			return c.JSON(http.StatusBadRequest, "Content cannot be empty")
		} else {
			content = ""
		}
	}
	if subject == "" {
		subject = "No Subject"
	}
	if author == "" {
		return c.JSON(http.StatusBadRequest, "Author cannot be empty")
	}
	if image == nil {
		return c.JSON(http.StatusBadRequest, "Image is required for threads")
	}
	if image.Size > 11<<20 {
		return c.JSON(http.StatusBadRequest, "File is too large")
	}
	ext := filepath.Ext(image.Filename)
	if !isValidImageExtension(ext) {
		return c.JSON(http.StatusBadRequest, "Invalid image extension")
	}
	// After the image is saved, generate the thumbnail
	imageURL, err := saveImage(boardID, image)
	if err != nil {
		return err
	}

	// Construct the full input path for the thumbnail generation
	inputImagePath := "boards/" + boardID + "/" + imageURL
	ext = filepath.Ext(imageURL)
	trimmedURL := strings.TrimSuffix(imageURL, ext)
	thumbnailPath := "thumbs/" + trimmedURL + ".jpg"

	// Generate the thumbnail
	if err := images.GenerateThumbnail(inputImagePath, thumbnailPath, 200, 200); err != nil {
		logs.Error("Error generating thumbnail: %v", err)
		return fmt.Errorf("thumbnail generation failed: %w", err)
	}

	AddGlobalPostCount()
	AddBoardPostCount(boardID)
	sticky := stickyValue == "on"
	locked := false
	if !auth.AdminCheck(c) {
		lockedValue := c.FormValue("isLocked")
		locked = lockedValue == "on"
	}
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
		ThumbURL:       trimmedURL + ".jpg",
		Subject:        subject,
		Author:         author,
		TrueUser:       trueuser,
		Timestamp:      time.Now().Format("01-02-2006 15:04:05"),
		IP:             c.RealIP(),
		Sticky:         sticky,
		Locked:         locked,
	}

	addToRecents(post.PostID)

	boardDir := "boards/" + boardID
	jsonFilePath := boardDir + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.OpenFile(jsonFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logs.Error("Error opening file: %v", err)
		return err
	}
	defer file.Close()
	var posts []Post
	if err := gob.NewDecoder(file).Decode(&posts); err != nil {
		if err.Error() != "EOF" {
			logs.Error("Error decoding JSON: %v", err)
			return err
		}
	}
	posts = append(posts, post)
	if err := file.Truncate(0); err != nil {
		logs.Error("Error truncating file: %v", err)
		return err
	}
	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		logs.Error("Error seeking file: %v", err)
		return err
	}
	if err := gob.NewEncoder(file).Encode(posts); err != nil {
		logs.Error("Error encoding JSON: %v", err)
		return err
	}
	if LatestPostsCheck(c, boardID) {
		AddRecentPost(post)
	}
	logs.Info("Thread created: %v", post.BoardID, post.ThreadID)
	sitemap := sitemap.Sitemap{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	sitemap.AddURL("https://achan.moe/board/"+url.PathEscape(boardID)+"/"+strconv.Itoa(threadID), "daily", "0.5")
	return nil
}

func CreateThread(c echo.Context) error {
	if auth.PremiumCheck(c) {
		limiter := rate.NewLimiter(rate.Every(5*time.Minute), 1)
		if !limiter.Allow() {
			logs.Debug("5 Minute cooldown")
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "5 Minute cooldown"})
		}
	}

	boardID, content, subject, author, trueuser, stickyValue, image, err := extractThreadData(c)
	if err != nil {
		logs.Error("Error extracting thread data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	q.Enqueue(func() {
		if err := processThread(c, boardID, content, subject, author, trueuser, stickyValue, image); err != nil {
			logs.Error("Error processing thread: %v", err)
		}
	})

	return nil
}
func addToRecents(postID string) {
	db := database.DB
	db.Create(&Recents{PostID: postID})
}
func extractPostData(c echo.Context) (string, string, string, string, string, *multipart.FileHeader, error) {
	boardID := c.Param("b")
	content := c.FormValue("content")
	author := c.FormValue("author")
	image, err := c.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		logs.Error("Error getting image file: %v", err)
		return "", "", "", "", "", nil, err
	}

	trueuser := "Anonymous"
	if auth.AuthCheck(c) {
		trueuser = auth.LoggedInUser(c).UUID
	}

	sanitize(content, "", author)

	return boardID, content, c.FormValue("replyto"), author, trueuser, image, nil
}

func processPost(c echo.Context, boardID, content, replyto, author, trueuser string, image *multipart.FileHeader, postid string) error {
	if boardID == "" {
		logs.Error("Board ID cannot be empty")
		return c.JSON(http.StatusBadRequest, "Board ID cannot be empty")
	}
	if CheckIfThreadLocked(c, boardID, c.Param("t")) {
		logs.Error("Thread is locked")
		return c.JSON(http.StatusBadRequest, "Thread is locked")
	}
	if CheckIfLocked(boardID) {
		logs.Error("Board is locked")
		return c.JSON(http.StatusBadRequest, "Board is locked")
	}
	if CheckIfArchived(boardID) {
		logs.Error("Board is archived")
		return c.JSON(http.StatusBadRequest, "Board is archived")
	}
	imgonly := CheckIfImageOnly(boardID)
	if imgonly && content != "" {
		logs.Error("This board only allows image posts")
		return c.JSON(http.StatusBadRequest, "This board only allows image posts")
	}
	threadID, err := strconv.Atoi(c.Param("t"))
	if err != nil {
		logs.Error("Invalid thread ID")
		return c.JSON(http.StatusBadRequest, "Invalid thread ID")
	}
	if threadIsFull(boardID, threadID) {
		logs.Error("Thread is full")
		return c.JSON(http.StatusBadRequest, "Thread is full")
	}
	if content == "" && !imgonly {
		if image == nil {
			logs.Error("Content cannot be empty")
			return c.JSON(http.StatusBadRequest, "Content cannot be empty")
		} else {
			content = ""
		}
	}
	if author == "" {
		logs.Error("Author cannot be empty")
		return c.JSON(http.StatusBadRequest, "Author cannot be empty")
	}
	var imageURL string
	if image != nil {
		if image.Size > 11<<20 {
			logs.Error("File is too large")
			return c.JSON(http.StatusBadRequest, "File is too large")
		}
		ext := filepath.Ext(image.Filename)
		if !isValidImageExtension(ext) {
			logs.Error("Invalid image extension")
			return c.JSON(http.StatusBadRequest, "Invalid image extension")
		}
		imageURL, err = saveImage(boardID, image)
		if err != nil {
			logs.Error("Error saving image: %v", err)
			return err
		}
		go images.GenerateThumbnail("boards/"+boardID+"/"+imageURL, "thumbs/"+imageURL, 200, 200)
	}
	AddGlobalPostCount()
	AddBoardPostCount(boardID)
	AddThreadPostCount(boardID, threadID)
	post := Post{
		BoardID:        boardID,
		ThreadID:       strconv.Itoa(threadID),
		PostID:         postid,
		Content:        content,
		PartialContent: content[:min(len(content), 20)],
		ImageURL:       imageURL,
		ThumbURL:       "thumbs/" + imageURL,
		Author:         author,
		TrueUser:       trueuser,
		Timestamp:      time.Now().Format("01-02-2006 15:04:05"),
		IP:             c.RealIP(),
		ParentID:       replyto,
	}
	addToRecents(post.PostID)
	if err := addPostToFile(boardID, threadID, post); err != nil {
		logs.Error("Error adding post to file: %v", err)
		return err
	}
	SetSessionSelfPostID(c, post.PostID)
	logs.Info("Post created: %v", post.BoardID, post.ThreadID, post.PostID)
	return nil
}
func CreatePost(c echo.Context) error {
	boardID, content, replyto, author, trueuser, image, err := extractPostData(c)
	if err != nil {
		logs.Error("Error extracting post data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if auth.PremiumCheck(c) {
		limiter := rate.NewLimiter(rate.Every(5*time.Minute), 1)
		if !limiter.Allow() {
			logs.Debug("5 Minute cooldown")
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "5 Minute cooldown"})
		}
	}
	postID := GenUUID()

	// Save session synchronously before enqueuing the task
	updatedIDs, err := SetSessionSelfPostID(c, postID)
	if err != nil {
		logs.Error("Failed to update session: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update session"})
	}
	sess, err := session.Get("session", c)
	if err != nil {
		logs.Error("Failed to get session: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get session"})
	}
	sess.Values["self_post_id"] = updatedIDs
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		logs.Error("Failed to save session: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save session"})
	}
	if replyto != "" && !checkReplyID(replyto) {
		logs.Error("Invalid reply ID")
		return c.JSON(http.StatusBadRequest, "Invalid reply ID")
	}

	q.Enqueue(func() {
		if err := processPost(c, boardID, content, replyto, author, trueuser, image, postID); err != nil {
			logs.Error("Error processing post: %v", err)
		}
	})

	return nil
}

func checkReplyID(replyID string) bool {
	db := database.DB
	return db.Where("post_id = ?", replyID).First(&Post{}).RowsAffected > 0
}
func isValidImageExtension(ext string) bool {
	validExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".webm"}
	for _, v := range validExtensions {
		if ext == v {
			logs.Debug("Valid image extension")
			return true
		}
	}
	return false
}

func GenUUID() string {
	db := database.DB
	for {
		rand.Seed(uint64(time.Now().UnixNano()))
		b := make([]byte, 4)
		_, err := rand.Read(b)
		if err != nil {
			logs.Fatal("Error generating UUID: %v", err)
		}
		id := hex.EncodeToString(b)

		if db.Where("post_id = ?", id).First(&Recents{}).RowsAffected == 0 {
			return id
		}

		logs.Debug("UUID collision, retrying")
		GenUUID()
	}
}

func LatestPostsCheck(c echo.Context, boardID string) bool {
	if CheckLatestPosts(boardID) {
		logs.Debug("Latest posts enabled")
		return false
	}
	logs.Debug("Latest posts disabled")
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
	logs.Debug("Thread locked: %v", posts[0].Locked)
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
		logs.Error("Error opening file: %v", err)
		return err
	}
	defer file.Close()

	var posts []Post
	if err := gob.NewDecoder(file).Decode(&posts); err != nil {
		logs.Error("Error decoding JSON: %v", err)
		return err
	}

	posts = append(posts, post)

	if err := file.Truncate(0); err != nil {
		logs.Error("Error truncating file: %v", err)
		return err
	}
	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		logs.Error("Error seeking file: %v", err)
		return err
	}
	if err := gob.NewEncoder(file).Encode(posts); err != nil {
		logs.Error("Error encoding JSON: %v", err)
		return err
	}

	return nil
}

func saveImage(boardID string, image *multipart.FileHeader) (string, error) {
	if image == nil {
		logs.Error("Image is nil")
		return "", nil
	}
	imagename := uuid.New().String()
	imageExt := filepath.Ext(image.Filename)
	if imageExt == "" {
		logs.Error("Invalid image extension")
		return "", fmt.Errorf("invalid image extension")
	}

	imageFile, err := image.Open()
	if err != nil {
		logs.Error("Error opening image file: %v", err)
		return "", fmt.Errorf("error opening image file: %v", err)
	}
	defer imageFile.Close()

	imageData, err := ioutil.ReadAll(imageFile)
	if err != nil {
		logs.Error("Error reading image file: %v", err)
		return "", fmt.Errorf("error reading image file: %v", err)
	}

	imageURL := imagename + imageExt
	baseImgDir := "boards/" + boardID + "/" + imagename + imageExt
	if err := ioutil.WriteFile(baseImgDir, imageData, 0644); err != nil {
		logs.Error("Error writing image file: %v", err)
		return "", fmt.Errorf("error writing image file: %v", err)
	}

	return imageURL, nil
}

func threadIsFull(boardID string, threadID int) bool {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		logs.Error("Error opening file: %v", err)
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
		logs.Error("Error reading directory: %v", err)
		return
	}
	var oldestThread string
	oldestTime := time.Now()
	for _, file := range files {
		if !file.IsDir() {
			filepath := dir + "/" + file.Name()
			f, err := os.Open(filepath)
			if err != nil {
				logs.Error("Error opening file: %v", err)
				continue
			}
			var posts []Post
			if err := gob.NewDecoder(f).Decode(&posts); err != nil || len(posts) == 0 {
				logs.Error("Error decoding JSON: %v", err)
				f.Close()
				continue
			}
			f.Close()
			timestamp, err := time.Parse("01-02-2006 15:04:05", posts[0].Timestamp)
			if err != nil {
				logs.Error("Error parsing timestamp: %v", err)
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
		logs.Error("Error reading directory: %v", err)
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
		logs.Error("Error reading directory: %v", err)
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
				logs.Error("Error opening file %s: %v", filePath, err)
				continue
			}

			var posts []Post
			if err := gob.NewDecoder(f).Decode(&posts); err != nil || len(posts) == 0 {
				f.Close()
				if err != nil {
					logs.Error("Error decoding JSON for file %s: %v", filePath, err)
				}
				continue
			}
			f.Close()

			// Parse the last post's timestamp
			lastPost := posts[len(posts)-1]
			lastTimestamp, err := time.Parse("01-02-2006 15:04:05", lastPost.Timestamp)
			if err != nil {
				logs.Error("Error parsing timestamp for file %s: %v", filePath, err)
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
		logs.Error("Failed to get session: %v", err)
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
		logs.Error("Error opening file: %v", err)
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
		logs.Error("Error decoding GOB: %v", err)
		return Post{} // Return an empty Post if there's an error or the array is empty
	}
	logs.Debug("GetThread: %v", posts[0])
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
		ThumbURL:       post.ThumbURL,
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
		logs.Error("Error opening file: %v", err)
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
		logs.Error("Error opening file: %v", err)
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

func ReportThread(c echo.Context) error {
	db := database.DB

	threadID := c.Param("t")
	boardID := c.Param("b")
	filepath := "boards/" + boardID + "/" + threadID + ".gob"
	file, err := os.Open(filepath)
	if err != nil {
		logs.Error("Error opening file: %v", err)
		return err
	}
	defer file.Close()
	var posts []Post
	gob.NewDecoder(file).Decode(&posts)
	posts[0].ReportCount++
	db.Save(&posts[0])

	return nil
}

func ReportPost(c echo.Context) error {
	db := database.DB

	postID := c.Param("p")
	var post Post
	db.Where("post_id = ?", postID).First(&post)
	post.ReportCount++
	db.Save(&post)

	return nil
}

func DeleteThread(c echo.Context) error {
	threadID := c.Param("t")
	board := c.Param("b")

	// Delete RecentPosts entries
	RemoveFromRecentPosts("", threadID, board)

	// Construct the path to the thread's JSON file
	threadFilePath := filepath.Join("boards/" + board + "/" + threadID + ".gob")

	// Open and decode the thread's JSON file
	gobFile, err := os.Open(threadFilePath)
	if err != nil {
		logs.Error("Failed to open thread file: %v", err)
		return c.JSON(http.StatusInternalServerError, "Internal server error: unable to open thread file")
	}
	defer gobFile.Close()

	var posts []Post
	if err := gob.NewDecoder(gobFile).Decode(&posts); err != nil {
		logs.Error("Failed to decode thread file: %v", err)
		return c.JSON(http.StatusInternalServerError, "Internal server error: unable to decode thread file: "+err.Error())
	}

	// Delete images associated with the thread
	for _, post := range posts {
		if post.ImageURL != "" {
			if err := os.Remove("boards/" + board + "/" + post.ImageURL); err != nil {
				logs.Error("Failed to delete image: %v", err)
			}
			if err := os.Remove("thumbs/" + post.ThumbURL); err != nil {
				logs.Error("Failed to delete thumbnail: %v", err)
			}
		}
	}

	// Delete the thread's JSON file
	if err := os.Remove(threadFilePath); err != nil {
		logs.Error("Failed to delete thread file: %v", err)
		return c.JSON(http.StatusInternalServerError, "Internal server error: failed to delete thread file")
	}

	return c.JSON(http.StatusOK, "Thread deleted")
}
func DeletePost(c echo.Context) error {
	postid := c.Param("p")
	threadid := c.Param("t")
	board := c.Param("b")

	// Construct the file path
	filePath := "boards/" + board + "/" + threadid + ".gob"
	// Open the JSON file
	gobFile, err := os.Open(filePath)
	if err != nil {
		logs.Error("Failed to open file: %s, error: %v", filePath, err)
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	defer gobFile.Close()

	// Decode the JSON file into posts
	var posts []Post
	if err := gob.NewDecoder(gobFile).Decode(&posts); err != nil {
		logs.Error("Failed to decode JSON: %v", err)
		return c.JSON(http.StatusInternalServerError, "Error decoding JSON")
	}

	// Find and delete the post
	for i, post := range posts {
		if post.PostID == postid {
			posts = append(posts[:i], posts[i+1:]...)
			break
		}
	}

	// delete image if exists
	var imageURL string
	var thumbURL string
	for _, post := range posts {
		if post.PostID == postid {
			imageURL = post.ImageURL
			thumbURL = post.ThumbURL
			break
		}
	}
	if imageURL != "" {
		// skip if null
		if err := os.Remove("boards/" + board + "/" + imageURL); err != nil {
			logs.Error("Failed to delete image: %v", err)
			return c.JSON(http.StatusInternalServerError, "Failed to delete image")
		}

		if err := os.Remove("thumbs/" + thumbURL); err != nil {
			logs.Error("Failed to delete thumbnail: %v", err)
			return c.JSON(http.StatusInternalServerError, "Failed to delete thumbnail")
		}
	}

	// Database operations (omitted for brevity)
	RemoveFromRecentPosts(postid, "", "")
	// Recreate the JSON file to update it
	gobFile, err = os.Create(filePath)
	if err != nil {
		logs.Error("Failed to create file: %s, error: %v", filePath, err)
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	defer gobFile.Close()

	// Encode the updated posts back into the JSON file
	if err := gob.NewEncoder(gobFile).Encode(posts); err != nil {
		logs.Error("Failed to encode JSON: %v", err)
		return c.JSON(http.StatusInternalServerError, "Error encoding JSON")
	}

	logs.Debug("Post deleted: %s", postid)
	return c.JSON(http.StatusOK, "Post deleted")
}

func RemoveFromRecentPosts(postID string, threadID string, board string) {
	// Check if both postID and threadID are zero, return early if true
	if postID == "" && threadID == "" && board == "" {
		logs.Warn("No postID, threadID, or boardID provided")
		return
	}

	// Open database connection
	db := database.DB

	// Construct the query
	if postID != "" {
		db.Where("post_id = ?", postID).Delete(&RecentPosts{})
	} else {
		db.Where("thread_id = ? AND board_id = ?", threadID, board).Delete(&RecentPosts{})
	}

	logs.Debug("Removed from RecentPosts: %s, %s, %s", postID, threadID, board)

}

// Helper function to remove a recent post by a specific field (e.g., post_id or thread_id)

func AddThreadPostCount(boardID string, threadID int) {
	filepath := "boards/" + boardID + "/" + strconv.Itoa(threadID) + ".gob"
	file, err := os.OpenFile(filepath, os.O_RDWR, 0644)
	if err != nil {
		logs.Error("Error opening file: %v", err)
		return
	}
	defer file.Close()

	var thread []Post
	if err := gob.NewDecoder(file).Decode(&thread); err != nil {
		logs.Error("Error decoding GOB: %v", err)
		return
	}

	if len(thread) > 0 {
		thread[0].PostCount++
	}

	// Move the file pointer to the beginning of the file
	if _, err := file.Seek(0, 0); err != nil {
		logs.Error("Error seeking file: %v", err)
		return
	}

	if err := gob.NewEncoder(file).Encode(&thread); err != nil {
		logs.Error("Error encoding JSON: %v", err)
		return
	}

	logs.Debug("Post count incremented: %v", boardID, threadID)
}

func sanitize(content string, subject string, author string) (string, string, string) {
	replacements := map[string]string{
		"'":  "&#39;",
		"\"": "&#34;",
		">":  "&#62;",
		"<":  "&#60;",
		"(":  "&#40;",
		")":  "&#41;",
		";":  "&#59;",
		"/*": "&#47;&#42;",
		"*/": "&#42;&#47;",
		"--": "&#45;&#45;",
	}

	escapeAndReplace := func(input string) string {
		input = template.HTMLEscapeString(input)
		for old, new := range replacements {
			input = strings.ReplaceAll(input, old, new)
		}
		return input
	}

	content = escapeAndReplace(content)
	if len(subject) > 30 {
		subject = subject[:30]
	}
	author = escapeAndReplace(author)
	subject = escapeAndReplace(subject)
	logs.Debug("Sanitized content")
	return content, subject, author
}
