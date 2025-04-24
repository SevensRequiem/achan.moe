package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"encoding/gob"

	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/mail"
)

var enc = ""

func init() {
	gob.Register(models.User{})
	godotenv.Load()
	enc = os.Getenv("ENCRYPT_KEY")
	if enc == "" {
		logs.Fatal("ENCRYPT_KEY is not set")
	}
}

type Group struct {
	models.Group
}

type JannyBoards struct {
	models.JannyBoards
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

func Getrandid() uint {
	// Generate a random ID
	max := big.NewInt(9999999999)
	min := big.NewInt(1000000000)
	n, err := rand.Int(rand.Reader, new(big.Int).Sub(max, min))
	if err != nil {
		panic("Failed to generate random ID: " + err.Error())
	}
	id := fmt.Sprintf("%d", n.Add(n, min))
	var user models.User

	// Check if the ID already exists in the database
	err = database.DB_Main.Collection("users").FindOne(context.TODO(), bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logs.Info("Generated ID: ", id, " for new user")
			idInt, err := strconv.Atoi(id)
			if err != nil {
				logs.Error("Failed to convert ID to int")
				return 0
			}
			return uint(idInt)
		}
		// Log any other error
		log.Println(err)
	}
	// If the ID exists, generate a new one
	return Getrandid()
}

func getrandusername() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	username := make([]byte, 10)
	max := big.NewInt(int64(len(chars))) // Set max to the length of chars
	for i := 0; i < 10; i++ {
		n, err := rand.Int(rand.Reader, max) // Generate a random number within the range of chars
		if err != nil {
			panic("Failed to generate random username: " + err.Error())
		}
		username[i] = chars[n.Int64()]
	}
	var user models.User

	err := database.DB_Main.Collection("users").FindOne(context.TODO(), bson.M{"username": string(username)}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logs.Info("Generated username: ", string(username), " for new user")
			return string(username)
		}

		log.Println(err)
	}

	// If the username exists, generate a new one
	return getrandusername()
}

func getrandpassword() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"
	password := make([]byte, 10)
	max := big.NewInt(int64(len(chars))) // Set max to the length of chars
	for i := 0; i < 10; i++ {
		n, err := rand.Int(rand.Reader, max) // Generate a random number within the range of chars
		if err != nil {
			panic("Failed to generate random password: " + err.Error())
		}
		password[i] = chars[n.Int64()]
	}
	return string(password)
}
func encryptPassword(password string) string {
	h := hmac.New(sha256.New, []byte(enc))
	h.Write([]byte(password))
	encpass := h.Sum(nil)
	encodedPass := base64.StdEncoding.EncodeToString(encpass)
	logs.Debug("Encoded Pass for new user")
	return encodedPass
}

func ChangeUserGroups(groups models.Group, id string) {

	_, err := database.DB_Main.Collection("users").UpdateOne(context.TODO(), bson.M{"_id": id}, bson.M{"$set": bson.M{"groups": groups}})
	if err != nil {
		logs.Error("Failed to update user groups")
	}
}

func NewUser(c echo.Context) error {
	var newpassword = getrandpassword()
	user := models.User{
		ID:            Getrandid(),
		Username:      getrandusername(),
		Password:      encryptPassword(newpassword),
		Groups:        models.Group{Admin: false, Moderator: false, Janny: models.JannyBoards{Boards: []string{}}},
		DateCreated:   time.Now().Format("2006-01-02 15:04:05"),
		LastLogin:     time.Now().Format("2006-01-02 15:04:05"),
		DoesExist:     true,
		Premium:       false,
		Email:         "",
		TransactionID: "",
	}

	_, err := database.DB_Main.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		logs.Error("Failed to create new user" + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create new user"})
	}
	info := map[string]string{"username": user.Username, "password": newpassword}
	logs.Debug("New User Created", user.Username)
	return c.JSON(http.StatusOK, info)
}

// login functions
//////////////////

func LoginHandler(c echo.Context) error {
	// Parse the request parameters
	username := c.FormValue("username")
	password := c.FormValue("password")
	if username == "" || password == "" {
		logs.Error("Empty username or password")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Empty username or password"})
	}
	user := GetUserByUsername(username)
	if encryptPassword(password) != user.Password {
		logs.Error("Invalid password for user: ", username)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
	}
	user.LastLogin = time.Now().Format("2006-01-02 15:04:05")

	_, err := database.DB_Main.Collection("users").UpdateOne(context.TODO(), bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"last_login": user.LastLogin}})
	if err != nil {
		logs.Error("Failed to update last login time")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update last login time"})
	}
	sess, _ := session.Get("session", c)
	sess.Values["user"] = user
	sess.Save(c.Request(), c.Response())
	return c.JSON(http.StatusOK, map[string]string{"success": "Logged in"})
}
func LogoutHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		logs.Error("Failed to get session")
		return err
	}
	sess.Options.MaxAge = -1
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		logs.Error("Failed to save session")
		return err
	}
	logs.Debug("User Logged Out")
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

