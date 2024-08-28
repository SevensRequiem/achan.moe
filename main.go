package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"achan.moe/bans"
	"achan.moe/database"
	"achan.moe/home"
	"achan.moe/routes"
	_ "github.com/go-sql-driver/mysql"
)

func init() {

	gob.Register(map[string]interface{}{})

}

func main() {
	e := echo.New()
	e.Use(bans.BanMiddleware)
	database.Init()
	err := godotenv.Load() // Load .env file from the current directory
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	secret := os.Getenv("SECRET")
	fmt.Println(secret)
	if secret == "" {
		log.Fatal("SECRET is not set")
	}
	e.Use(session.Middleware(sessions.NewCookieStore([]byte(secret))))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${id} ${time_rfc3339} ${remote_ip} > ${method} > ${uri} > ${status} ${latency_human}\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("10M"))
	e.Use(middleware.RequestID())
	//e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(7)))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:1556",
			"https://localhost:1556",
			"https://achan.moe/*",
			"https://www.achan.moe",
			"https://www.achan.moe",
			"https://achan.moe:8080",
			"https://www.achan.moe:8080",
			"https://neo.achan.moe/*",
			"https://www.neo.achan.moe",
			"https://cloudflare.com",
			"https://www.cloudflare.com",
			"https://cloudflare.com/*",
			"https://www.cloudflare.com/*",
		},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	e.Use(middleware.Gzip())
	e.Use(middleware.Secure())
	baseUrl := os.Getenv("BASE_URL")
	middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "cookie:_csrf",
		CookiePath:     "/",
		CookieDomain:   baseUrl,
		CookieSecure:   true,
		CookieHTTPOnly: false,
		CookieSameSite: http.SameSiteStrictMode,
	})

	fmt.Println("=====================================")
	//auth.InitDB()
	fmt.Println("=====================================")
	//bans.InitDB()
	//bans.Init()
	fmt.Println("=====================================")
	//discord.DiscordBot()
	fmt.Println("=====================================")
	//backup.BackupDB()

	accesslog, err := os.OpenFile("access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		e.Logger.Fatal(err)
	}
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${remote_ip} - ${id} [${time_rfc3339}] \"${method} ${uri} HTTP/1.1\" ${status} ${bytes_sent}\n",
		Output: accesslog, // Set the Output to the log file
	}))
	e.Renderer = home.NewTemplateRenderer("views/*.html")
	routes.Routes(e)
	e.HTTPErrorHandler = home.ErrorHandler

	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("PORT is not set")
	}

	e.Start(":" + strconv.Itoa(port))
}
