package main

import (
	"encoding/gob"
	"fmt"
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

	"achan.moe/auth"
	"achan.moe/bans"
	"achan.moe/database"
	"achan.moe/home"
	"achan.moe/routes"
	"achan.moe/utils/cache"
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
	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600, // 1 hour
		HttpOnly: true,
		Secure:   false,                // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode, // Adjust according to your needs
	}

	e.Use(session.Middleware(store))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${id} ${time_rfc3339} ${remote_ip} > ${method} > ${uri} > ${status} ${latency_human}\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("11M"))
	e.Use(middleware.RequestID())
	//e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(7)))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"https://www.sandbox.paypal.com",
			"https://www.paypalobjects.com",
		},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
	}))
	baseUrl := os.Getenv("BASE_URL")
	middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "cookie:_csrf",
		CookiePath:     "/",
		CookieDomain:   baseUrl,
		CookieSecure:   true,
		CookieHTTPOnly: true,
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
	s5 := schedule.NewScheduler()
	s5.ScheduleTask(schedule.Task{
		Action: func() {
			minecraft.GetServerStatus()
			bans.ExpireCheck()
		},
		Duration: 5 * time.Minute,
	})
	s5.Run()

	s24h := schedule.NewScheduler()
	s24h.ScheduleTask(schedule.Task{
		Action: func() {
			auth.ExpireUsers()
		},
		Duration: 24 * time.Hour,
	})
	s24h.Run()

	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("PORT is not set")
	}
	//mail.TestMail()
	c := cache.New()

	c.Set("foo", "bar")
	c.Set("baz", "qux")

	fmt.Println("Cache items:")
	items := c.ListAll()
	for k, v := range items {
		fmt.Printf("%s: %s\n", k, v)
	}
	//go tests.Test()
	//go env.RegenEncryptedKey()
	//go env.RegenSecretKey()
	//go plugins.LoadPlugins(e)
	e.StartTLS(":"+strconv.Itoa(port), "certificates/cert.pem", "certificates/key.pem")
}
