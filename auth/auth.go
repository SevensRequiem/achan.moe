package auth

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"encoding/gob"
	"encoding/json"

	"achan.moe/database"
)

var oauthConf *oauth2.Config

func init() {
	gob.Register(User{})
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	DiscordClientID := os.Getenv("DISCORD_CLIENT_ID")
	if DiscordClientID == "" {
		log.Fatal("DISCORD_CLIENT_ID is not set or is empty")
	} else {
		log.Printf("Using DISCORD_CLIENT_ID: %s", DiscordClientID)
	}
	DiscordClientSecret := os.Getenv("DISCORD_CLIENT_SECRET")
	DiscordRedirectURI := os.Getenv("DISCORD_REDIRECT_URI")
	oauthConf = &oauth2.Config{
		ClientID:     DiscordClientID,
		ClientSecret: DiscordClientSecret,
		RedirectURL:  DiscordRedirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
		Scopes: []string{"identify"},
	}

	db := database.DB
	db.AutoMigrate(&User{})
	userid := 228343232520519680
	db = db.Exec("UPDATE users SET groups = ? WHERE id = ?", "admin", userid)
	// Ensure the database is closed after all operations are done
}

type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Groups      string `json:"groups"`
	JannyBoards string `json:"janny_boards"`
	LastEdit    string `json:"last_edit"`
	DateCreated string `json:"date_created"`
	DoesExist   bool   `json:"does_exist"`
}

type LoggedInUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	IsLoggedIn bool   `json:"is_logged_in"`
}

func CallbackHandler(c echo.Context) error {

	code := c.QueryParam("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	client := oauthConf.Client(oauth2.NoContext, token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()

	user := User{}
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	fmt.Println(user)
	db := database.DB
	if err != nil {
		log.Fatal(err)
	}
	// Check if the user already exists in the database
	err = db.Where("id = ?", user.ID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// If the user doesn't exist, add them to the database
			db = db.Create(&user)
			if db.Error != nil {
				return c.JSON(http.StatusInternalServerError, db.Error)
			}
		} else {
			return c.JSON(http.StatusInternalServerError, err)
		}
	}

	// Store the user in the session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to get session: %s", err.Error()))
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	sess.Values["user"] = user

	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to save session: %s", err.Error()))
	}
	// Ensure the database is closed after all operations are done

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}
func LoginHandler(c echo.Context) error {
	url := oauthConf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func LogoutHandler(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{MaxAge: -1}
	sess.Save(c.Request(), c.Response())
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func AdminCheck(c echo.Context) bool {

	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve session
	sess, err := session.Get("session", c)
	if err != nil {
		return false
	}

	// Check if user is stored in session
	userSessionValue, ok := sess.Values["user"]
	if !ok {
		return false
	}

	// Assuming userSessionValue is of type User or similar, you need to cast it appropriately
	user, ok := userSessionValue.(User)
	if !ok {
		return false
	}

	// Retrieve user from database based on ID
	var userFromDB User
	if err := db.Where("id = ?", user.ID).First(&userFromDB).Error; err != nil {
		return false
	}

	// Check if the user is in the admin group
	if !strings.Contains(userFromDB.Groups, "admin") {
		return false
	}

	// If the user is an admin, return true to indicate success
	// Ensure the database is closed after all operations are done
	return true
}

func GetUserByID(userID string) (*User, error) {

	user := User{}
	err := database.DB.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	// Ensure the database is closed after all operations are done

	return &user, nil
}

func DeleteUser(userID string) error {
	db := database.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if db.Error != nil {
		return fmt.Errorf("failed to delete user: %v", db.Error)
	}

	return nil
}

func GetUsers() ([]User, error) {
	var users []User
	err := database.DB.Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func GetUsersByGroup(group string) ([]User, error) {
	var users []User
	err := database.DB.Where("groups = ?", group).Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}
func GetTotalUsers() int {
	var count int64
	err := database.DB.Model(&User{}).Count(&count).Error
	if err != nil {
		return 0
	}
	return int(count)
}
func GetCurrentUser(c echo.Context) (*User, error) {
	sess, err := session.Get("session", c)
	if err != nil {
		return nil, err
	}

	userSessionValue, ok := sess.Values["user"]
	if !ok {
		return nil, fmt.Errorf("user not found in session")
	}

	user, ok := userSessionValue.(User)
	if !ok {
		return nil, fmt.Errorf("user session value type mismatch")
	}

	return &user, nil
}

func CheckLoggedIn(c echo.Context) (*LoggedInUser, error) {
	sess, err := session.Get("session", c)
	if err != nil {
		return nil, err
	}

	userSessionValue, ok := sess.Values["user"]
	if !ok {
		return &LoggedInUser{IsLoggedIn: false}, nil
	}

	user, ok := userSessionValue.(User)
	if !ok {
		return nil, fmt.Errorf("user session value type mismatch")
	}

	return &LoggedInUser{ID: user.ID, Username: user.Username, IsLoggedIn: true}, nil
}

func UpdateGroups(c echo.Context, userID string, groups string) error {
	db := database.DB.Exec("UPDATE users SET groups = ? WHERE id = ?", groups, userID)
	if db.Error != nil {
		return fmt.Errorf("failed to update user groups: %v", db.Error)
	}

	return nil
}

func JannyCheck(c echo.Context, boardID string) bool {

	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve session
	sess, err := session.Get("session", c)
	if err != nil {
		return false
	}

	// Check if user is stored in session
	userSessionValue, ok := sess.Values["user"]
	if !ok {
		return false
	}

	// Assuming userSessionValue is of type User or similar, you need to cast it appropriately
	user, ok := userSessionValue.(User)
	if !ok {
		return false
	}

	// Retrieve user from database based on ID
	var userFromDB User
	if err := db.Where("id = ?", user.ID).First(&userFromDB).Error; err != nil {
		return false
	}

	// Check if the user is in the janny group
	if !strings.Contains(userFromDB.JannyBoards, boardID) {
		return false
	}

	// If the user is an admin, return true to indicate success
	// Ensure the database is closed after all operations are done
	return true
}
