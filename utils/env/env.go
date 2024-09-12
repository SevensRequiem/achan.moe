package env

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"achan.moe/auth"
	"achan.moe/database"
	"github.com/joho/godotenv"
)

func GetEnv(key string) string {
	return os.Getenv(key)
}
func RegenEncryptedKey() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}

	dotenvPath := dir + "/.env"
	err = godotenv.Load(dotenvPath)
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	db := database.DB
	length, err := strconv.Atoi(os.Getenv("ENCRYPT_KEY_LENGTH"))
	if err != nil {
		log.Fatalf("Invalid ENCRYPT_KEY_LENGTH: %v", err)
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}
	chars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	for i := 0; i < length; i++ {
		b[i] = chars[b[i]%byte(len(chars))]
	}

	newKey := string(b)

	oldKey := os.Getenv("ENCRYPT_KEY")
	if oldKey == "" {
		log.Fatalf("Old ENCRYPT_KEY environment variable is not set")
	}

	os.Setenv("ENCRYPT_KEY", newKey)

	// Update .env file with the new key
	updateEnvFile(dotenvPath, "ENCRYPT_KEY", newKey)

	var users []auth.User
	if err := db.Raw("SELECT uuid, password FROM users").Scan(&users).Error; err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}

	fmt.Println("Regenerating password hashes")
	for _, user := range users {
		// Decrypt password using old key
		decrypted := decodePassword(user.Password, oldKey)

		// Encrypt password using new key
		encrypted := encryptPassword(decrypted, newKey)
		fmt.Println("Updating password for", user.UUID)

		if err := db.Exec("UPDATE users SET password = ? WHERE uuid = ?", encrypted, user.UUID).Error; err != nil {
			log.Fatalf("Failed to update user password: %v", err)
		}
	}
	fmt.Println("Done regenerating password hashes")
}

// Helper function to update the .env file
func updateEnvFile(filepath, key, value string) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Error reading .env file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var updatedLines []string
	keyFound := false

	for _, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			updatedLines = append(updatedLines, key+"="+value)
			keyFound = true
		} else {
			updatedLines = append(updatedLines, line)
		}
	}

	if !keyFound {
		updatedLines = append(updatedLines, key+"="+value)
	}

	err = ioutil.WriteFile(filepath, []byte(strings.Join(updatedLines, "\n")), 0644)
	if err != nil {
		log.Fatalf("Error writing to .env file: %v", err)
	}
}

// Decode the encrypted password using the provided key
func decodePassword(encryptedPassword, key string) string {
	decoded, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		log.Println("Failed to decode password:", err)
		return ""
	}

	h := hmac.New(sha256.New, []byte(key))
	h.Write(decoded)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Encrypt the password using the provided key
func encryptPassword(password, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(password))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func RegenSecretKey() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}

	dotenvPath := dir + "/.env"
	err = godotenv.Load(dotenvPath)
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	length, err := strconv.Atoi(os.Getenv("SECRET_LENGTH"))
	if err != nil {
		log.Fatalf("Invalid SECRET_LENGTH: %v", err)
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}
	chars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	for i := 0; i < length; i++ {
		b[i] = chars[b[i]%byte(len(chars))]
	}

	newKey := string(b)

	oldKey := os.Getenv("SECRET")
	if oldKey == "" {
		log.Fatalf("Old SECRET environment variable is not set")
	}

	os.Setenv("SECRET", newKey)

	// Update .env file with the new key
	updateEnvFile(dotenvPath, "SECRET", newKey)
}
