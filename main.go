package main

import (
	"encoding/gob"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"achan.moe/bans"
	"achan.moe/database"
	"achan.moe/home"
	"achan.moe/routes"
	"achan.moe/utils/minecraft"
	"achan.moe/utils/schedule"
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
	if secret == "" {
		log.Fatal("SECRET is not set")
	}
	e.Use(session.Middleware(sessions.NewCookieStore([]byte(secret))))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${id} ${time_rfc3339} ${remote_ip} > ${method} > ${uri} > ${status} ${latency_human}\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("11M"))
	e.Use(middleware.RequestID())
	//e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(7)))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:1313",
			"https://localhost:1313",
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
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
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

	// schedules
	scheduler := schedule.NewScheduler()
	scheduler.ScheduleTask(schedule.Task{
		Action: func() {
			bans.ExpireCheck()
			minecraft.GetServerStatus()
		},
		Duration: 5 * time.Minute,
	})
	scheduler.Run()

	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("PORT is not set")
	}

	e.Start(":" + strconv.Itoa(port))
}
