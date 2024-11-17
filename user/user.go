package user

import (
	"context"
	"net/http"
	"strings"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/home"
	"achan.moe/models"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var user models.User

type RecentReputation struct {
	PlusReputation  int
	MinusReputation int
	IP              string
	ID              string
}

func PlusReputation(c echo.Context) error {
	db := database.DB_Users
	userID := c.FormValue("id")
	ip := c.RealIP()

	var recentRep RecentReputation
	collection := db.Collection("recent_reputations")
	err := collection.FindOne(context.Background(), bson.M{"ip": ip, "id": userID}).Decode(&recentRep)
	if err == nil {
		if recentRep.MinusReputation > 0 {
			// Change reputation from minus to plus
			recentRep.MinusReputation = 0
			recentRep.PlusReputation = 1
			user.MinusReputation--
			user.PlusReputation++
			if _, err := collection.ReplaceOne(context.Background(), bson.M{"ip": ip, "id": userID}, recentRep); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update recent reputation"})
			}
			if _, err := db.Collection("users").ReplaceOne(context.Background(), bson.M{"uuid": userID}, user); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user reputation"})
			}
			return c.JSON(http.StatusOK, map[string]string{"message": "Reputation changed to positive"})
		}
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "You have already given reputation to this user"})
	}

	// Find the user by ID
	if err := db.Collection("users").FindOne(context.Background(), bson.M{"uuid": userID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	// Increment the user's reputation
	user.PlusReputation++

	// Save the recent reputation
	recentRep = RecentReputation{PlusReputation: 1, IP: ip, ID: userID}
	if _, err := collection.InsertOne(context.Background(), recentRep); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recent reputation"})
	}

	// Update the user's reputation in the database
	if _, err := db.Collection("users").ReplaceOne(context.Background(), bson.M{"uuid": userID}, user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update reputation"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Reputation increased"})
}

func MinusReputation(c echo.Context) error {
	db := database.DB_Users
	userID := c.FormValue("id")
	ip := c.RealIP()

	var recentRep RecentReputation
	collection := db.Collection("recent_reputations")
	err := collection.FindOne(context.Background(), bson.M{"ip": ip, "id": userID}).Decode(&recentRep)
	if err == nil {
		if recentRep.PlusReputation > 0 {
			// Change reputation from plus to minus
			recentRep.PlusReputation = 0
			recentRep.MinusReputation = 1
			user.PlusReputation--
			user.MinusReputation++
			if _, err := collection.ReplaceOne(context.Background(), bson.M{"ip": ip, "id": userID}, recentRep); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update recent reputation"})
			}
			if _, err := db.Collection("users").ReplaceOne(context.Background(), bson.M{"uuid": userID}, user); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user reputation"})
			}
			return c.JSON(http.StatusOK, map[string]string{"message": "Reputation changed to negative"})
		}
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "You have already given reputation to this user"})
	}

	// Find the user by ID
	if err := db.Collection("users").FindOne(context.Background(), bson.M{"uuid": userID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	// Increment the user's reputation
	user.MinusReputation++

	// Save the recent reputation
	recentRep = RecentReputation{MinusReputation: 1, IP: ip, ID: userID}
	if _, err := collection.InsertOne(context.Background(), recentRep); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recent reputation"})
	}

	// Update the user's reputation in the database
	if _, err := db.Collection("users").ReplaceOne(context.Background(), bson.M{"uuid": userID}, user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update reputation"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Reputation decreased"})
}

type UserResponse struct {
	Username        string `json:"username"`
	UUID            string `json:"uuid"`
	Groups          Groups `json:"groups"`
	PlusReputation  int    `json:"plus_reputation"`
	MinusReputation int    `json:"minus_reputation"`
	Posts           int    `json:"posts"`
	Threads         int    `json:"threads"`
}

type Groups struct {
	Admin     bool        `json:"admin"`
	Moderator bool        `json:"moderator"`
	Janny     JannyBoards `json:"janny"`
}

type JannyBoards struct {
	Boards []string `json:"boards"`
}

func convertAuthGroupToGroups(authGroup models.Group) Groups {
	return Groups{
		Admin:     authGroup.Admin,
		Moderator: authGroup.Moderator,
		Janny: JannyBoards{
			Boards: authGroup.Janny.Boards,
		},
	}
}

