package admin

import (
	"errors"
	"net/http"
	"os"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
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

var db = database.DB

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

	imgonly := c.FormValue("imageonly")
	if imgonly == "" {
		return c.JSON(http.StatusBadRequest, "Imageonly cannot be empty")
	}

	// check if board exists
	var board Board
	db := database.DB
	result := db.Where("board_id = ?", boardID).First(&board)
	if result.Error == nil {
		// Record is found, throw an error
		logs.Error("Record found, operation not allowed in CreateBoard")
		return c.JSON(http.StatusBadRequest, "Record found, operation not allowed")
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Record not found, proceed with your operation
		if err := database.DB.Exec("INSERT INTO boards (board_id, name, description, latest_posts, image_only) VALUES (?, ?, ?, ?, ?)", boardID, name, description, recentposts, imgonly).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to insert record")
		}
		os.Mkdir("boards/"+boardID, 0755)            // Consider error handling for directory creation
		os.Mkdir("boards/"+boardID+"/banners", 0755) // Consider error handling for directory creation
		return c.JSON(http.StatusOK, board)          // Respond with the created board
	} else {
		// Some other error occurred during the query execution
		logs.Error("Error occurred during query execution in CreateBoard")
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
}

// DeleteBoard deletes a board
func DeleteBoard(c echo.Context) {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
		logs.Error("Unauthorized DeleteBoard request")
	}
	boardID := c.Param("b")
	if boardID == "" {
		c.JSON(http.StatusBadRequest, "ID cannot be empty")
	}
	db := database.DB
	db.Delete(&Board{}, "board_id = ?", boardID)

	os.RemoveAll("boards/" + boardID)
}
