package admin

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Board is a struct for a board
type Board struct {
	BoardID     string `json:"id" gorm:"column:board_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PostCount   int64  `json:"post_count"`
	ImageOnly   bool   `json:"image_only" gorm:"default:false"`
	Locked      bool   `json:"locked" gorm:"default:false"`
	Archived    bool   `json:"archived"	gorm:"default:false"`
	LatestPosts bool   `json:"latest_posts" gorm:"default:false"`
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
	BoardID   string `json:"BoardID"`
	ThreadID  string `json:"ThreadID"`
	PostID    string `json:"PostID"`
	Content   string `json:"Content"`
	ImageURL  string `json:"ImageURL"`
	Subject   string `json:"Subject"`
	Author    string `json:"Author"`
	ParentID  string `json:"ParentID"`
	Timestamp string `json:"Timestamp"`
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
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
}

// DeleteBoard deletes a board
func DeleteBoard(c echo.Context) {
	if !auth.AdminCheck(c) {
		c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	boardID := c.Param("b")
	if boardID == "" {
		c.JSON(http.StatusBadRequest, "ID cannot be empty")
	}
	db := database.DB
	db.Delete(&Board{}, "board_id = ?", boardID)

	os.RemoveAll("boards/" + boardID)
}

func DeleteThread(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}

	threadID := c.Param("t")
	board := c.Param("b")

	// Convert threadID from string to int64
	threadIDInt, err := strconv.ParseInt(threadID, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid thread ID")
	}
	// Delete RecentPosts entries
	RemoveFromRecentPosts(0, threadIDInt)

	// Construct the path to the thread's JSON file
	threadFilePath := filepath.Join("boards", board, threadID+".json")

	// Open and decode the thread's JSON file
	jsonFile, err := os.Open(threadFilePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	defer jsonFile.Close()

	var posts []Post
	if err := json.NewDecoder(jsonFile).Decode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error decoding JSON")
	}

	// Delete images associated with the thread
	for _, post := range posts {
		if post.ImageURL != "" {
			imagePath := filepath.Join("boards", board, post.ImageURL)
			if err := os.Remove(imagePath); err != nil {
				return c.JSON(http.StatusInternalServerError, "Failed to delete image")
			}
		}
	}

	// Delete the thread's JSON file
	if err := os.Remove(threadFilePath); err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to delete thread")
	}

	return c.JSON(http.StatusOK, "Thread deleted")
}

func DeletePost(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	postid := c.Param("p")
	threadid := c.Param("t")
	board := c.Param("b")

	// Construct the file path
	filePath := "boards/" + board + "/" + threadid + ".json"
	// Open the JSON file
	jsonFile, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open file: %s, error: %v", filePath, err)
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	defer jsonFile.Close()

	// Decode the JSON file into posts
	var posts []Post
	if err := json.NewDecoder(jsonFile).Decode(&posts); err != nil {
		log.Printf("Error decoding JSON from file: %s, error: %v", filePath, err)
		return c.JSON(http.StatusInternalServerError, "Error decoding JSON")
	}

	// Find and delete the post
	// Convert postid from string to int64
	postidInt, err := strconv.ParseInt(postid, 10, 64)
	if err != nil {
		log.Printf("Error converting postid to int64: %v", err)
		return c.JSON(http.StatusBadRequest, "Invalid post ID")
	}

	// Find and delete the post
	for i, post := range posts {
		if post.PostID == postidInt {
			posts = append(posts[:i], posts[i+1:]...)
			break
		}
	}

	// delete image if exists
	var imageURL string
	for _, post := range posts {
		if strconv.FormatInt(post.PostID, 10) == postid {
			imageURL = post.ImageURL
			break
		}
	}
	if imageURL != "" {
		// skip if null
		if err := os.Remove("boards/" + board + "/" + imageURL); err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to delete image")
		}
	}

	// Database operations (omitted for brevity)
	RemoveFromRecentPosts(postidInt, 0)
	// Recreate the JSON file to update it
	jsonFile, err = os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file: %s, error: %v", filePath, err)
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	defer jsonFile.Close()

	// Encode the updated posts back into the JSON file
	if err := json.NewEncoder(jsonFile).Encode(posts); err != nil {
		log.Printf("Error encoding JSON to file: %s, error: %v", filePath, err)
		return c.JSON(http.StatusInternalServerError, "Error encoding JSON")
	}

	// Return success message
	return c.JSON(http.StatusOK, "Post deleted")
}
func JannyDeleteThread(c echo.Context) error {
	if !auth.JannyCheck(c, c.Param("b")) {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	threadid := c.Param("t")
	//convert to int64
	threadidInt, err := strconv.ParseInt(threadid, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid thread ID")
	}
	board := c.Param("b")
	allowedboard := db.Where("janny_boards = ?", board).First(&auth.User{})
	if allowedboard.Error != nil {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	RemoveFromRecentPosts(0, threadidInt)
	//read the json file and fetch the image url
	jsonFile, err := os.Open("boards/" + board + "/" + threadid + ".json")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	// delete image
	defer jsonFile.Close()
	var posts []Post
	if err := json.NewDecoder(jsonFile).Decode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error decoding JSON")
	}
	// get image url from post
	var imageURL string
	for _, post := range posts {
		imageURL = post.ImageURL
		break
	}
	if imageURL != "" {
		// skip if null
		if err := os.Remove("boards/" + board + "/" + imageURL); err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to delete image")
		}
	}
	if err := os.Remove("boards/" + board + "/" + threadid + ".json"); err != nil {
		// Handle the error, for example, log it and return an appropriate error message to the client
		return c.JSON(http.StatusInternalServerError, "Failed to delete thread")
	}

	return c.JSON(http.StatusOK, "Thread deleted")
}

func JannyDeletePost(c echo.Context) error {
	if !auth.JannyCheck(c, c.Param("b")) {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}
	postid := c.Param("p")
	threadid := c.Param("t")
	board := c.Param("b")

	allowedboard := db.Where("janny_boards = ?", board).First(&auth.User{})
	if allowedboard.Error != nil {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}

	// Open the JSON file
	jsonFile, err := os.Open("boards/" + board + "/" + threadid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	defer jsonFile.Close()

	// Decode the JSON file into posts
	var posts []Post
	if err := json.NewDecoder(jsonFile).Decode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error decoding JSON")
	}
	// get image url from post
	var imageURL string
	for _, post := range posts {
		if strconv.FormatInt(post.PostID, 10) == postid {
			imageURL = post.ImageURL
			break
		}
	}
	// delete image
	if imageURL != "" {
		// skip if null
		if err := os.Remove("boards/" + board + "/" + imageURL); err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to delete image")
		}
	}

	// Find and delete the post
	// Convert postid from string to int64
	postidInt, err := strconv.ParseInt(postid, 10, 64)
	if err != nil {
		log.Printf("Error converting postid to int64: %v", err)
		return c.JSON(http.StatusBadRequest, "Invalid post ID")
	}

	// Find and delete the post
	for i, post := range posts {
		if post.PostID == postidInt {
			posts = append(posts[:i], posts[i+1:]...)
			break
		}
	}
	RemoveFromRecentPosts(postidInt, 0)
	// Recreate the JSON file to update it
	jsonFile, err = os.Create("boards/" + board + "/" + threadid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Internal server error")
	}
	defer jsonFile.Close()

	// Encode the updated posts back into the JSON file
	if err := json.NewEncoder(jsonFile).Encode(posts); err != nil {
		return c.JSON(http.StatusInternalServerError, "Error encoding JSON")
	}

	// Return success message
	return c.JSON(http.StatusOK, "Post deleted")
}

func RemoveFromRecentPosts(postID, threadID int64) {
	// Check if both postID and threadID are zero, return early if true
	if postID == 0 && threadID == 0 {
		return
	}

	// Open database connection
	db := database.DB

	// Remove recent post by postID if it's not zero
	if postID != 0 {
		removeRecentPostByField(db, "post_id", postID)
	}

	// Remove recent post by threadID if it's not zero and different from postID
	if threadID != 0 {
		removeRecentPostByField(db, "thread_id", threadID)
	}

}

// Helper function to remove a recent post by a specific field (e.g., post_id or thread_id)
func removeRecentPostByField(db *gorm.DB, field string, value int64) {
	var recentPost RecentPosts
	db.Where(field+" = ?", value).First(&recentPost)
	db.Where(field+" = ?", value).Delete(&recentPost)
}

// UpdateUserRole updates a user's role
func UpdateUserRole(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}

	// Get the user's ID
	userID := c.FormValue("userid")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, "ID cannot be empty")
	}

	// Get the new role
	newRole := c.FormValue("group")
	if newRole == "" {
		return c.JSON(http.StatusBadRequest, "group cannot be empty")
	}

	if c.FormValue("boardid") == "" {
		// Update the user's role for all boards
		if err := db.Exec("UPDATE users SET groups = ? WHERE id = ?", newRole, userID).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to update user role")
		}
	} else {
		// Update the user's role for a specific board
		boardID := c.FormValue("boardid")
		if err := db.Exec("UPDATE users SET groups = ?, janny_boards = ? WHERE id = ?", newRole, boardID, userID).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to update user role")
		}
	}

	return c.JSON(http.StatusOK, "User role updated")
}
