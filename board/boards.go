package board

import (
	"encoding/gob"
	"encoding/hex"
	"errors"
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
	ImageOnly   bool   `gob:"image_only"`   //todo
	Locked      bool   `gob:"locked"`       //todo
	Archived    bool   `gob:"archived"`     //todo
	LatestPosts bool   `gob:"latest_posts"` //todo
	Pages       int    `gob:"pages"`        //todo
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

type PostCounter struct {
	ID        int   `gob:"ID" gorm:"primaryKey"`
	PostCount int64 `gob:"PostCount" gorm:"default:0"`
}

func init() {
	db := database.DB

	db.AutoMigrate(&Board{})
	db.AutoMigrate(&RecentPosts{})
	db.AutoMigrate(&PostCounter{})

}

func CreateThread(c echo.Context) error {
	if auth.PremiumCheck(c) {
		var limiter = rate.NewLimiter(rate.Every(5*time.Minute), 1)
		if !limiter.Allow() {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "5 Minute cooldown"})
		}
	}

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
	imgonly := CheckIfImageOnly(boardID)
	if imgonly && c.FormValue("content") != "" {
		return c.JSON(http.StatusBadRequest, "This board only allows image posts")
	}
	// get thread id
	boardDir := "boards/" + boardID
	// scan all files in the board directory each gob file is a thread, titled with an integer
	files, err := ioutil.ReadDir(boardDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error2")
	}

	// get the thread id by counting the number of gob files in the board directory
	threadID := 1
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".gob") {
			threadID++
		}
	}
	// if 30 threads already exist, return an error
	if threadID > 30 {
		DeleteLastThread(boardID)
	}
	// get post content
	content := c.FormValue("content")
	if content == "" && !imgonly {
		return c.JSON(http.StatusBadRequest, "Content cannot be empty")
	}
	// get post subject
	subject := c.FormValue("subject")
	if subject == "" {
		subject = "No Subject"
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
	// max filesize
	if image != nil && image.Size > 11<<20 {
		return c.JSON(http.StatusBadRequest, "File is too large")
	}
	// make sure image is only the following formats, gif, jpg, jpeg, png, webm, mp4, webp, pdf
	if image != nil {
		imageExt := filepath.Ext(image.Filename)
		if imageExt != ".gif" && imageExt != ".jpg" && imageExt != ".jpeg" && imageExt != ".png" && imageExt != ".webm" && imageExt != ".mp4" && imageExt != ".webp" && imageExt != ".pdf" {
			return c.JSON(http.StatusBadRequest, "Invalid image extension")
		}
	}

	imageURL, err := saveImage(boardID, image)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	go images.GenerateThumbnail("boards/"+boardID+"/"+imageURL, "thumbs/"+imageURL, 200, 200)
	// save the image to the board directory
	// /img/:b/:f
	sticky := false
	if !auth.AdminCheck(c) {
		stickyValue := c.FormValue("isSticky")
		if stickyValue == "on" {
			sticky = true
		}
	}
	if !auth.JannyCheck(c, boardID) {
		stickyValue := c.FormValue("isSticky")
		if stickyValue == "on" {
			sticky = true
		}
	}
	locked := false
	if !auth.AdminCheck(c) {
		lockedValue := c.FormValue("isLocked")
		if lockedValue == "on" {
			locked = true
		}
	}
	if !auth.JannyCheck(c, boardID) {
		lockedValue := c.FormValue("isLocked")
		if lockedValue == "on" {
			locked = true
		}
	}
	AddGlobalPostCount()
	AddBoardPostCount(boardID)

	// Use the safeEndIndex for slicing content

	safeEndIndex := 20
	if len(content) < safeEndIndex {
		safeEndIndex = len(content)
	}

	post := Post{
		BoardID:        boardID,
		ThreadID:       strconv.Itoa(threadID),
		PostID:         GenUUID(),
		Content:        content,
		PartialContent: content[:safeEndIndex],
		ImageURL:       imageURL,
		ThumbURL:       "thumbs/" + imageURL,
		Subject:        subject,
		Author:         author,
		Timestamp:      time.Now().Format("01-02-2006 15:04:05"),
		IP:             c.RealIP(),
		Sticky:         sticky,
		Locked:         locked,
	}
	SetSessionSelfPostID(c, post.PostID)
	// create a gob file for the thread
	jsonFilePath := boardDir + "/" + strconv.Itoa(threadID) + ".gob"
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
	if err := gob.NewEncoder(file).Encode(posts); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error encoding JSON")
	}

	boardName := url.PathEscape(c.Param("b"))
	threadIDStr := strconv.Itoa(threadID)
	redirectURL := "/board/" + boardName + "/" + threadIDStr
	if LatestPostsCheck(c, boardID) {
		AddRecentPost(post)
	}
	sitemap := sitemap.Sitemap{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}
	sitemap.AddURL("https://achan.moe/board/"+boardName+"/"+threadIDStr, "daily", "0.5")
	return c.Redirect(http.StatusFound, redirectURL)
}
func checkReplyID(id string) bool {
	db := database.DB
	return db.Where("post_id = ?", id).First(&Post{}).RowsAffected > 0
}

