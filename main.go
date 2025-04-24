package main

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"achan.moe/auth"
	"achan.moe/bans"
	"achan.moe/board"
	"achan.moe/database"
	"achan.moe/home"
	"achan.moe/logs"
	"achan.moe/routes"
	"achan.moe/utils/cache"
	"achan.moe/utils/schedule"
	"achan.moe/utils/stats"
)

func init() {
	gob.Register(map[string]interface{}{})
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		fmt.Println("Dropping collections")
		database.Drops()
		fmt.Println("Migrating bans")
		database.Migratebansfromsql()
		fmt.Println("Migrating boards")
		database.Migrateboardsfromsql()
		fmt.Println("Migrating threads")
		board.MigrateToMongoFromGob()
		fmt.Println("Migrating misc")
		database.Migratemisc()
		fmt.Println("finish")
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "reset" {
		fmt.Println("Dropping collections")
		database.Drops()
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "process" {
		fmt.Println("Generating HTML + CSS")
		processHTMLCSS()
		os.Exit(0)
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	stats.CalcTotalSize()
	cache.On = true
	cache.InitCaches()

	e := echo.New()

	e.Use(bans.BanMiddleware)
	err := godotenv.Load()
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
	e.Use(middleware.BodyLimit("10M"))
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
	e.Renderer = home.NewTemplateRenderer("views/dst/*.html")
	routes.Routes(e)
	e.HTTPErrorHandler = home.ErrorHandler
	// schedules
	s5 := schedule.NewScheduler()
	s5.ScheduleTask(schedule.Task{
		Action: func() {
			bans.ExpireCheck()
			stats.SetTotalSize(stats.CalcTotalSize())
			stats.SetTotalUsers()
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

	logs.Info("Server started on port " + portStr)

	e.StartTLS(":"+strconv.Itoa(port), "certificates/cert.pem", "certificates/key.pem")
}

func processHTMLCSS() {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)

	srcDir := "views/src"
	destDir := "views/dst"

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)
		err = os.MkdirAll(filepath.Dir(destPath), os.ModePerm)
		if err != nil {
			return err
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		err = m.Minify("text/html", destFile, srcFile)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logs.Error("Error processing HTML/CSS: ", err)
	} else {
		logs.Info("HTML/CSS processing completed")
	}
}