// checks
// ////////////
func AdminCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(models.User)
	if !ok {
		return false
	}

	// Check if the user is an admin
	logs.Debug("Admin Check " + user.Username + " " + strconv.FormatUint(uint64(user.ID), 10) + " " + strconv.FormatBool(user.Groups.Admin))
	return user.Groups.Admin
}

func ModeratorCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(models.User)
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
	user, ok := sess.Values["user"].(models.User)
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
	user, ok := sess.Values["user"].(models.User)
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
	_, ok := sess.Values["user"].(models.User)
	logs.Debug("Auth Check", ok)
	return ok
}

func LoggedInUser(c echo.Context) models.User {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Retrieve the user from the session
	user, _ := sess.Values["user"].(models.User)
	logs.Debug("Logged In User", user.Username)
	return user
}
func BannedCheck(c echo.Context) bool {
	// Retrieve the session
	sess, _ := session.Get("session", c)

	// Check if the user is in the session and not nil
	user, ok := sess.Values["user"].(models.User)
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
	user, ok := sess.Values["user"].(models.User)
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

	var user models.User
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
	if user.ID == 0 {
		logs.Error("User not found")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Update password if provided
	if password != "" {
		user.Password = encryptPassword(password)
	}

	// Save updated user to the database
	if err := database.DB_Main.Collection("users").FindOneAndUpdate(context.TODO(), bson.M{"_id": user.ID}, bson.M{"$set": user}).Err(); err != nil {
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

	user, ok := sess.Values["user"].(models.User)
	if !ok {
		logs.Error("User not found in session")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found in session"})
	}

	password := c.FormValue("password")

	// validate input parameters
	if password == "" {
		logs.Error("Password is required")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password is required"})
	}

	// update user password
	user.Password = encryptPassword(password)

	// save user to database

	if _, err := database.DB_Main.Collection("users").UpdateOne(context.TODO(), bson.M{"_id": user.ID}, bson.M{"$set": user}); err != nil {
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
func GetTotalUsers() int {
	var count int64
	count, err := database.DB_Main.Collection("users").CountDocuments(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Failed to count users: ", err)
		return 0
	}
	// Return the total number of users
	logs.Debug("Total Users: ", count)
	intCount := int(count)
	return intCount
}

func GetUserByID(id uint) models.User {
	// Obtain the database connection from the `database` package

	// Retrieve the user from the database based on the ID
	var user models.User
	database.DB_Main.Collection("users").FindOne(context.TODO(), bson.M{"_id": id}).Decode(&user)

	// Return the user
	logs.Debug("User Found: ", user.Username)
	return user
}

func GetUserByUsername(username string) models.User {

	var user models.User
	database.DB_Main.Collection("users").FindOne(context.TODO(), bson.M{"username": username}).Decode(&user)
	logs.Debug("User Found: ", user.Username)
	return user
}

func ListAdmins() []models.User {
	// Obtain the database connection from the `database` package

	// Retrieve the admins from the database
	var admins []models.User
	collection := database.DB_Main.Collection("users")
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

func ListModerators() []models.User {
	// Obtain the database connection from the `database` package

	// Retrieve the moderators from the database
	var moderators []models.User
	collection := database.DB_Main.Collection("users")
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

func ListJannies() []models.User {
	// Obtain the database connection from the `database` package

	// Retrieve the jannies from the database
	var jannies []models.User
	collection := database.DB_Main.Collection("users")
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

	var users []models.User
	collection := database.DB_Main.Collection("users")
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
				_, err := collection.DeleteOne(context.Background(), bson.M{"_id": user.ID})
				if err != nil {
					logs.Error("Failed to delete user: ", err)
				}
			}
		}
	}
}

func NewPremiumUser(c echo.Context, email string, transactionid string) {
	userID := Getrandid()

	user := models.User{
		ID:       userID,
		Username: strings.Split(email, "@")[0],
		Password: encryptPassword(getrandpassword()),
		Groups: models.Group{
			Admin:     false,
			Moderator: false,
			Janny:     models.JannyBoards{Boards: []string{}},
		},
		DateCreated:   time.Now().Format("2006-01-02 15:04:05"),
		LastLogin:     time.Now().Format("2006-01-02 15:04:05"),
		DoesExist:     true,
		Premium:       true,
		Email:         email,
		TransactionID: transactionid,
	}
	database.DB_Main.Collection("users").InsertOne(context.Background(), user)
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
	user, ok := sess.Values["user"].(models.User)
	if !ok {
		logs.Error("User not found in session")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found in session"})
	}

	// Retrieve the user from the database

	var u models.User
	database.DB_Main.Collection("users").FindOne(context.TODO(), bson.M{"_id": user.ID}).Decode(&u)

	// Delete the user from the database
	if err := database.DB_Main.Collection("users").FindOneAndDelete(context.TODO(), bson.M{"_id": user.ID}).Err(); err != nil {
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

func UserSession(c echo.Context) *models.User {
	sess, _ := session.Get("session", c)

	user, ok := sess.Values["user"].(models.User)
	if !ok {
		logs.Error("User not found in session")
		return nil
	}

	return &user
}
