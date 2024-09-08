package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"

	"encoding/gob"

	"achan.moe/database"
)

func init() {
	gob.Register(User{})
	db := database.DB
	db.AutoMigrate(&User{})
}

type User struct {
	PrimaryID   uint   `gorm:"primary_key"`
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Groups      Group
	DateCreated string `json:"date_created" gorm:"default:CURRENT_TIMESTAMP"`
	LastLogin   string `json:"last_login"`
	DoesExist   bool   `json:"does_exist"`
}

type Group struct {
	Admin     bool `json:"admin"`
	Moderator bool `json:"moderator"`
	Janny     JannyBoards
}

type JannyBoards struct {
	Boards []string `json:"boards"`
}

// new user functions
/////////////////////

func getrandid() string {
	// Generate a random ID and check if it already exists in the database, if it does, generate a new one
	id := fmt.Sprintf("%d", rand.Intn(1000000000))
	var user User
	err := database.DB.Where("id = ?", id).First(&user).Error
	if err != nil {
		log.Println(err)
	}
	if user.UUID != "" {
		getrandid()
	}
	return id
}

func getrandusername() string {
	// Generate a random username and check if it already exists in the database, if it does, generate a new one
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	username := make([]byte, 10)
	for i := range username {
		username[i] = chars[rand.Intn(len(chars))]
	}
	var user User
	err := database.DB.Where("username = ?", string(username)).First(&user).Error
	if err != nil {
		log.Println(err)
	}
	if user.Username != "" {
		getrandusername()
	}
	return string(username)
}

func getrandpassword() string {
	enc := os.Getenv("ENCRYPT_KEY")

	// Generate a random password using hmac encryption

	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"
	password := make([]byte, 10)
	for i := range password {
		password[i] = chars[rand.Intn(len(chars))]
	}
	h := hmac.New(sha256.New, []byte(enc))
	h.Write(password)
	encpass := h.Sum(nil)
	return string(encpass)

}

var newid = getrandid()
var newusername = getrandusername()
var newpassword = getrandpassword()

func NewUser(c echo.Context) error {
	// Create a new user with the generated random values
	db := database.DB
	user := User{UUID: newid, Username: newusername, Password: newpassword, Groups: Group{Admin: false, Moderator: false, Janny: JannyBoards{Boards: []string{}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: false}
	db.Create(&user)
	// encode in json
	info := map[string]string{"uuid": user.UUID, "username": user.Username, "password": user.Password, "date_created": user.DateCreated, "last_login": user.LastLogin}
	encinfo, err := json.Marshal(info)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(encinfo))
	return c.JSON(http.StatusOK, info)
}

// login functions
//////////////////

func LoginHandler(c echo.Context) error {
	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the username and password from the request
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Retrieve the user from the database based on the username
	var user User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid username or password"})
	}

	// Check if the password is correct
	h := hmac.New(sha256.New, []byte(os.Getenv("ENCRYPT_KEY")))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
	if user.Password != string(encpass) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid username or password"})
	}

	// Update the last login time
	user.LastLogin = time.Now().Format("2006-01-02 15:04:05")
	db.Save(&user)

	// Store the user in the session
	sess, _ := session.Get("session", c)
	sess.Values["user"] = user
	sess.Save(c.Request(), c.Response())

	// Redirect the user to the home page
	return c.Redirect(http.StatusTemporaryRedirect, "/")

}

func LogoutHandler(c echo.Context) error {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Clear the session
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	// Redirect the user to the home page
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

// checks
// ////////////
func AdminCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Retrieve the user from the session
	user := sess.Values["user"].(User)

	// Check if the user is an admin
	return user.Groups.Admin
}

func ModeratorCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Retrieve the user from the session
	user := sess.Values["user"].(User)

	// Check if the user is a moderator
	return user.Groups.Moderator
}
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
func JannyCheck(c echo.Context, board string) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Retrieve the user from the session
	user := sess.Values["user"].(User)

	// Check if the user is a janny for the board
	return contains(user.Groups.Janny.Boards, board)
}

func AuthCheck(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Retrieve the session
		sess, _ := session.Get("session", c)

		// Check if the user is logged in
		if sess.Values["user"] == nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/login")
		}

		// Call the next handler
		return next(c)
	}
}

// funcs
// ////////////
func GetTotalUsers() int64 {
	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the total number of users from the database
	var count int64
	db.Model(&User{}).Count(&count)

	// Return the total number of users
	return count
}

func GetUserByID(uuid uint) User {
	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the user from the database based on the ID
	var user User
	db.First(&user, uuid)

	// Return the user
	return user
}

func GetUserByUsername(username string) User {
	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the user from the database based on the username
	var user User
	db.Where("username = ?", username).First(&user)

	// Return the user
	return user
}

func ListAdmins() []User {
	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the admins from the database
	var admins []User
	db.Where("groups.admin = ?", true).Find(&admins)

	// Return the admins
	return admins
}

func ListModerators() []User {
	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the moderators from the database
	var moderators []User
	db.Where("groups.moderator = ?", true).Find(&moderators)

	// Return the moderators
	return moderators
}

func ListJannies() []User {
	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the jannies from the database
	var jannies []User
	db.Where("groups.janny = ?", true).Find(&jannies)

	// Return the jannies
	return jannies
}
