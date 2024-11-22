package board

import (
	"context"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"achan.moe/boardimages"
	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/cache"
	"achan.moe/utils/queue"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/rand"
)

var User = models.User{}
var client = database.Client

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

func GenUUID(boardid string) string {
	if boardid == "" {
		logs.Fatal("Error: boardid cannot be empty")
	}

	db := database.Client.Database(boardid)
	ctx := context.Background()
	cursor, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding collections: %v", err)
		return ""
	}
	defer cursor.Close(ctx)
	id := genrandid()

	for cursor.Next(ctx) {
		var collection struct {
			Name string `bson:"name"`
		}
		if err := cursor.Decode(&collection); err != nil {
			logs.Error("Error decoding collection: %v", err)
			return ""
		}
		if collection.Name == id {
			id = genrandid()
			cursor.Close(ctx)
			cursor, err = db.ListCollections(ctx, bson.M{})
			if err != nil {
				logs.Error("Error finding collections: %v", err)
				return ""
			}
			continue
		}
	}

	return id
}

func genrandid() string {
	rand.Seed(uint64(time.Now().UnixNano()))
	var b [4]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
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
	db := database.DB_Boards.Collection(boardID)
	var threadpost models.ThreadPost
	db.FindOne(context.Background(), bson.M{"boardid": boardID, "thread_id": threadID, "locked": true}).Decode(&threadpost)
	return threadpost.Locked
}

func CheckLatestPosts(boardID string) bool {
	db := database.DB_Main.Collection("recent_posts")
	var board models.Board
	db.FindOne(context.Background(), bson.M{"boardid": boardID}).Decode(&board)
	return board.LatestPosts
}

func threadIsFull(boardID string, threadID int) bool {
	db := client.Database(boardID)
	ctx := context.Background()
	cursor, err := db.Collection(strconv.Itoa(threadID)).Find(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding posts: %v", err)
		return false
	}
	defer cursor.Close(ctx)
	var posts []models.Posts
	for cursor.Next(ctx) {
		var post models.Posts
		if err := cursor.Decode(&post); err != nil {
			logs.Error("Error decoding post: %v", err)
			return false
		}
		posts = append(posts, post)
	}
	if len(posts) >= 500 {
		return true
	}
	return false
}

func DeleteLastThread(c echo.Context, boardID string) {
	db := database.Client.Database(boardID)
	ctx := context.Background()
	cursor, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding collections: %v", err)
		return
	}
	defer cursor.Close(ctx)
	var threads []struct {
		ThreadPost        models.ThreadPost
		LastPostTimestamp int64
	}
	for cursor.Next(ctx) {
		var collection struct {
			Name string `bson:"name"`
		}
		if err := cursor.Decode(&collection); err != nil {
			logs.Error("Error decoding collection: %v", err)
			return
		}
		if collection.Name == "thumbs" || collection.Name == "images" || collection.Name == "banners" {
			continue
		}
		var threadPost models.ThreadPost
		err := db.Collection(collection.Name).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", 1}})).Decode(&threadPost)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			logs.Error("Error finding first document in collection %s: %v", collection.Name, err)
			return
		}
		var lastPost models.ThreadPost
		err = db.Collection(collection.Name).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", -1}})).Decode(&lastPost)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			logs.Error("Error finding last document in collection %s: %v", collection.Name, err)
			return
		}
		threads = append(threads, struct {
			ThreadPost        models.ThreadPost
			LastPostTimestamp int64
		}{
			ThreadPost:        threadPost,
			LastPostTimestamp: lastPost.Timestamp,
		})

		// Delete associated images and thumbnails
		if threadPost.Image != "" {
			_, err := db.Collection("images").DeleteOne(ctx, bson.M{"_id": threadPost.Image})
			if err != nil {
				logs.Error("Error deleting image: %v", err)
			}
		}
		if threadPost.Thumbnail != "" {
			_, err := db.Collection("thumbs").DeleteOne(ctx, bson.M{"_id": threadPost.Thumbnail})
			if err != nil {
				logs.Error("Error deleting thumbnail: %v", err)
			}
		}
	}
}
func PurgeBoard(boardID string) {
	db := client.Database(boardID)
	ctx := context.Background()
	db.Drop(ctx)
}

