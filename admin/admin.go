package admin

import (
	"context"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"achan.moe/auth"
	"achan.moe/board"
	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/cache"
	"achan.moe/utils/queue"
	"github.com/labstack/echo/v4"
)

var client = database.Client

// CreateBoard creates a new board
func CreateBoard(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}

	// get board name
	name := c.FormValue("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, "Name cannot be empty")
	}
	// get board id
	boardID := c.FormValue("id")
	if boardID == "" {
		return c.JSON(http.StatusBadRequest, "ID cannot be empty")
	}
	// get board description
	description := c.FormValue("description")
	if description == "" {
		return c.JSON(http.StatusBadRequest, "Description cannot be empty")
	}
	recentposts := c.FormValue("recentposts")
	if recentposts == "" {
		return c.JSON(http.StatusBadRequest, "Recentposts cannot be empty")
	}
	imgonly := c.FormValue("imageonly") == "true"

	// check if board exists
	var board models.Board
	db := database.DB_Main
	collection := db.Collection("boards")
	err := collection.FindOne(context.Background(), bson.M{"board_id": boardID}).Decode(&board)
	if err == nil {
		return c.JSON(http.StatusBadRequest, "Board already exists")
	}
	boardsclient := database.Client

	// create board
	board = models.Board{
		BoardID:     boardID,
		Name:        name,
		Description: description,
		PostCount:   0,
		ImageOnly:   imgonly,
		Locked:      false,
		Archived:    false,
	}
	_, err = collection.InsertOne(context.Background(), board)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Error creating board")
	}

	if boardsclient == nil {
		return c.JSON(http.StatusInternalServerError, "Database client is not initialized")
	}

	boardsclient.Database(boardID)
	boardsclient.Database(boardID).CreateCollection(context.Background(), "images")
	boardsclient.Database(boardID).CreateCollection(context.Background(), "thumbs")
	boardsclient.Database(boardID).CreateCollection(context.Background(), "banners")
	return c.JSON(http.StatusOK, "Board created")
}

// DeleteBoard deletes a board
func DeleteBoard(c echo.Context) {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
		return
	}

	boardID := c.Param("id")
	if boardID == "" {
		c.JSON(http.StatusBadRequest, "ID cannot be empty")
		return
	}

	db := database.DB_Main
	collection := db.Collection("boards")
	_, err := collection.DeleteOne(context.Background(), bson.M{"board_id": boardID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Error deleting board")
		return
	}
	c.JSON(http.StatusOK, "Board deleted")
}

func DeleteThread(c echo.Context, boardID string, threadID string) {
	queue.Q.Enqueue("thread:delete", func() { processDeleteThread(c, boardID, threadID) })
}

func DeletePost(c echo.Context, boardID string, threadID string, postID string) {
	queue.Q.Enqueue("post:delete", func() { processDeletePost(c, boardID, threadID, postID) })
}

func processDeleteThread(c echo.Context, boardID string, threadID string) {
	db := client.Database(boardID)
	ctx := context.Background()
	var thread models.ThreadPost
	err := db.Collection(threadID).FindOne(ctx, bson.M{"thread_id": threadID}).Decode(&thread)
	if err != nil {
		logs.Error("Error decoding thread: %v", err)
		return
	}

	imageid := thread.Image
	thumbid := thread.Thumbnail
	imageObjectID, err := primitive.ObjectIDFromHex(imageid)
	if err != nil {
		logs.Error("Invalid image ID: %v", err)
		return
	}

	thumbObjectID, err := primitive.ObjectIDFromHex(thumbid)
	if err != nil {
		logs.Error("Invalid thumbnail ID: %v", err)
		return
	}

	_, err = db.Collection("images").DeleteOne(ctx, bson.M{"_id": imageObjectID})
	if err != nil {
		logs.Error("Error deleting image: %v", err)
	}

	_, err = db.Collection("thumbs").DeleteOne(ctx, bson.M{"_id": thumbObjectID})
	if err != nil {
		logs.Error("Error deleting thumbnail: %v", err)
	}

	_, err = db.Collection(threadID).DeleteMany(ctx, bson.M{})
	if err != nil {
		logs.Error("Error deleting thread: %v", err)
	}
	board.RemoveFromRecentPosts("", threadID, boardID)
	cache.DeleteThreadFromCache(boardID, threadID)
}

func processDeletePost(c echo.Context, boardID string, threadID string, postID string) {
	db := client.Database(boardID)
	ctx := context.Background()
	var post models.Posts
	err := db.Collection(threadID).FindOne(ctx, bson.M{"post_id": postID}).Decode(&post)
	if err != nil {
		logs.Error("Error decoding post: %v", err)
		return
	}

	imageid := post.Image
	thumbid := post.Thumbnail
	imageObjectID, err := primitive.ObjectIDFromHex(imageid)
	if err != nil {
		logs.Error("Invalid image ID: %v", err)
		return
	}

	thumbObjectID, err := primitive.ObjectIDFromHex(thumbid)
	if err != nil {
		logs.Error("Invalid thumbnail ID: %v", err)
		return
	}

	_, err = db.Collection("images").DeleteOne(ctx, bson.M{"_id": imageObjectID})
	if err != nil {
		logs.Error("Error deleting image: %v", err)
	}

	_, err = db.Collection("thumbs").DeleteOne(ctx, bson.M{"_id": thumbObjectID})
	if err != nil {
		logs.Error("Error deleting thumbnail: %v", err)
	}

	_, err = db.Collection(threadID).DeleteOne(ctx, bson.M{"post_id": postID})
	if err != nil {
		logs.Error("Error deleting post: %v", err)
	}
	cache.DeletePostFromCache(boardID, threadID, postID)
}
