package board

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/utils/queue"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/nfnt/resize"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/rand"
)

type Board struct {
	BoardID     string `bson:"boardid"`
	Name        string `bson:"name"`
	Description string `bson:"description"`
	PostCount   int64  `bson:"post_count"`
	ImageOnly   bool   `bson:"image_only"`
	Locked      bool   `bson:"locked"`
	Archived    bool   `bson:"archived"`
	LatestPosts bool   `bson:"latest_posts"`
	Pages       int    `bson:"pages"`
}

type ThreadPost struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	BoardID        string             `bson:"boardid"`
	ThreadID       string             `bson:"thread_id"`
	PostID         string             `bson:"post_id"`
	Content        string             `bson:"content"`
	PartialContent string             `bson:"partial_content"`
	Image          string             `bson:"image"`
	Thumbnail      string             `bson:"thumb"`
	Subject        string             `bson:"subject"`
	Author         string             `bson:"author"`
	TrueUser       string             `bson:"true_user"`
	Timestamp      int64              `bson:"timestamp"`
	IP             string             `bson:"ip"`
	Sticky         bool               `bson:"sticky"`
	Locked         bool               `bson:"locked"`
	PostCount      int                `bson:"post_count"`
	ReportCount    int                `bson:"report_count"`
}

type Post struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	BoardID        string             `bson:"boardid"`
	ParentID       string             `bson:"parent_id"`
	PostID         string             `bson:"post_id"`
	Content        string             `bson:"content"`
	PartialContent string             `bson:"partial_content"`
	Image          string             `bson:"image"`
	Thumbnail      string             `bson:"thumb"`
	Subject        string             `bson:"subject"`
	Author         string             `bson:"author"`
	TrueUser       string             `bson:"true_user"`
	Timestamp      int64              `bson:"timestamp"`
	IP             string             `bson:"ip"`
	ReportCount    int                `bson:"report_count"`
}

type RecentPosts struct {
	ID             int64  `bson:"_id,omitempty"`
	BoardID        string `bson:"boardid"`
	ThreadID       string `bson:"thread_id"`
	PostID         string `bson:"post_id"`
	Content        string `bson:"content"`
	PartialContent string `bson:"partial_content"`
	Image          string `bson:"image_url"`
	Thumbnail      string `bson:"thumb_url"`
	Subject        string `bson:"subject"`
	Author         string `bson:"author"`
	TrueUser       string `bson:"true_user"`
	ParentID       string `bson:"parent_id"`
	Timestamp      int64  `bson:"timestamp"`
}

type Image struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Image    []byte             `bson:"image"`
	Filetype string             `bson:"filetype"`
	Size     int64              `bson:"size"`
	Height   int                `bson:"height"`
	Width    int                `bson:"width"`
}

type Recents struct {
	ID     int64  `bson:"_id,omitempty"`
	PostID string `bson:"post_id"`
}

type PostCounter struct {
	ID        int   `bson:"_id,omitempty"`
	PostCount int64 `bson:"post_count"`
}

var User = auth.User{}
var manager = queue.NewQueueManager()
var q = manager.GetQueue("thread", 10)
var client = database.Client

func init() {
	manager.ProcessQueuesWithPrefix("thread")
}

