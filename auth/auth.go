package auth

import (
	"context"
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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/exp/rand"
	"golang.org/x/time/rate"

	"encoding/gob"

	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/utils/mail"
)

func init() {
	gob.Register(User{})
	DefaultAdmin()
}

func DefaultAdmin() {
	db := database.DB_Main
	var user User
	db.Collection("users").FindOne(context.TODO(), bson.M{"uuid": "1337"}).Decode(&user)
	if user.Username == "" {
		NewManualUser("1337", "admin", "admin")
		logs.Info("Default Admin Created")
	}
}

type User struct {
	ID              uint `gorm:"primaryKey"`
	UUID            string
	Username        string
	Password        string
	Groups          Group
	DateCreated     string
	LastLogin       string
	DoesExist       bool
	Premium         bool
	Permanent       bool
	Banned          bool
	Email           string
	TransactionID   string
	PlusReputation  int
	MinusReputation int
	Posts           int
	Threads         int
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
		logs.Info("unsupported data type: %T", value, "in JannyBoards Scan")
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
	err := database.DB_Main.Collection("users").FindOne(context.TODO(), bson.M{"UUID": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logs.Info("Generated ID: ", id, " for new user")
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
	err := database.DB_Main.Collection("users").FindOne(context.TODO(), bson.M{"username": string(username)}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logs.Info("Generated username: ", string(username), " for new user")
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
	logs.Info("Generated password: ", string(password), " for new user")
	return string(password)
}
func encryptPassword(password string) string {
	enc := os.Getenv("ENCRYPT_KEY")
	h := hmac.New(sha256.New, []byte(enc))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
	encodedPass := base64.StdEncoding.EncodeToString(encpass)
	logs.Debug("Encoded Pass for new user")
	return encodedPass
}

func ManualGenPassword(password string) string {
	enc := os.Getenv("ENCRYPT_KEY")
	h := hmac.New(sha256.New, []byte(enc))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
	encodedPass := base64.StdEncoding.EncodeToString(encpass)
	logs.Debug("Manually Generated Password for user")
	return encodedPass
}

var limiter = rate.NewLimiter(1/60.0, 1)

func NewUser(c echo.Context) error {
	if !limiter.Allow() {
		logs.Debug("One Minute cooldown for new user")
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "One Minute cooldown"})
	}

	var newid = Getrandid()
	var newusername = getrandusername()
	var newpassword = getrandpassword()
	var encpass = encryptPassword(newpassword)
	db := database.DB_Main
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

	db.Collection("users").InsertOne(context.Background(), user)
	info := map[string]string{"username": user.Username, "password": newpassword}
	logs.Debug("New User Created", user.Username)
	return c.JSON(http.StatusOK, info)
}
func NewManualUser(userID string, username string, password string) {
	db := database.DB_Main
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
	db.Collection("users").InsertOne(context.Background(), user)
	logs.Debug("New Manual User Created", user.Username)
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
	logs.Debug("Decoded Password for user")
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
		logs.Error("Empty username or password")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid username or password"})
	}

	// Retrieve the user from the database based on the username
	user := GetUserByUsername(username)

	// Check if the password is correct
	if !checkPassword(password, user) {
		logs.Error("Invalid username or password")
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
	logs.Debug("Checking Password for user")
	return encodedPass == user.Password
}
func LogoutHandler(c echo.Context) error {
	// Retrieve the session
	sess, err := session.Get("session", c)
	if err != nil {
		logs.Error("Failed to get session")
		return err
	}

	// Clear the session
	sess.Options.MaxAge = -1
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		logs.Error("Failed to save session")
		return err
	}

	// Redirect the user to the home page
	logs.Debug("User Logged Out")
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
	logs.Debug("Admin Check")
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
	logs.Debug("Moderator Check")
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
	logs.Debug("Janny Check for board: ", board)
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
	logs.Debug("Moderator Check")
	return user.Groups.Moderator
}
func AuthCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	_, ok := sess.Values["user"].(User)
	logs.Debug("Auth Check", ok)
	return ok
}

func LoggedInUser(c echo.Context) User {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Retrieve the user from the session
	user, _ := sess.Values["user"].(User)
	logs.Debug("Logged In User", user.Username)
	return user
}
func BannedCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(User)
	if !ok {
		return false
	}

	// Check if the user is banned
	logs.Debug("Banned Check")
	return user.Banned
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
	logs.Debug("Premium Check")
	return user.Premium
}

// funcs
// ////////////

