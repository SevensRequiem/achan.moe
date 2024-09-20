package user

import (
	"net/http"
	"strings"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/home"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var user auth.User

type RecentReputation struct {
	PlusReputation  int
	MinusReputation int
	IP              string
	ID              string
}

func init() {
	db := database.DB
	db.AutoMigrate(&RecentReputation{})
}

func PlusReputation(c echo.Context) error {
	if !auth.AuthCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	db := database.DB
	userID := c.FormValue("id")
	ip := c.RealIP()

	var recentRep RecentReputation
	if db.Where("ip = ? AND id = ?", ip, userID).First(&recentRep).Error == nil {
		if recentRep.MinusReputation > 0 {
			// Change reputation from minus to plus
			recentRep.MinusReputation = 0
			recentRep.PlusReputation = 1
			user.MinusReputation--
			user.PlusReputation++
			if err := db.Save(&recentRep).Error; err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update recent reputation"})
			}
			if err := db.Save(&user).Error; err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user reputation"})
			}
			return c.JSON(http.StatusOK, map[string]string{"message": "Reputation changed to positive"})
		}
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "You have already given reputation to this user"})
	}

	// Find the user by ID
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	// Increment the user's reputation
	user.PlusReputation++

	// Save the recent reputation
	recentRep = RecentReputation{PlusReputation: 1, IP: ip, ID: userID}
	if err := db.Save(&recentRep).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recent reputation"})
	}

	// Update the user's reputation in the database
	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update reputation"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Reputation increased"})
}

func MinusReputation(c echo.Context) error {
	if !auth.AuthCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	db := database.DB
	userID := c.FormValue("id")
	ip := c.RealIP()

	var recentRep RecentReputation
	if db.Where("ip = ? AND id = ?", ip, userID).First(&recentRep).Error == nil {
		if recentRep.PlusReputation > 0 {
			// Change reputation from plus to minus
			recentRep.PlusReputation = 0
			recentRep.MinusReputation = 1
			user.PlusReputation--
			user.MinusReputation++
			if err := db.Save(&recentRep).Error; err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update recent reputation"})
			}
			if err := db.Save(&user).Error; err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user reputation"})
			}
			return c.JSON(http.StatusOK, map[string]string{"message": "Reputation changed to negative"})
		}
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "You have already given reputation to this user"})
	}

	// Find the user by ID
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	// Decrement the user's reputation
	user.MinusReputation++

	// Save the recent reputation
	recentRep = RecentReputation{MinusReputation: 1, IP: ip, ID: userID}
	if err := db.Save(&recentRep).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recent reputation"})
	}

	// Update the user's reputation in the database
	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update reputation"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Reputation decreased"})
}

func GetUser(c echo.Context) error {
	db := database.DB
	userID := c.Param("id")

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

	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	return c.JSON(http.StatusOK, map[string]int{"plus": user.PlusReputation, "minus": user.MinusReputation})
}

func ListUsers(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB
	var users []auth.User
	if err := db.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListUsersByReputation(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB
	var users []auth.User
	if err := db.Order("plus_reputation desc").Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users by reputation"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListUsersByJoinDate(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB
	var users []auth.User
	if err := db.Order("created_at desc").Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users by join date"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListUsersByLastLogin(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB
	var users []auth.User
	if err := db.Order("last_login desc").Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users by last login"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListAdmins(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB
	var users []auth.User
	if err := db.Where("admin = ?", true).Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list admins"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListModerators(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB
	var users []auth.User
	if err := db.Where("moderator = ?", true).Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list moderators"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListJannies(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB
	var users []auth.User
	if err := db.Where("jannie = ?", true).Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list jannies"})
	}

	return c.JSON(http.StatusOK, users)
}

func UpdateUserGroups(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	db := database.DB
	userID := c.FormValue("id")
	admin := c.FormValue("admin")
	moderator := c.FormValue("moderator")
	janny := c.FormValue("janny")
	jannyboards := c.FormValue("jannyboards")

	var user auth.User

	// Find the user by ID
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	// Update admin status
	if admin == "on" {
		user.Groups.Admin = true
	} else {
		user.Groups.Admin = false
	}

	// Update moderator status
	if moderator == "on" {
		user.Groups.Moderator = true
	} else {
		user.Groups.Moderator = false
	}

	// Update janny status and jannyboards
	if janny == "on" {
		user.Groups.Janny.Boards = strings.Split(jannyboards, ",")
	} else {
		user.Groups.Janny.Boards = []string{}
	}

	// Save the updated user to the database
	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update groups"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Groups updated"})
}

func Routes(e *echo.Echo) {
	e.GET("/profile", func(c echo.Context) error {
		return home.ProfileHandler(c)
	})

	e.POST("/profile/edit", func(c echo.Context) error {
		return auth.UpdateUser(c)
	})

	e.POST("/profile/delete", func(c echo.Context) error {
		return auth.DeleteUser(c)
	})

	e.GET("/user/:id", GetUser)
	e.GET("/user/:id/reputation", GetUserReputation)
	e.POST("/user/:id/plusreputation", PlusReputation)
	e.POST("/user/:id/minusreputation", MinusReputation)
	e.GET("/api/admin/users", ListUsers)
	e.GET("/api/admin/users/reputation", ListUsersByReputation)
	e.GET("/api/admin/users/joindate", ListUsersByJoinDate)
	e.GET("/api/admin/users/lastlogin", ListUsersByLastLogin)
	e.GET("/api/admin/admins", ListAdmins)
	e.GET("/api/admin/moderators", ListModerators)
	e.GET("/api/admin/jannies", ListJannies)
	e.POST("/admin/groups/edit", UpdateUserGroups)
}
