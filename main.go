package main

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"achan.moe/auth"
	"achan.moe/bans"
	"achan.moe/database"
	"achan.moe/home"
	"achan.moe/logs"
	"achan.moe/routes"
	"achan.moe/utils/minecraft"
	"achan.moe/utils/queue"
	"achan.moe/utils/schedule"
)

func init() {

	gob.Register(map[string]interface{}{})

}

func main() {

	e := echo.New()
	e.Use(bans.BanMiddleware)
	err := godotenv.Load() // Load .env file from the current directory
	if err != nil {
		logs.Fatal("Error loading .env file")
	}
	secret := os.Getenv("SECRET")
	if secret == "" {
		logs.Fatal("SECRET is not set")
	}
	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	e.Use(session.Middleware(store))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${id} ${time_rfc3339} ${remote_ip} > ${method} > ${uri} > ${status} ${latency_human}\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("11M"))
	e.Use(middleware.RequestID())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"https://achan.moe",
		},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://img.shields.io https://placehold.co https://static.cloudflareinsights.com https://achan.moe; connect-src 'self'; font-src 'self'; frame-src 'self'; object-src 'none'; media-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self';",
		HSTSMaxAge:            31536000,
		HSTSExcludeSubdomains: true,
		HSTSPreloadEnabled:    true,
		ReferrerPolicy:        "same-origin",
	}))
	baseUrl := os.Getenv("BASE_URL")
	middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "cookie:_csrf",
		CookieName:     "_csrf",
		CookieMaxAge:   86400,
		CookiePath:     "/",
		CookieDomain:   baseUrl,
		CookieSecure:   true,
		CookieHTTPOnly: false,
		CookieSameSite: http.SameSiteStrictMode,
	})

	accesslog, err := os.OpenFile("access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		logs.Fatal("Failed to open access.log")
	}
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${remote_ip} - ${id} [${time_rfc3339}] \"${method} ${uri} HTTP/1.1\" ${status} ${bytes_sent}\n",
		Output: accesslog, // Set the Output to the log file
	}))
	e.Renderer = home.NewTemplateRenderer("views/*.html")
	routes.Routes(e)
	e.HTTPErrorHandler = home.ErrorHandler

	// schedules
	s5 := schedule.NewScheduler()
	s5.ScheduleTask(schedule.Task{
		Action: func() {
			minecraft.GetServerStatus()
			bans.ExpireCheck()
		},
		Duration: 5 * time.Minute,
	})
	go s5.Run()

	s24h := schedule.NewScheduler()
	s24h.ScheduleTask(schedule.Task{
		Action: func() {
			auth.ExpireUsers()
		},
		Duration: 24 * time.Hour,
	})
	go s24h.Run()

	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logs.Fatal("PORT is not set")
	}
	//mail.TestMail()
	//go env.RegenEncryptedKey()
	//go env.RegenSecretKey()
	//go plugins.LoadPlugins(e)
	queue.NewQueueManager().ProcessAll()
	logs.Info("Server started on port " + portStr)
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		fmt.Println("Dropping collections")
		database.Drops()
		fmt.Println("Migrating bans")
		database.Migratebansfromsql()
		fmt.Println("Migrating boards")
		database.Migrateboardsfromsql()
		fmt.Println("finish")
	}
	if len(os.Args) > 1 && os.Args[1] == "reset" {
		fmt.Println("Dropping collections")
		database.Drops()
	}
	//s := souin_echo.NewMiddleware(souin_echo.DefaultConfiguration)
	//e.Use(s.Process)
	e.StartTLS(":"+strconv.Itoa(port), "certificates/cert.pem", "certificates/key.pem")
}