func GetBoardName(boardID string) string {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	var board models.Board
	db.FindOne(ctx, bson.M{"boardid": boardID}).Decode(&board)
	return board.Name

}

func GetBoard(boardID string) models.Board {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	var board models.Board
	db.FindOne(ctx, bson.M{"boardid": boardID}).Decode(&board)
	return board

}

func GetBoardID(boardID string) string {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	var board models.Board
	db.FindOne(ctx, bson.M{"boardid": boardID}).Decode(&board)
	fmt.Println(board.BoardID)
	return board.BoardID
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

func GetThread(boardID string, threadID string) models.ThreadPost {
	db := client.Database(boardID)
	ctx := context.Background()
	var threadpost models.ThreadPost
	err := db.Collection(threadID).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", 1}})).Decode(&threadpost)
	if err != nil {
		logs.Error("Error finding first document in collection %s: %v", threadID, err)
	}
	return threadpost
}

func AddRecentPost(post models.Posts, threadpost models.ThreadPost) {
	db := database.DB_Main.Collection("recent_posts")
	ctx := context.Background()

	if post != (models.Posts{}) {
		_, err := db.InsertOne(ctx, post)
		if err != nil {
			logs.Error("Error inserting recent post: %v", err)
		}
	} else {
		_, err := db.InsertOne(ctx, threadpost)
		if err != nil {
			logs.Error("Error inserting recent thread post: %v", err)
		}
	}
}

func CheckIfLocked(boardID string) (bool, error) {
	if database.Client == nil {
		return false, errors.New("MongoDB client is not initialized")
	}

	db := database.DB_Main
	collection := db.Collection("boards")

	var board struct {
		Locked bool `bson:"locked"`
	}

	err := collection.FindOne(context.Background(), bson.M{"boardid": boardID}).Decode(&board)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // Board not found, assume not locked
		}
		logs.Error("Error finding board: %v", err)
		return false, err
	}

	return board.Locked, nil
}
func CheckIfArchived(boardID string) (bool, error) {
	if database.Client == nil {
		return false, errors.New("MongoDB client is not initialized")
	}

	db := database.DB_Main
	collection := db.Collection("boards")

	var board struct {
		Archived bool `bson:"archived"`
	}

	err := collection.FindOne(context.Background(), bson.M{"boardid": boardID}).Decode(&board)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // Board not found, assume not archived
		}
		logs.Error("Error finding board: %v", err)
		return false, err
	}

	return board.Archived, nil
}

func CheckIfImageOnly(boardID string) (bool, error) {
	if database.Client == nil {
		return false, errors.New("MongoDB client is not initialized")
	}

	db := database.DB_Main
	collection := db.Collection("boards")

	var board struct {
		ImageOnly bool `bson:"image_only"`
	}

	err := collection.FindOne(context.Background(), bson.M{"boardid": boardID}).Decode(&board)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // Board not found, assume not image only
		}
		logs.Error("Error finding board: %v", err)
		return false, err
	}

	return board.ImageOnly, nil
}

func ThreadCheckLocked(c echo.Context, boardid string, threadid string) bool {
	db := client.Database(boardid)
	ctx := context.Background()
	var threadpost models.ThreadPost
	db.Collection(threadid).FindOne(ctx, bson.M{"thread_id": threadid}).Decode(&threadpost)
	return threadpost.Locked
}

func AddGlobalPostCount() int64 {
	db := database.DB_Main.Collection("data")
	var counter models.PostCounter
	db.FindOne(context.Background(), bson.M{}).Decode(&counter)
	_, err := db.UpdateOne(context.Background(), bson.M{}, bson.M{"$inc": bson.M{"post_count": 1}})
	if err != nil {
		logs.Error("Error incrementing post count: %v", err)
	}
	return counter.PostCount
}
func GetGlobalPostCount() int64 {
	db := database.DB_Main.Collection("data")
	var counter models.PostCounter
	db.FindOne(context.Background(), bson.M{}).Decode(&counter)
	return counter.PostCount

}

