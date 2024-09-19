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
	"strings"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	"encoding/gob"

	"achan.moe/database"
	"achan.moe/utils/mail"
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
	user := User{UUID: Getrandid(), Username: getrandusername(), Password: getrandpassword(), Groups: Group{Admin: true, Moderator: true, Janny: JannyBoards{Boards: []string{"a", "b"}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
	db.Create(&user)
	user = User{UUID: Getrandid(), Username: getrandusername(), Password: getrandpassword(), Groups: Group{Admin: false, Moderator: true, Janny: JannyBoards{Boards: []string{"c", "d"}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
	db.Create(&user)
	user = User{UUID: Getrandid(), Username: getrandusername(), Password: getrandpassword(), Groups: Group{Admin: false, Moderator: false, Janny: JannyBoards{Boards: []string{}}}, DateCreated: time.Now().Format("2006-01-02 15:04:05"), LastLogin: time.Now().Format("2006-01-02 15:04:05"), DoesExist: true}
	db.Create(&user)
}

type User struct {
	UUID          string
	Username      string
	Password      string
	Groups        Group
	DateCreated   string
	LastLogin     string
	DoesExist     bool
	Premium       bool
	Email         string
	TransactionID string
	Reputation    int
	Posts         int
	Threads       int
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

func Getrandid() string {
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
	return Getrandid()
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
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"
	password := make([]byte, 10)
	seed := uint64(rand.Intn(1000000000000000000))
	rand.Seed(seed)
	for i := range password {
		num := rand.Intn(len(chars))
		password[i] = chars[num]
	}
	return string(password)
}
func encryptPassword(password string) string {
	enc := os.Getenv("ENCRYPT_KEY")
	h := hmac.New(sha256.New, []byte(enc))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
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

var limiter = rate.NewLimiter(1/60.0, 1)

func NewUser(c echo.Context) error {
	if !limiter.Allow() {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "One Minute cooldown"})
	}

	var newid = Getrandid()
	var newusername = getrandusername()
	var newpassword = getrandpassword()
	var encpass = encryptPassword(newpassword)
	db := database.DB
	user := User{
		UUID:          newid,
		Username:      newusername,
		Password:      encpass,
		Groups:        Group{Admin: false, Moderator: false, Janny: JannyBoards{Boards: []string{}}},
		DateCreated:   time.Now().Format("2006-01-02 15:04:05"),
		LastLogin:     time.Now().Format("2006-01-02 15:04:05"),
		DoesExist:     true,
		Premium:       false,
		Email:         "",
		TransactionID: "",
	}

	db.Create(&user)
	info := map[string]string{"username": user.Username, "password": newpassword}
	return c.JSON(http.StatusOK, info)
}
func NewManualUser(userID string, username string, password string) {
	db := database.DB
	user := User{
		UUID:          userID,
		Username:      username,
		Password:      ManualGenPassword(password),
		Groups:        Group{Admin: false, Moderator: false, Janny: JannyBoards{Boards: []string{}}},
		DateCreated:   time.Now().Format("2006-01-02 15:04:05"),
		LastLogin:     time.Now().Format("2006-01-02 15:04:05"),
		DoesExist:     true,
		Premium:       false,
		Email:         "",
		TransactionID: "",
	}
	db.Create(&user)
}
func DecodePassword(encrypted_password string) string {
	enc := os.Getenv("ENCRYPT_KEY")
	decoded, err := base64.StdEncoding.DecodeString(encrypted_password)

	if err != nil {
		log.Println(err)
		return ""
	}
	h := hmac.New(sha256.New, []byte(enc))
	h.Write(decoded)
	encpass := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(encpass)
}

// login functions
//////////////////

func LoginHandler(c echo.Context) error {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Parse the request parameters
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid username or password"})
	}

	// Retrieve the user from the database based on the username
	user := GetUserByUsername(username)

	// Check if the password is correct
	if !checkPassword(password, user) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
	}

	// Update the last login time
	user.LastLogin = time.Now().Format("2006-01-02 15:04:05")

	// Save the user to the session
	sess.Values["user"] = user
	sess.Save(c.Request(), c.Response())

	// Redirect the user to the home page
	return c.JSON(http.StatusOK, map[string]string{"success": "Logged in"})
}
func checkPassword(password string, user User) bool {
	enc := os.Getenv("ENCRYPT_KEY")
	h := hmac.New(sha256.New, []byte(enc))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
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
func ModCheck(c echo.Context) bool {
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
func AuthCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	_, ok := sess.Values["user"].(User)
	return ok
}

func PremiumCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(User)
	if !ok {
		return false
	}

	// Check if the user is premium
	return user.Premium
}

// funcs
// ////////////

func EditUser(c echo.Context) error {
	db := database.DB
	var user User
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Validate input parameters
	if username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Username is required"})
	}

	// Retrieve user by username
	user = GetUserByUsername(username)

	// Check if user exists
	if user.UUID == "" {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Update password if provided
	if password != "" {
		user.Password = ManualGenPassword(password)
	}

	// Save updated user to the database
	if err := db.Where("uuid = ?", user.UUID).Updates(user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, user)
}

func UpdateUser(c echo.Context) error {
	// get user from session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get session"})
	}

	user, ok := sess.Values["user"].(User)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found in session"})
	}

	// update user fields
	if username := c.FormValue("username"); username != "" {
		user.Username = username
	}
	if password := c.FormValue("password"); password != "" {
		user.Password = ManualGenPassword(password)
	}

	// save user to database
	db := database.DB
	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// delete session
	sess.Options.MaxAge = -1
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save session"})
	}

	return c.JSON(http.StatusOK, user)
}
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
	db.Where("uuid = ?", uuid).First(&user)

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

func ExpireUsers() {
	fmt.Println("Checking user login expirations....")
	db := database.DB

	var users []User
	db.Find(&users)

	// Iterate over the users
	for _, user := range users {
		if !user.Premium {
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
}

func NewPremiumUser(c echo.Context, email string, transactionid string) {
	userID := Getrandid()
	db := database.DB
	user := User{
		UUID:     userID,
		Username: strings.Split(email, "@")[0],
		Password: ManualGenPassword("password"), // Set a default password or generate one
		Groups: Group{
			Admin:     false,
			Moderator: false,
			Janny:     JannyBoards{Boards: []string{}},
		},
		DateCreated:   time.Now().Format("2006-01-02 15:04:05"),
		LastLogin:     time.Now().Format("2006-01-02 15:04:05"),
		DoesExist:     true,
		Premium:       true,
		Email:         email,
		TransactionID: transactionid,
	}
	db.Create(&user)
	//login the user
	sess, _ := session.Get("session", c)
	sess.Values["user"] = user
	sess.Save(c.Request(), c.Response())
	go mail.SendEmail(email, "Welcome to achan.moe!", "Your account has been created successfully! Your username is: "+user.Username+" and your password is: "+user.Password+" You can change both your username and password after logging in.")
	c.Redirect(http.StatusTemporaryRedirect, "/profile")
}
