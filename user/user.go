package user

import (
	"net/http"
	"strconv"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var user auth.User

func PlusReputation(c echo.Context) error {
	db := database.DB
	userID := c.FormValue("id")

	// Find the user by ID
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	// Increment the user's reputation
	user.Reputation++

	// Update the user's reputation in the database
	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update reputation"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Reputation increased"})
}

func MinusReputation(c echo.Context) error {
	db := database.DB
	userID := c.FormValue("id")

	// Find the user by ID
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	// Decrement the user's reputation
	user.Reputation--

	// Update the user's reputation in the database
	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update reputation"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Reputation decreased"})
}

func GetUser(c echo.Context) error {
	db := database.DB
	userID := c.Param("id")

	// Find the user by ID
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	return c.JSON(http.StatusOK, user)
}

func GetUserReputation(c echo.Context) error {
	db := database.DB
	userID := c.Param("id")

	// Find the user by ID
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"reputation": strconv.Itoa(user.Reputation)})
}