func AddBoardPostCount(boardID string) {
	db := database.DB_Main.Collection("boards")
	_, err := db.UpdateOne(context.Background(), bson.M{"boardid": boardID}, bson.M{"$inc": bson.M{"post_count": 1}})
	if err != nil {
		logs.Error("Error incrementing post count: %v", err)
	}
	return
}

func GetBoardPostCount(boardID string) (int, error) {
	if database.Client == nil {
		return 0, errors.New("MongoDB client is not initialized")
	}

	db := database.DB_Main
	collection := db.Collection("boards")

	var board struct {
		PostCount int `bson:"post_count"`
	}

	err := collection.FindOne(context.Background(), bson.M{"boardid": boardID}).Decode(&board)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil // Board not found, assume no posts
		}
		logs.Error("Error finding board: %v", err)
		return 0, err
	}

	return board.PostCount, nil
}

func GetBoardDescription(boardID string) (string, error) {
	if database.Client == nil {
		return "", errors.New("MongoDB client is not initialized")
	}

	db := database.DB_Main
	collection := db.Collection("boards")

	var board struct {
		Description string `bson:"description"`
	}

	err := collection.FindOne(context.Background(), bson.M{"boardid": boardID}).Decode(&board)
	if err != nil {
		logs.Error("Error finding board description: %v", err)
		return "", err
	}

	return board.Description, nil
}

func GetTotalPostCount() int64 {
	boards := models.GetBoards()
	var total int64
	for _, board := range boards {
		db := client.Database(board.BoardID)
		ctx := context.Background()
		cursor, err := db.ListCollections(ctx, bson.M{})
		if err != nil {
			logs.Error("Error finding collections: %v", err)
			return 0
		}
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var collection struct {
				Name string `bson:"name"`
			}
			if err := cursor.Decode(&collection); err != nil {
				logs.Error("Error decoding collection: %v", err)
				return 0
			}
			if collection.Name == "thumbs" || collection.Name == "images" || collection.Name == "banners" {
				continue
			}
			count, err := db.Collection(collection.Name).CountDocuments(ctx, bson.M{})
			if err != nil {
				logs.Error("Error counting documents: %v", err)
				return 0
			}
			total += count
		}
	}
	return total
}

func ReportPost(c echo.Context) error {
	boardid := c.Param("b")
	threadid := c.FormValue("threadid")
	postid := c.FormValue("postid")
	db := client.Database(boardid)
	ctx := context.Background()
	_, err := db.Collection(threadid).UpdateOne(ctx, bson.M{"post_id": postid}, bson.M{"$inc": bson.M{"report_count": 1}})
	if err != nil {
		logs.Error("Error incrementing report count: %v", err)
	}
	return nil
}

func RemoveFromRecentPosts(postID string, threadID string, board string) {
	db := database.DB_Main.Collection("recent_posts")
	ctx := context.Background()
	_, err := db.DeleteOne(ctx, bson.M{"post_id": postID, "thread_id": threadID, "boardid": board})
	if err != nil {
		logs.Error("Error deleting recent post: %v", err)
	}

	return
}

// Helper function to remove a recent post by a specific field (e.g., post_id or thread_id)

func AddThreadPostCount(boardID string, threadID string) {
	db := client.Database(boardID)
	ctx := context.Background()
	filter := bson.M{} // Matches the first document in the collection
	update := bson.M{"$inc": bson.M{"post_count": 1}}
	_, err := db.Collection(threadID).UpdateOne(ctx, filter, update)
	if err != nil {
		logs.Error("Error incrementing post count: %v", err)
	}
	return
}