func EditUser(c echo.Context) error {
	db := database.DB_Main
	var user User
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Validate input parameters
	if username == "" {
		logs.Error("Username is required")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Username is required"})
	}

	// Retrieve user by username
	user = GetUserByUsername(username)

	// Check if user exists
	if user.UUID == "" {
		logs.Error("User not found")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Update password if provided
	if password != "" {
		user.Password = ManualGenPassword(password)
	}

	// Save updated user to the database
	if err := db.Collection("users").FindOneAndUpdate(context.TODO(), bson.M{"UUID": user.UUID}, bson.M{"$set": user}).Err(); err != nil {
		logs.Error("Failed to update user")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
	}

	return c.JSON(http.StatusOK, user)
}

func UpdateUser(c echo.Context) error {
	// get user from session
	sess, err := session.Get("session", c)
	if err != nil {
		logs.Error("Failed to get session")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get session"})
	}

	user, ok := sess.Values["user"].(User)
	if !ok {
		logs.Error("User not found in session")
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
	db := database.DB_Main
	if err := db.Collection("users").FindOneAndUpdate(context.TODO(), bson.M{"UUID": user.UUID}, bson.M{"$set": user}).Err(); err != nil {
		logs.Error("Failed to update user")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// delete session
	sess.Options.MaxAge = -1
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		logs.Error("Failed to save session")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save session"})
	}

	return nil
}
func GetTotalUsers() int64 {
	// Obtain the database connection from the `database` package
	db := database.DB_Main

	// Retrieve the total number of users from the database
	var count int64
	db.Collection("users").CountDocuments(context.Background(), bson.M{})

	// Return the total number of users
	logs.Debug("Total Users: ", count)
	return count
}

func GetUserByID(uuid uint) User {
	// Obtain the database connection from the `database` package
	db := database.DB_Main

	// Retrieve the user from the database based on the ID
	var user User
	db.Collection("users").FindOne(context.TODO(), bson.M{"UUID": uuid}).Decode(&user)

	// Return the user
	logs.Debug("User Found: ", user.Username)
	return user
}

func GetUserByUsername(username string) User {
	// Obtain the database connection from the `database` package
	db := database.DB_Main

	// Retrieve the user from the database based on the username
	var user User
	db.Collection("users").FindOne(context.TODO(), bson.M{"username": username}).Decode(&user)

	// Return the user
	logs.Debug("User Found: ", user.Username)
	return user
}

func ListAdmins() []User {
	// Obtain the database connection from the `database` package
	db := database.DB_Main

	// Retrieve the admins from the database
	var admins []User
	collection := db.Collection("users")
	cursor, err := collection.Find(context.Background(), bson.M{"groups.admin": true})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &admins); err != nil {
		log.Fatal(err)
	}

	// Return the admins
	logs.Debug("Admins Found: ", admins)
	return admins
}

func ListModerators() []User {
	// Obtain the database connection from the `database` package
	db := database.DB_Main

	// Retrieve the moderators from the database
	var moderators []User
	collection := db.Collection("users")
	cursor, err := collection.Find(context.Background(), bson.M{"groups.moderator": true})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &moderators); err != nil {
		log.Fatal(err)
	}

	// Return the moderators
	logs.Debug("Moderators Found: ", moderators)
	return moderators
}

func ListJannies() []User {
	// Obtain the database connection from the `database` package
	db := database.DB_Main

	// Retrieve the jannies from the database
	var jannies []User
	collection := db.Collection("users")
	cursor, err := collection.Find(context.Background(), bson.M{"groups.janny": bson.M{"$ne": nil}})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &jannies); err != nil {
		log.Fatal(err)
	}

	// Return the jannies
	logs.Debug("Jannies Found: ", jannies)
	return jannies
}
func ExpireUsers() {
	fmt.Println("Checking user login expirations....")
	db := database.DB_Main

	var users []User
	collection := db.Collection("users")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &users); err != nil {
		log.Fatal(err)
	}

	// Iterate over the users
	for _, user := range users {
		if !user.Premium || !user.Permanent {
			lastLogin, err := time.Parse("2006-01-02 15:04:05", user.LastLogin)
			if err != nil {
				logs.Error("Failed to parse last login time")
				continue
			}

			if time.Since(lastLogin).Hours() > 720 {
				_, err := collection.DeleteOne(context.Background(), bson.M{"UUID": user.UUID})
				if err != nil {
					logs.Error("Failed to delete user: ", err)
				}
			}
		}
	}
}

func NewPremiumUser(c echo.Context, email string, transactionid string) {
	userID := Getrandid()
	db := database.DB_Main
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
	db.Collection("users").InsertOne(context.Background(), user)
	//login the user
	sess, _ := session.Get("session", c)
	sess.Values["user"] = user
	sess.Save(c.Request(), c.Response())
	logs.Info("New Premium User Created", user.Username)
	go mail.AddMailToQueue(email, "Welcome to achan.moe!", "Your account has been created successfully! Your username is: "+user.Username+" and your password is: "+user.Password+" You can change both your username and password after logging in.")
	c.Redirect(http.StatusTemporaryRedirect, "/profile")
}

func DeleteUser(c echo.Context) error {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Retrieve the user from the session
	user, ok := sess.Values["user"].(User)
	if !ok {
		logs.Error("User not found in session")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found in session"})
	}

	// Retrieve the user from the database
	db := database.DB_Main
	var u User
	db.Collection("users").FindOne(context.TODO(), bson.M{"UUID": user.UUID}).Decode(&u)

	// Delete the user from the database
	if err := db.Collection("users").FindOneAndDelete(context.TODO(), bson.M{"UUID": user.UUID}).Err(); err != nil {
		logs.Error("Failed to delete user")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete user"})
	}

	// Clear the session
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	// Redirect the user to the home page
	logs.Debug("User Deleted", user.Username)
	return c.JSON(http.StatusOK, map[string]string{"success": "User deleted"})
}