func saveThumb(boardID string, imageFile *multipart.FileHeader) (string, error) {
	db := client.Database(boardID)
	file, err := imageFile.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		logs.Error("Error decoding image: %v", err)
		return "", err
	}

	thumb := resize.Thumbnail(250, 250, img, resize.Lanczos3)
	var thumbBuffer bytes.Buffer
	err = png.Encode(&thumbBuffer, thumb)
	if err != nil {
		logs.Error("Error encoding thumbnail: %v", err)
		return "", err
	}

	imageDoc := Image{
		Image:    thumbBuffer.Bytes(),
		Filetype: "image/png",
		Size:     int64(thumbBuffer.Len()),
		Height:   thumb.Bounds().Dy(),
		Width:    thumb.Bounds().Dx(),
	}

	result, err := db.Collection("thumbs").InsertOne(context.Background(), imageDoc)
	if err != nil {
		logs.Error("Error inserting thumbnail: %v", err)
		return "", err
	}

	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func saveImage(boardID string, imageFile *multipart.FileHeader) (string, error) {
	db := client.Database(boardID)
	file, err := imageFile.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the file into a byte slice
	imageData, err := io.ReadAll(file)
	if err != nil {
		logs.Error("Error reading image file: %v", err)
		return "", err
	}

	imageDoc := Image{
		Image:    imageData,
		Filetype: imageFile.Header.Get("Content-Type"),
		Size:     imageFile.Size,
		// Height and Width can be set if you decode the image
	}

	result, err := db.Collection("images").InsertOne(context.Background(), imageDoc)
	if err != nil {
		logs.Error("Error inserting image: %v", err)
		return "", err
	}

	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func ReturnThumb(c echo.Context, boardID string, thumbID string) error {
	db := client.Database(boardID)
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(thumbID)
	if err != nil {
		logs.Error("Invalid thumbnail ID '%s': %v", thumbID, err)
		return c.String(http.StatusBadRequest, "Invalid thumbnail ID")
	}

	var image Image
	err = db.Collection("thumbs").FindOne(ctx, bson.M{"_id": objectID}).Decode(&image)
	if err != nil {
		logs.Error("Error finding thumbnail: %v", err)
		return c.String(http.StatusNotFound, "Thumbnail not found")
	}

	c.Response().Header().Set("Content-Type", image.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(image.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Write(image.Image)

	return nil
}

func ReturnImage(c echo.Context, boardID string, imageID string) error {
	db := client.Database(boardID)
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(imageID)
	if err != nil {
		logs.Error("Invalid image ID '%s': %v", imageID, err)
		return c.String(http.StatusBadRequest, "Invalid image ID")
	}

	var image Image
	err = db.Collection("images").FindOne(ctx, bson.M{"_id": objectID}).Decode(&image)
	if err != nil {
		logs.Error("Error finding image: %v", err)
		return c.String(http.StatusNotFound, "Image not found")
	}

	c.Response().Header().Set("Content-Type", image.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(image.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Write(image.Image)

	return nil
}
func addToRecents(postID string) {
	db := database.DB_Main
	db.Collection("recents").InsertOne(context.Background(), bson.M{"post_id": postID})
}

func checkReplyID(replyID string) bool {
	db := database.DB_Main
	var post Post
	db.Collection("posts").FindOne(context.Background(), bson.M{"post_id": replyID}).Decode(&post)
	return post.ParentID == replyID
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

func GenUUID(boardid string) string {
	if boardid == "" {
		logs.Fatal("Error: boardid cannot be empty")
	}

	db := database.Client.Database(boardid)
	for {
		b := make([]byte, 4)
		_, err := rand.Read(b)
		if err != nil {
			logs.Fatal("Error generating UUID: %v", err)
		}
		id := hex.EncodeToString(b)

		// Check for UUID collision
		var result bson.M
		err = db.Collection("posts").FindOne(context.Background(), bson.M{"post_id": id}).Decode(&result)
		if err == mongo.ErrNoDocuments {
			// No collision, return the generated UUID
			return id
		} else if err != nil {
			logs.Error("Error checking UUID: %v", err)
		} else {
			logs.Debug("UUID collision, retrying")
		}
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
	db := database.DB_Boards.Collection(boardID)
	var threadpost ThreadPost
	db.FindOne(context.Background(), bson.M{"boardid": boardID, "thread_id": threadID, "locked": true}).Decode(&threadpost)
	return threadpost.Locked
}

func CheckLatestPosts(boardID string) bool {
	db := database.DB_Main.Collection("recent_posts")
	var board Board
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
	var posts []Post
	for cursor.Next(ctx) {
		var post Post
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
	db := client.Database(boardID)
	ctx := context.Background()
	cursor, err := db.Collection("threads").Find(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding threads: %v", err)
		return
	}
	defer cursor.Close(ctx)
	var threads []ThreadPost
	for cursor.Next(ctx) {
		var thread ThreadPost
		if err := cursor.Decode(&thread); err != nil {
			logs.Error("Error decoding thread: %v", err)
			return
		}
		threads = append(threads, thread)
	}
	if len(threads) == 0 {
		return
	}
	latestThread := threads[len(threads)-1]
	DeleteThread(c, boardID, latestThread.ThreadID)
}
func PurgeBoard(boardID string) {
	db := client.Database(boardID)
	ctx := context.Background()
	db.Drop(ctx)
}
func GetBoards() []Board {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	cursor, err := db.Find(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding boards: %v", err)
		return nil
	}
	defer cursor.Close(ctx)
	var boards []Board
	for cursor.Next(ctx) {
		var board Board
		if err := cursor.Decode(&board); err != nil {
			logs.Error("Error decoding board: %v", err)
			return nil
		}
		boards = append(boards, board)
	}
	return boards
}

func GetLatestPosts(n int) ([]RecentPosts, error) {
	db := database.DB_Main.Collection("recent_posts")
	ctx := context.Background()
	cursor, err := db.Find(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding recent posts: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)
	var recentPosts []RecentPosts
	for cursor.Next(ctx) {
		var post RecentPosts
		if err := cursor.Decode(&post); err != nil {
			logs.Error("Error decoding post: %v", err)
			return nil, err
		}
		recentPosts = append(recentPosts, post)
	}
	return recentPosts, nil
}

func GetBoardName(boardID string) string {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	var board Board
	db.FindOne(ctx, bson.M{"boardid": boardID}).Decode(&board)
	return board.Name

}

func GetBoard(boardID string) Board {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	var board Board
	db.FindOne(ctx, bson.M{"boardid": boardID}).Decode(&board)
	return board

}

func GetBoardID(boardID string) string {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	var board Board
	db.FindOne(ctx, bson.M{"boardid": boardID}).Decode(&board)
	fmt.Println(board.BoardID)
	return board.BoardID
}

func GetThreads(boardID string) []ThreadPost {
	db := client.Database(boardID)
	ctx := context.Background()
	cursor, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding collections: %v", err)
		return nil
	}
	defer cursor.Close(ctx)
	var threads []struct {
		ThreadPost        ThreadPost
		LastPostTimestamp int64
	}
	for cursor.Next(ctx) {
		var collection struct {
			Name string `bson:"name"`
		}
		if err := cursor.Decode(&collection); err != nil {
			logs.Error("Error decoding collection: %v", err)
			return nil
		}
		if collection.Name == "thumbs" || collection.Name == "images" {
			continue
		}
		var threadPost ThreadPost
		err := db.Collection(collection.Name).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", 1}})).Decode(&threadPost)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			logs.Error("Error finding first document in collection %s: %v", collection.Name, err)
			return nil
		}
		var lastPost ThreadPost
		err = db.Collection(collection.Name).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", -1}})).Decode(&lastPost)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			logs.Error("Error finding last document in collection %s: %v", collection.Name, err)
			return nil
		}
		threads = append(threads, struct {
			ThreadPost        ThreadPost
			LastPostTimestamp int64
		}{
			ThreadPost:        threadPost,
			LastPostTimestamp: lastPost.Timestamp,
		})
	}

	sort.SliceStable(threads, func(i, j int) bool {
		if threads[i].ThreadPost.Sticky != threads[j].ThreadPost.Sticky {
			return threads[i].ThreadPost.Sticky
		}
		return threads[i].LastPostTimestamp > threads[j].LastPostTimestamp
	})

	var sortedThreads []ThreadPost
	for _, thread := range threads {
		sortedThreads = append(sortedThreads, thread.ThreadPost)
	}

	return sortedThreads
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

func GetThread(boardID string, threadID string) ThreadPost {
	db := client.Database(boardID)
	ctx := context.Background()
	var threadpost ThreadPost
	err := db.Collection(threadID).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", 1}})).Decode(&threadpost)
	if err != nil {
		logs.Error("Error finding first document in collection %s: %v", threadID, err)
	}
	return threadpost
}

func GetThreadPosts(boardID string, threadID string) []Post {
	db := client.Database(boardID)
	ctx := context.Background()

	// Find all documents except the earliest one
	cursor, err := db.Collection(threadID).Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{"timestamp", 1}}).SetSkip(1))
	if err != nil {
		logs.Error("Error finding posts: %v", err)
		return nil
	}
	defer cursor.Close(ctx)

	var posts []Post
	for cursor.Next(ctx) {
		var post Post
		if err := cursor.Decode(&post); err != nil {
			logs.Error("Error decoding post: %v", err)
			return nil
		}
		posts = append(posts, post)
	}

	// Sort the posts by timestamp in ascending order
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].Timestamp < posts[j].Timestamp
	})

	return posts
}
func AddRecentPost(post Post, threadpost ThreadPost) {
	db := database.DB_Main.Collection("recent_posts")
	ctx := context.Background()

	if post != (Post{}) {
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
	var threadpost ThreadPost
	db.Collection(threadid).FindOne(ctx, bson.M{"thread_id": threadid}).Decode(&threadpost)
	return threadpost.Locked
}

func AddGlobalPostCount() int64 {
	db := database.DB_Main.Collection("data")
	var counter PostCounter
	db.FindOne(context.Background(), bson.M{}).Decode(&counter)
	_, err := db.UpdateOne(context.Background(), bson.M{}, bson.M{"$inc": bson.M{"post_count": 1}})
	if err != nil {
		logs.Error("Error incrementing post count: %v", err)
	}
	return counter.PostCount
}
func GetGlobalPostCount() int64 {
	db := database.DB_Main.Collection("data")
	var counter PostCounter
	db.FindOne(context.Background(), bson.M{}).Decode(&counter)
	return counter.PostCount

}

func AddBoardPostCount(boardID string) {
	db := database.DB_Boards.Collection(boardID)
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
	db := database.DB_Main.Collection("data")
	var counter PostCounter
	db.FindOne(context.Background(), bson.M{}).Decode(&counter)
	return counter.PostCount
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

func DeleteThread(c echo.Context, boardID string, threadID string) {
	db := client.Database(boardID)
	ctx := context.Background()
	_, err := db.Collection(threadID).DeleteMany(ctx, bson.M{})
	if err != nil {
		logs.Error("Error deleting thread: %v", err)
		return
	}
	RemoveFromRecentPosts("", threadID, boardID)

}
func DeletePost(c echo.Context, boardID string, threadID string, postID string) {
	db := client.Database(boardID)
	ctx := context.Background()
	_, err := db.Collection(threadID).DeleteOne(ctx, bson.M{"post_id": postID})
	if err != nil {
		logs.Error("Error deleting post: %v", err)
		return
	}
	RemoveFromRecentPosts(postID, threadID, boardID)

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
	_, err := db.Collection(threadID).UpdateOne(ctx, bson.M{"thread_id": threadID}, bson.M{"$inc": bson.M{"post_count": 1}})
	if err != nil {
		logs.Error("Error incrementing post count: %v", err)
	}
	return
}

func sanitize(content string, subject string, author string) (string, string, string) {
	content = template.HTMLEscapeString(content)
	subject = template.HTMLEscapeString(subject)
	author = template.HTMLEscapeString(author)
	return content, subject, author
}

func CreateThread(c echo.Context) error {
	boardID := c.Param("b")
	locked, err := CheckIfLocked(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking if board is locked")
	}
	if locked {
		return c.String(http.StatusForbidden, "Board is locked")
	}
	archived, err := CheckIfArchived(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking if board is archived")
	}
	if archived {
		return c.String(http.StatusForbidden, "Board is archived")
	}

	imageOnly, err := CheckIfImageOnly(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking if board is image only")
	}
	if imageOnly {
		return c.String(http.StatusForbidden, "Board is image only")
	}
	if LatestPostsCheck(c, boardID) {
		DeleteLastThread(c, boardID)
	}
	if GetTotalPostCount() >= 5000000 {
		return c.String(http.StatusForbidden, "Post limit reached")
	}
	boardPostCount, err := GetBoardPostCount(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error getting board post count")
	}
	if boardPostCount >= 500000 {
		return c.String(http.StatusForbidden, "Post limit reached")
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
		thumb, err := saveThumb(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving thumbnail")
		}
		Thumbnail = thumb
		image, err := saveImage(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving image")
		}
		Image = image
	}

	partialContent := content
	if len(content) > 100 {
		partialContent = content[:100]
	}

	threadpost := ThreadPost{
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
	}

	q.Enqueue(func() {
		processThreadPost(threadpost, c)
	})

	return c.JSON(http.StatusOK, "Thread created")
}

func processThreadPost(threadpost ThreadPost, c echo.Context) {
	db := client.Database(threadpost.BoardID)
	ctx := context.Background()
	db.CreateCollection(ctx, threadpost.ThreadID)
	_, err := db.Collection(threadpost.ThreadID).InsertOne(ctx, threadpost)
	if err != nil {
		logs.Error("Error inserting post: %v", err)
		return
	}
	AddGlobalPostCount()
	AddBoardPostCount(threadpost.BoardID)

	AddThreadPostCount(threadpost.BoardID, threadpost.ThreadID)
	AddRecentPost(Post{}, threadpost)
	SetSessionSelfPostID(c, threadpost.PostID)
	c.Redirect(http.StatusFound, "/board/"+threadpost.BoardID+"/thread/"+threadpost.ThreadID)
}

func CreatePost(c echo.Context) error {
	boardID := c.Param("b")
	if boardID == "" {
		return c.String(http.StatusBadRequest, "Board ID cannot be empty")
	}
	locked, err := CheckIfLocked(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking if board is locked")
	}
	if locked {
		return c.String(http.StatusForbidden, "Board is locked")
	}
	archived, err := CheckIfArchived(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking if board is archived")
	}
	if archived {
		return c.String(http.StatusForbidden, "Board is archived")
	}

	imageOnly, err := CheckIfImageOnly(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking if board is image only")
	}
	if imageOnly {
		return c.String(http.StatusForbidden, "Board is image only")
	}
	if LatestPostsCheck(c, boardID) {
		DeleteLastThread(c, boardID)
	}
	if GetTotalPostCount() >= 5000000 {
		return c.String(http.StatusForbidden, "Post limit reached")
	}
	boardPostCount, err := GetBoardPostCount(boardID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error getting board post count")
	}
	if boardPostCount >= 500000 {
		return c.String(http.StatusForbidden, "Post limit reached")
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
		thumb, err := saveThumb(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving thumbnail")
		}
		Thumbnail = thumb
		image, err := saveImage(boardID, imageFile)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error saving image")
		}
		Image = image
	}

	partialContent := content
	if len(content) > 100 {
		partialContent = content[:100]
	}

	post := Post{
		BoardID:        boardID,
		ParentID:       threadID,
		PostID:         postID,
		Content:        content,
		PartialContent: partialContent,
		Image:          Image,
		Thumbnail:      Thumbnail,
		Subject:        subject,
		Timestamp:      timestamp,
	}

	q.Enqueue(func() {
		processPost(post, c)
	})

	return c.JSON(http.StatusOK, "Post created")
}

func processPost(post Post, c echo.Context) {
	db := client.Database(post.BoardID)
	ctx := context.Background()
	_, err := db.Collection(post.ParentID).InsertOne(ctx, post)
	if err != nil {
		logs.Error("Error inserting post: %v", err)
		return
	}
	AddGlobalPostCount()
	AddBoardPostCount(post.BoardID)
	AddThreadPostCount(post.BoardID, post.ParentID)
	AddRecentPost(post, ThreadPost{})
	SetSessionSelfPostID(c, post.PostID)
}