func GetTotalThreadPostCount(boardID string, threadID string) (int, error) {
	if database.Client == nil {
		return 0, errors.New("MongoDB client is not initialized")
	}

	db := database.Client.Database(boardID)
	collection := db.Collection(threadID)

	filter := bson.M{
		"$nor": []bson.M{
			{"collection": "banners"},
			{"collection": "thumbs"},
			{"collection": "images"},
		},
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		logs.Error("Error counting documents: %v", err)
		return 0, err
	}

	return int(count), nil
}

func GetTotalThreadCount(boardID string) (int, error) {
	if database.Client == nil {
		return 0, errors.New("MongoDB client is not initialized")
	}

	db := database.Client.Database(boardID)
	cursor, err := db.ListCollections(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error finding collections: %v", err)
		return 0, err
	}
	defer cursor.Close(context.Background())

	var count int
	for cursor.Next(context.Background()) {
		count++
	}

	return count, nil
}

func sanitize(content string, subject string, author string) (string, string, string) {
	content = template.HTMLEscapeString(content)
	subject = template.HTMLEscapeString(subject)
	author = template.HTMLEscapeString(author)
	return content, subject, author
}

func CreateThread(c echo.Context) error {
	boardID := c.Param("b")
	locked := cache.CheckBoardLocked(boardID)
	if locked {
		return c.String(http.StatusForbidden, "Board is locked")
	}
	archived := cache.CheckBoardArchived(boardID)
	if archived {
		return c.String(http.StatusForbidden, "Board is archived")
	}

	imageOnly := cache.CheckBoardImageOnly(boardID)
	if imageOnly && c.FormValue("image") == "" || imageOnly && c.FormValue("content") != "" {
		return c.String(http.StatusForbidden, "Board is image only")
	}
	count := cache.GetTotalThreadCount(c, boardID)
	if count >= 30 {
		DeleteLastThread(c, boardID)
	}
	author := c.FormValue("author")
	subject := c.FormValue("subject")
	content := c.FormValue("content")
	imageFile, err := c.FormFile("image")
	if err != nil {
		logs.Error("Error getting image file: %v", err)
		return c.String(http.StatusBadRequest, "Error getting image file")
	}
	if imageFile != nil {
		ext := filepath.Ext(imageFile.Filename)
		if !isValidImageExtension(ext) {
			return c.String(http.StatusBadRequest, "Invalid image extension")
		}
	}
	sticky := false
	if c.FormValue("isSticky") == "on" {
		sticky = true
	}
	locked = false
	if c.FormValue("isLocked") == "on" {
		locked = true
	}
	content, subject, author = sanitize(content, subject, author)

	uuid := GenUUID(boardID)
	threadID := uuid
	postID := uuid
	timestampStr := strconv.FormatInt(time.Now().Unix(), 10)
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error parsing timestamp")
	}
	Thumbnail := ""
	Image := ""
	if imageFile != nil {
		thumb, err := boardimages.SaveThumb(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving thumbnail")
		}
		Thumbnail = thumb
		image, err := boardimages.SaveImage(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving image")
		}
		Image = image
	}

	partialContent := content
	if len(content) > 100 {
		partialContent = content[:100]
	}
	session, err := session.Get("session", c)
	if err != nil {
		logs.Error("Failed to get session: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	var trueuser string

	userSessionValue, ok := session.Values["user"]
	if !ok {
		return nil
	}

	user, ok := userSessionValue.(models.User)
	if !ok {
		return nil
	}

	if user.Username != "" {
		trueuser = user.Username
	} else {
		trueuser = ""
	}

	threadpost := models.ThreadPost{
		Author:         author,
		BoardID:        boardID,
		ThreadID:       threadID,
		PostID:         postID,
		Content:        content,
		PartialContent: partialContent,
		Image:          Image,
		Thumbnail:      Thumbnail,
		Subject:        subject,
		Timestamp:      timestamp,
		Sticky:         sticky,
		Locked:         locked,
		PostCount:      1,
		ReportCount:    0,
		IP:             c.RealIP(),
		TrueUser:       trueuser,
	}

	queue.Q.Enqueue("thread:create", func() { processThreadPost(threadpost, c) })

	return c.JSON(http.StatusOK, "Thread created")
}

func processThreadPost(threadpost models.ThreadPost, c echo.Context) func() {
	globalpostcount := cache.GetGlobalPostCount()
	threadpost.PostNumber = int64(globalpostcount) + 1
	cache.AddThreadToCache(threadpost.BoardID, threadpost)
	cache.AddToRecentThreadsCache(threadpost)
	db := client.Database(threadpost.BoardID)
	ctx := context.Background()
	db.CreateCollection(ctx, threadpost.ThreadID)
	_, err := db.Collection(threadpost.ThreadID).InsertOne(ctx, threadpost)
	if err != nil {
		logs.Error("Error inserting post: %v", err)
		return nil
	}
	AddGlobalPostCount()
	cache.CacheGlobalPostCount()
	AddBoardPostCount(threadpost.BoardID)
	AddRecentPost(models.Posts{}, threadpost)
	SetSessionSelfPostID(c, threadpost.PostID)
	return nil
}

func CreatePost(c echo.Context) error {
	boardID := c.Param("b")
	if boardID == "" {
		return c.String(http.StatusBadRequest, "Board ID cannot be empty")
	}
	locked := cache.CheckBoardLocked(boardID)
	if locked {
		return c.String(http.StatusForbidden, "Board is locked")
	}
	archived := cache.CheckBoardArchived(boardID)
	if archived {
		return c.String(http.StatusForbidden, "Board is archived")
	}

	imageOnly := cache.CheckBoardImageOnly(boardID)
	if imageOnly && c.FormValue("image") == "" || imageOnly && c.FormValue("content") != "" {
		return c.String(http.StatusForbidden, "Board is image only")
	}

	count := cache.GetTotalThreadPostCount(boardID, c.Param("t"))
	if count >= 300 {
		return c.String(http.StatusForbidden, "Thread post limit reached")
	}

	if cache.CheckDuplicatePostContent(boardID, c.Param("t"), c.FormValue("content")) {
		return c.String(http.StatusForbidden, "Duplicate post detected")
	}
	author := c.FormValue("author")
	subject := c.FormValue("subject")
	content := c.FormValue("content")
	imageFile, err := c.FormFile("image")
	if err != nil {
		imageFile = nil
	} else {
		ext := filepath.Ext(imageFile.Filename)
		if !isValidImageExtension(ext) {
			return c.String(http.StatusBadRequest, "Invalid image extension")
		}
	}
	content, subject, author = sanitize(content, subject, author)

	threadID := c.Param("t")
	if threadID == "" {
		return c.String(http.StatusBadRequest, "Thread ID cannot be empty")
	}
	postID := GenUUID(boardID)
	timestampStr := strconv.FormatInt(time.Now().Unix(), 10)
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error parsing timestamp")
	}
	Thumbnail := ""
	Image := ""
	if imageFile != nil {
		thumb, err := boardimages.SaveThumb(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving thumbnail")
		}
		Thumbnail = thumb
		image, err := boardimages.SaveImage(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving image")
		}
		Image = image
	}

	partialContent := content
	if len(content) > 100 {
		partialContent = content[:100]
	}

	session, err := session.Get("session", c)
	if err != nil {
		logs.Error("Failed to get session: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to get session")
	}
	var trueuser string

	userSessionValue, ok := session.Values["user"]
	if !ok {
		return nil
	}

	user, ok := userSessionValue.(models.User)
	if !ok {
		return nil
	}

	if user.Username != "" {
		trueuser = user.Username
	} else {
		trueuser = ""
	}

	post := models.Posts{
		Author:         author,
		BoardID:        boardID,
		ParentID:       threadID,
		PostID:         postID,
		Content:        content,
		PartialContent: partialContent,
		Image:          Image,
		Thumbnail:      Thumbnail,
		Subject:        subject,
		Timestamp:      timestamp,
		IP:             c.RealIP(),
		TrueUser:       trueuser,
	}

	queue.Q.Enqueue("post:create", func() { processPost(post, c) })
	return c.JSON(http.StatusOK, "Post created")
}