func GetUser(c echo.Context) error {
	db := database.DB_Users
	userID := c.Param("id")

	if err := db.Collection("users").FindOne(context.Background(), bson.M{"uuid": userID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	response := UserResponse{
		Username:        user.Username,
		UUID:            user.UUID,
		Groups:          convertAuthGroupToGroups(user.Groups),
		PlusReputation:  user.PlusReputation,
		MinusReputation: user.MinusReputation,
		Posts:           user.Posts,
		Threads:         user.Threads,
	}

	return c.JSON(http.StatusOK, response)
}

func GetUserReputation(c echo.Context) error {
	db := database.DB_Users
	userID := c.Param("id")

	if err := db.Collection("users").FindOne(context.Background(), bson.M{"uuid": userID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to query user"})
	}

	return c.JSON(http.StatusOK, map[string]int{
		"plus_reputation":  user.PlusReputation,
		"minus_reputation": user.MinusReputation,
	})
}

func ListUsers(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB_Users
	cursor, err := db.Collection("users").Find(context.Background(), bson.M{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users"})
	}
	defer cursor.Close(context.Background())

	var users []UserResponse
	for cursor.Next(context.Background()) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode user"})
		}
		users = append(users, UserResponse{
			Username:        user.Username,
			UUID:            user.UUID,
			Groups:          convertAuthGroupToGroups(user.Groups),
			PlusReputation:  user.PlusReputation,
			MinusReputation: user.MinusReputation,
			Posts:           user.Posts,
			Threads:         user.Threads,
		})
	}

	return c.JSON(http.StatusOK, users)

}

func ListUsersByReputation(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB_Users
	cursor, err := db.Collection("users").Find(context.Background(), bson.M{}, &options.FindOptions{
		Sort: bson.M{"plus_reputation": -1},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users by reputation"})
	}
	defer cursor.Close(context.Background())

	var users []UserResponse
	for cursor.Next(context.Background()) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode user"})
		}
		users = append(users, UserResponse{
			Username:        user.Username,
			UUID:            user.UUID,
			Groups:          convertAuthGroupToGroups(user.Groups),
			PlusReputation:  user.PlusReputation,
			MinusReputation: user.MinusReputation,
			Posts:           user.Posts,
			Threads:         user.Threads,
		})
	}

	return c.JSON(http.StatusOK, users)
}

func ListUsersByJoinDate(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB_Users
	cursor, err := db.Collection("users").Find(context.Background(), bson.M{}, &options.FindOptions{
		Sort: bson.M{"join_date": 1},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users by join date"})
	}
	defer cursor.Close(context.Background())

	var users []UserResponse
	for cursor.Next(context.Background()) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode user"})
		}
		users = append(users, UserResponse{
			Username:        user.Username,
			UUID:            user.UUID,
			Groups:          convertAuthGroupToGroups(user.Groups),
			PlusReputation:  user.PlusReputation,
			MinusReputation: user.MinusReputation,
			Posts:           user.Posts,
			Threads:         user.Threads,
		})
	}

	return c.JSON(http.StatusOK, users)

}

func ListUsersByLastLogin(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB_Users
	cursor, err := db.Collection("users").Find(context.Background(), bson.M{}, &options.FindOptions{
		Sort: bson.M{"last_login": -1},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list users by last login"})
	}
	defer cursor.Close(context.Background())

	var users []UserResponse
	for cursor.Next(context.Background()) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode user"})
		}
		users = append(users, UserResponse{
			Username:        user.Username,
			UUID:            user.UUID,
			Groups:          convertAuthGroupToGroups(user.Groups),
			PlusReputation:  user.PlusReputation,
			MinusReputation: user.MinusReputation,
			Posts:           user.Posts,
			Threads:         user.Threads,
		})
	}

	return c.JSON(http.StatusOK, users)

}
func ListAdmins(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB_Users
	var users []models.User
	cursor, err := db.Collection("users").Find(context.Background(), bson.M{"groups.admin": true})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list admins"})
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &users); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list admins"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListModerators(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB_Users
	var users []models.User
	cursor, err := db.Collection("users").Find(context.Background(), bson.M{"groups.moderator": true})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list moderators"})
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &users); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list moderators"})
	}

	return c.JSON(http.StatusOK, users)
}

func ListJannies(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := database.DB_Users
	var users []models.User
	cursor, err := db.Collection("users").Find(context.Background(), bson.M{"groups.janny.boards": bson.M{"$ne": nil}})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list jannies"})
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &users); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list jannies"})
	}

	return c.JSON(http.StatusOK, users)
}

func UpdateUserGroups(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	db := database.DB_Users
	userID := c.FormValue("id")
	admin := c.FormValue("admin")
	moderator := c.FormValue("moderator")
	janny := c.FormValue("janny")
	jannyboards := c.FormValue("jannyboards")
	if userID == "1337" {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You cannot change the groups of the root user"})
	}
	var user models.User

	// Find the user by ID
	if err := db.Collection("users").FindOne(context.Background(), bson.M{"uuid": userID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
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

	user.Permanent = true

	// Update janny status and jannyboards
	if janny == "on" {
		user.Groups.Janny.Boards = strings.Split(jannyboards, ",")
	} else {
		user.Groups.Janny.Boards = []string{}
	}

	// Save the updated user to the database
	if err := db.Collection("users").FindOneAndReplace(context.Background(), bson.M{"uuid": userID}, user).Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user groups"})
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
