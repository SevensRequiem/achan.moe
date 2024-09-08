package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"
	"gorm.io/gorm"

	"encoding/gob"

	"achan.moe/database"
)

func init() {
	gob.Register(User{})
	db := database.DB
	db.AutoMigrate(&User{})
	//dummydata()
	DefaultAdmin()
}
func DefaultAdmin() {
	db := database.DB
	var user User
	db.Where("UUID = ?", 1337).First(&user)
	if user.Username == "" {
		user := User{UUID: "1337", Username: "admin", Password: ManualGenPassword("admin"), Groups: Group{Admin: true, Moderator: true, Janny: JannyBoards{Boards: []string{}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
		db.Create(&user)
	}
}
func dummydata() {
	db := database.DB
	user := User{UUID: getrandid(), Username: getrandusername(), Password: getrandpassword(), Groups: Group{Admin: true, Moderator: true, Janny: JannyBoards{Boards: []string{"a", "b"}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
	db.Create(&user)
	user = User{UUID: getrandid(), Username: getrandusername(), Password: getrandpassword(), Groups: Group{Admin: false, Moderator: true, Janny: JannyBoards{Boards: []string{"c", "d"}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
	db.Create(&user)
	user = User{UUID: getrandid(), Username: getrandusername(), Password: getrandpassword(), Groups: Group{Admin: false, Moderator: false, Janny: JannyBoards{Boards: []string{}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
	db.Create(&user)
}

type User struct {
	PrimaryID   uint   `gorm:"primary_key"`
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Groups      Group  `json:"groups" gorm:"type:json"`
	DateCreated string `json:"date_created" gorm:"default:CURRENT_TIMESTAMP"`
	LastLogin   string `json:"last_login"`
	DoesExist   bool   `json:"does_exist"`
}

type Group struct {
	Admin     bool        `json:"admin"`
	Moderator bool        `json:"moderator"`
	Janny     JannyBoards `json:"janny"`
}

type JannyBoards struct {
	Boards []string `json:"boards"`
}

// Implement the Scanner interface for Group
func (g *Group) Scan(value interface{}) error {
	if value == nil {
		*g = Group{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported data type: %T", value)
	}
	return json.Unmarshal(bytes, g)
}

// Implement the Valuer interface for Group
func (g Group) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// Implement the Scanner interface for JannyBoards
func (jb *JannyBoards) Scan(value interface{}) error {
	if value == nil {
		*jb = JannyBoards{Boards: []string{}}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported data type: %T", value)
	}
	return json.Unmarshal(bytes, jb)
}

// Implement the Valuer interface for JannyBoards
func (jb JannyBoards) Value() (driver.Value, error) {
	return json.Marshal(jb)
}

// new user functions
/////////////////////

func getrandid() string {
	// Generate a random ID
	id := fmt.Sprintf("%d", rand.Intn(1000000000))
	var user User

	// Check if the ID already exists in the database
	err := database.DB.Where("uuid = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// ID does not exist, return the generated ID
			return id
		}
		// Log any other error
		log.Println(err)
	}

	// If the ID exists, generate a new one
	return getrandid()
}

func getrandusername() string {
	// Generate a random username
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	username := make([]byte, 10)
	for i := range username {
		username[i] = chars[rand.Intn(len(chars))]
	}
	var user User

	// Check if the username already exists in the database
	err := database.DB.Where("username = ?", string(username)).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Username does not exist, return the generated username
			return string(username)
		}
		// Log any other error
		log.Println(err)
	}

	// If the username exists, generate a new one
	return getrandusername()
}

func getrandpassword() string {
	enc := os.Getenv("ENCRYPT_KEY")
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"
	password := make([]byte, 10)
	for i := range password {
		password[i] = chars[rand.Intn(len(chars))]
	}
	h := hmac.New(sha256.New, []byte(enc))
	h.Write(password)
	encpass := h.Sum(nil)
	// Encode the hash in Base64
	encodedPass := base64.StdEncoding.EncodeToString(encpass)
	return encodedPass
}

func ManualGenPassword(password string) string {
	enc := os.Getenv("ENCRYPT_KEY")
	h := hmac.New(sha256.New, []byte(enc))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
	encodedPass := base64.StdEncoding.EncodeToString(encpass)
	return encodedPass
}

func NewUser(c echo.Context) error {
	var newid = getrandid()
	var newusername = getrandusername()
	var newpassword = getrandpassword()
	db := database.DB
	user := User{UUID: newid, Username: newusername, Password: newpassword, Groups: Group{Admin: false, Moderator: false, Janny: JannyBoards{Boards: []string{}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
	db.Create(&user)
	// encode in json
	info := map[string]string{"username": user.Username, "password": user.Password}
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
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is already logged in
	if sess.Values["user"] != nil {
		// Redirect the user to the home page
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	// Retrieve the username and password from the form
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Obtain the database connection from the `database` package
	db := database.DB

	// Retrieve the user from the database based on the username
	var user User
	db.Where("username = ?", username).First(&user)

	// Check if the user exists and the password is correct
	if user.DoesExist && checkPassword(password, user) {
		// Update the user's last login time
		user.LastLogin = time.Now().Format("2006-01-02 15:04:05")
		db.Save(&user)

		// Store the user in the session
		sess.Values["user"] = user
		sess.Save(c.Request(), c.Response())

		// Redirect the user to the home page
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	// Redirect the user to the login page
	return c.Redirect(http.StatusTemporaryRedirect, "/login")

}
func checkPassword(password string, user User) bool {
	enc := os.Getenv("ENCRYPT_KEY")
	h := hmac.New(sha256.New, []byte(enc))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
	// Encode the hash in Base64
	encodedPass := base64.StdEncoding.EncodeToString(encpass)
	return encodedPass == user.Password
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

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(User)
	if !ok {
		return false
	}

	// Check if the user is an admin
	return user.Groups.Admin
}

func ModeratorCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(User)
	if !ok {
		return false
	}

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

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(User)
	if !ok {
		return false
	}

	// Check if the user is a janny
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

func ExpireUser() {
	db := database.DB

	var users []User
	db.Find(&users)

	// Iterate over the users
	for _, user := range users {
		lastLogin, err := time.Parse("2006-01-02 15:04:05", user.LastLogin)
		if err != nil {
			log.Println(err)
			continue
		}

		if time.Since(lastLogin).Hours() > 720 {

			db.Delete(&user)
		}
	}
}