func processPost(post models.Posts, c echo.Context) func() {
	globalpostcount := cache.GetGlobalPostCount()
	post.PostNumber = int64(globalpostcount) + 1
	cache.AddPostToThreadCache(post.BoardID, post.ParentID, post)
	cache.AddThreadPostCountToCache(post.BoardID, post.ParentID)
	db := client.Database(post.BoardID)
	ctx := context.Background()
	_, err := db.Collection(post.ParentID).InsertOne(ctx, post)
	if err != nil {
		logs.Error("Error inserting post: %v", err)
		return nil
	}
	AddGlobalPostCount()
	cache.CacheGlobalPostCount()
	AddBoardPostCount(post.BoardID)
	AddThreadPostCount(post.BoardID, post.ParentID)
	SetSessionSelfPostID(c, post.PostID)
	return nil
}

type OldPost struct {
	BoardID        string `bson:"boardid"`
	ParentID       string `bson:"parent_id"`
	ThreadID       string `bson:"thread_id"`
	PostID         string `bson:"post_id"`
	Content        string `bson:"content"`
	PartialContent string `bson:"partial_content"`
	Image          string `bson:"image"`
	ImageURL       string `bson:"image_url"`
	Thumbnail      string `bson:"thumb"`
	Subject        string `bson:"subject"`
	Author         string `bson:"author"`
	Timestamp      string `bson:"timestamp"`
	TrueUser       string `bson:"true_user"`
	IP             string `bson:"ip"`
	Sticky         bool   `bson:"sticky"`
	Locked         bool   `bson:"locked"`
}