func CreateThreadPost(c echo.Context) error {
	boardID := c.Param("b")
	if auth.PremiumCheck(c) {
		var limiter = rate.NewLimiter(rate.Every(5*time.Minute), 1)
		if !limiter.Allow() {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "5 Minute cooldown"})
		}
	}

	replyto := c.FormValue("replyto")
	if replyto == "" {
		replyto = ""
	}

	if replyto != "" && !checkReplyID(replyto) {
		return c.JSON(http.StatusBadRequest, "Invalid reply ID")
	}

	if boardID == "" {
		return c.JSON(http.StatusBadRequest, "Board ID cannot be empty")
	}
	if CheckIfThreadLocked(c, boardID, c.Param("t")) {
		return c.JSON(http.StatusForbidden, "Thread is locked")
	}

	if CheckIfLocked(boardID) {
		return c.JSON(http.StatusForbidden, "Board is locked")
	}

	if CheckIfArchived(boardID) {
		return c.JSON(http.StatusForbidden, "Board is archived")
	}
	imgonly := CheckIfImageOnly(boardID)
	if imgonly && c.FormValue("content") != "" {
		return c.JSON(http.StatusBadRequest, "This board only allows image posts")
	}

	threadID, err := strconv.Atoi(c.Param("t"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid thread ID")
	}

	content := c.FormValue("content")
	if content == "" && !imgonly {
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
	if image != nil && image.Size > 11<<20 {
		return c.JSON(http.StatusBadRequest, "File is too large")
	}
	if threadIsFull(boardID, threadID) {
		return c.JSON(http.StatusForbidden, "Thread is full")
	}
	if image != nil {
		imageExt := filepath.Ext(image.Filename)
		if imageExt != ".gif" && imageExt != ".jpg" && imageExt != ".jpeg" && imageExt != ".png" && imageExt != ".webm" && imageExt != ".mp4" && imageExt != ".webp" && imageExt != ".pdf" {
			return c.JSON(http.StatusBadRequest, "Invalid image extension")
		}
	}
	imageURL, err := saveImage(boardID, image)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	go images.GenerateThumbnail("boards/"+boardID+"/"+imageURL, "thumbs/"+imageURL, 200, 200)
	AddGlobalPostCount()
	AddBoardPostCount(boardID)

	// Use the safeEndIndex for slicing content
	safeEndIndex := 20
	if len(content) < safeEndIndex {
		safeEndIndex = len(content)
	}
	post := Post{
		BoardID:        boardID,
		ThreadID:       strconv.Itoa(threadID),
		PostID:         GenUUID(),
		Content:        content,
		PartialContent: content[:safeEndIndex],
		ImageURL:       imageURL,
		ThumbURL:       "thumbs/" + imageURL,
		Author:         author,
		Timestamp:      time.Now().Format("01-02-2006 15:04:05"),
		IP:             c.RealIP(),
		ParentID:       replyto,
	}
	if err := addPostToFile(boardID, threadID, post); err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to add post to thread")
	}
	SetSessionSelfPostID(c, post.PostID)
	boardName := url.PathEscape(boardID)
	threadIDStr := strconv.Itoa(threadID)
	redirectURL := "/board/" + boardName + "/" + threadIDStr
	if LatestPostsCheck(c, boardID) {
		AddRecentPost(post)
	}

	return c.Redirect(http.StatusFound, redirectURL)
}
func GenUUID() string {
	b := make([]byte, 4) // 16 bytes = 128 bits
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}
	id := hex.EncodeToString(b)
	db := database.DB
	if db.Where("post_id = ?", id).First(&Post{}).RowsAffected > 0 {
		return GenUUID()
	}
	return id
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

func SetSessionSelfPostID(c echo.Context, postID string) {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Printf("Error getting session: %v", err)
		return
	}

	// Debugging the session values
	log.Printf("Session before update: %v", sess.Values)

	// Check if the key exists and is an array
	if ids, ok := sess.Values["self_post_id"].([]string); ok {
		sess.Values["self_post_id"] = append(ids, postID)
	} else {
		sess.Values["self_post_id"] = []string{postID}
	}

	// Debugging the session values after update
	log.Printf("Session after update: %v", sess.Values)

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Error saving session: %v", err)
	}
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
