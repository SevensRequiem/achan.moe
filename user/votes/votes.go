package votes

import (
	"encoding/gob"
	"net/http"
	"os"

	"achan.moe/board"
	"github.com/labstack/echo/v4"
)

func VoteRoutes(e *echo.Echo) {
	e.POST("/vote/up/:b/:t", UpvoteThread)
	e.POST("/vote/down/:b/:t", DownvoteThread)
	e.POST("/vote/up/:b/:t/:p", UpvotePost)
	e.POST("/vote/down/:b/:t/:p", DownvotePost)
}

func UpvoteThread(c echo.Context) error {
	boardID := c.Param("b")
	threadID := c.Param("t")
	filePath := "boards/" + boardID + "/" + threadID + ".gob"

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Thread not found"})
	}
	defer file.Close()

	var posts []board.Post
	if err := gob.NewDecoder(file).Decode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode posts"})
	}

	if len(posts) > 0 {
		posts[0].Upvotes++ // Upvote the first post
	} else {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "No posts found in thread"})
	}

	file.Seek(0, 0) // Reset file pointer
	if err := gob.NewEncoder(file).Encode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode posts"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Upvoted first post in thread"})
}

func DownvoteThread(c echo.Context) error {
	boardID := c.Param("b")
	threadID := c.Param("t")
	filePath := "boards/" + boardID + "/" + threadID + ".gob"

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Thread not found"})
	}
	defer file.Close()

	var posts []board.Post
	if err := gob.NewDecoder(file).Decode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode posts"})
	}

	if len(posts) > 0 {
		posts[0].Downvotes++ // Downvote the first post
	} else {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "No posts found in thread"})
	}

	file.Seek(0, 0)
	if err := gob.NewEncoder(file).Encode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode posts"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Downvoted first post in thread"})
}

func UpvotePost(c echo.Context) error {
	boardID := c.Param("b")
	threadID := c.Param("t")
	postID := c.Param("p")
	filePath := "boards/" + boardID + "/" + threadID + ".gob"

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}
	defer file.Close()

	var posts []board.Post
	if err := gob.NewDecoder(file).Decode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode posts"})
	}

	for i := range posts {
		if posts[i].PostID == postID {
			posts[i].Upvotes++ // Upvote the specific post
			break
		}
	}

	file.Seek(0, 0)
	if err := gob.NewEncoder(file).Encode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode posts"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Upvoted post", "post": postID})
}

func DownvotePost(c echo.Context) error {
	boardID := c.Param("b")
	threadID := c.Param("t")
	postID := c.Param("p")
	filePath := "boards/" + boardID + "/" + threadID + ".gob"

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}
	defer file.Close()

	var posts []board.Post
	if err := gob.NewDecoder(file).Decode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode posts"})
	}

	for i := range posts {
		if posts[i].PostID == postID {
			posts[i].Downvotes++ // Downvote the specific post
			break
		}
	}

	file.Seek(0, 0)
	if err := gob.NewEncoder(file).Encode(&posts); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode posts"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Downvoted post", "post": postID})
}