func MigrateToMongoFromGob() {
	dirPath := "boards"
	// find all board directories
	boards, err := os.ReadDir(dirPath)
	if err != nil {
		logs.Error("Error reading directory: %v", err)
		return
	}
	for _, board := range boards {
		if !board.IsDir() {
			continue
		}
		boardID := board.Name()
		boardPath := filepath.Join(dirPath, boardID)
		db := client.Database(boardID)
		ctx := context.Background()
		db.CreateCollection(ctx, "thumbs")
		db.CreateCollection(ctx, "images")
		db.CreateCollection(ctx, "banners")
		// find all .gob files in the board directory
		threadFiles, err := os.ReadDir(boardPath)
		if err != nil {
			logs.Error("Error reading board directory: %v", err)
			return
		}
		for _, threadFile := range threadFiles {
			genuuid := GenUUID(boardID)
			if threadFile.IsDir() || filepath.Ext(threadFile.Name()) != ".gob" {
				continue
			}
			// read the .gob file
			file, err := os.Open(filepath.Join(boardPath, threadFile.Name()))
			if err != nil {
				logs.Error("Error opening file: %v", err)
				return
			}
			defer file.Close()
			// decode the .gob file
			dec := gob.NewDecoder(file)
			var oldPosts []OldPost
			err = dec.Decode(&oldPosts)
			if err != nil {
				logs.Error("Error decoding file: %v", err)
				return
			}
			if len(oldPosts) == 0 {
				logs.Error("No posts found in file: %v", threadFile.Name())
				continue
			}
			// create a unique collection for the thread
			collectionName := genuuid
			err = db.CreateCollection(ctx, collectionName)
			if err != nil {
				logs.Error("Error creating collection: %v", err)
				return
			}
			// convert and insert the thread post
			oldThreadPost := oldPosts[0]
			threadPost := models.ThreadPost{
				ID:             primitive.NewObjectID(),
				BoardID:        oldThreadPost.BoardID,
				ThreadID:       genuuid,
				PostID:         genuuid,
				Content:        oldThreadPost.Content,
				PartialContent: oldThreadPost.PartialContent,
				Image:          insertImage(ctx, db, boardPath, boardID, oldThreadPost.ImageURL),
				Thumbnail:      insertThumbnail(ctx, db, boardPath, boardID, oldThreadPost.ImageURL),
				Subject:        oldThreadPost.Subject,
				Author:         oldThreadPost.Author,
				TrueUser:       oldThreadPost.TrueUser,
				Timestamp:      parseTimestamp(oldThreadPost.Timestamp),
				IP:             oldThreadPost.IP,
				Sticky:         oldThreadPost.Sticky,
				Locked:         oldThreadPost.Locked,
				PostCount:      countPosts(oldPosts),
				ReportCount:    0,
			}
			_, err = db.Collection(collectionName).InsertOne(ctx, threadPost)
			if err != nil {
				logs.Error("Error inserting thread post: %v", err)
				return
			}
			// convert and insert the replies
			for _, oldReply := range oldPosts[1:] {
				logs.Debug("timestamp: %v", oldReply.Timestamp)
				reply := models.Posts{
					ID:             primitive.NewObjectID(),
					BoardID:        oldReply.BoardID,
					ParentID:       genuuid,
					PostID:         oldReply.PostID,
					Content:        oldReply.Content,
					PartialContent: oldReply.PartialContent,
					Image:          insertImage(ctx, db, boardPath, boardID, oldReply.ImageURL),
					Thumbnail:      insertThumbnail(ctx, db, boardPath, boardID, oldReply.ImageURL),
					Subject:        oldReply.Subject,
					Author:         oldReply.Author,
					TrueUser:       oldReply.TrueUser,
					Timestamp:      parseTimestamp(oldReply.Timestamp),
					IP:             oldReply.IP,
					ReportCount:    0,
				}
				_, err := db.Collection(collectionName).InsertOne(ctx, reply)
				if err != nil {
					logs.Error("Error inserting reply: %v", err)
					return
				}
			}
		}
	}
}

