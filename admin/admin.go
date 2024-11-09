package admin

import (
	"context"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/labstack/echo/v4"
)

// Board is a struct for a board
type Board struct {
	BoardID     string `gob:"id" gorm:"column:board_id"`
	Name        string `gob:"name"`
	Description string `gob:"description"`
	PostCount   int64  `gob:"post_count"`
	ImageOnly   bool   `gob:"image_only" gorm:"default:false"`
	Locked      bool   `gob:"locked" gorm:"default:false"`
	Archived    bool   `gob:"archived"	gorm:"default:false"`
	LatestPosts bool   `gob:"latest_posts" gorm:"default:false"`
}

type Post struct {
	BoardID   string `gob:"BoardID"`
	ThreadID  string `gob:"ThreadID"`
	PostID    string `gob:"PostID"`
	Content   string `gob:"Content"`
	ImageURL  string `gob:"ImageURL"`
	Subject   string `gob:"Subject"`
	Author    string `gob:"Author"`
	ParentID  string `gob:"ParentID"`
	Timestamp string `gob:"Timestamp"`
	IP        string `gob:"IP"`
	Sticky    bool   `gob:"Sticky"`
	Locked    bool   `gob:"Locked"`
}

type RecentPosts struct {
	BoardID   string `gob:"BoardID"`
	ThreadID  string `gob:"ThreadID"`
	PostID    string `gob:"PostID"`
	Content   string `gob:"Content"`
	ImageURL  string `gob:"ImageURL"`
	Subject   string `gob:"Subject"`
	Author    string `gob:"Author"`
	ParentID  string `gob:"ParentID"`
	Timestamp string `gob:"Timestamp"`
}

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
	var board Board
	db := database.DB_Main
	collection := db.Collection("boards")
	err := collection.FindOne(context.Background(), bson.M{"board_id": boardID}).Decode(&board)
	if err == nil {
		return c.JSON(http.StatusBadRequest, "Board already exists")
	}
	boardsclient := database.Client

	// create board
	board = Board{
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