func parseTimestamp(timestamp string) int64 {
	// Assuming the timestamp is in the format "09-18-2024 05:14:11"
	layout := "01-02-2006 15:04:05"
	t, err := time.Parse(layout, timestamp)
	if err != nil {
		logs.Error("Error parsing timestamp: %v", err)
		return 0
	}
	return t.Unix()
}

func countPosts(posts []OldPost) int {
	return len(posts)
}

func insertImage(ctx context.Context, db *mongo.Database, boardPath, boardid, imageURL string) string {
	imagePath := filepath.Join(boardPath, imageURL)
	imageID := primitive.NewObjectID()

	// Read the image file
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		logs.Error("Error reading image file: %v", err)
		return ""
	}

	imageDoc := bson.M{"_id": imageID,
		"image":    imageData,
		"filetype": http.DetectContentType(imageData),
		"size":     len(imageData),
		"height":   0,
		"width":    0,
	}
	_, err = db.Collection("images").InsertOne(ctx, imageDoc)
	if err != nil {
		logs.Error("Error inserting image: %v", err)
		return ""
	}
	logs.Info("Successfully inserted image with ID: %s", imageID.Hex())
	return imageID.Hex()
}

func insertThumbnail(ctx context.Context, db *mongo.Database, boardPath, boardid, imageURL string) string {
	thumbPath := filepath.Join(boardPath, imageURL)
	thumbID := primitive.NewObjectID()

	// Read the thumbnail file
	thumbData, err := os.ReadFile(thumbPath)
	if err != nil {
		logs.Error("Error reading thumbnail file: %v", err)
		return ""
	}

	thumbDoc := bson.M{"_id": thumbID, "image": thumbData, "filetype": http.DetectContentType(thumbData), "size": len(thumbData), "height": 0, "width": 0}
	_, err = db.Collection("thumbs").InsertOne(ctx, thumbDoc)
	if err != nil {
		logs.Error("Error inserting thumbnail: %v", err)
		return ""
	}
	logs.Info("Successfully inserted thumbnail with ID: %s", thumbID.Hex())
	return thumbID.Hex()
}
