package routes

import (
	"fmt"

	"achan.moe/admin"
	"achan.moe/auth"
	"achan.moe/bans"
	"achan.moe/board"
	"achan.moe/home"
	"github.com/labstack/echo/v4"
)

func Routes(e *echo.Echo) {
	e.GET("/", func(c echo.Context) error {
		return home.HomeHandler(c)
	})

	e.GET("/terms", func(c echo.Context) error {
		return home.TermsHandler(c)
	})
	e.GET("/privacy", func(c echo.Context) error {
		return home.PrivacyHandler(c)
	})
	e.GET("/contact", func(c echo.Context) error {
		return home.ContactHandler(c)
	})
	e.GET("/donate", func(c echo.Context) error {
		return home.DonateHandler(c)
	})

	e.GET("/board/:b", func(c echo.Context) error {
		return home.BoardHandler(c)
	})

	e.GET("/board/:b/:t", func(c echo.Context) error {
		return home.ThreadHandler(c)
	})

	e.GET("/board/:b/:t/:p", func(c echo.Context) error {
		return home.PostHandler(c)
	})

	e.POST("/board/:b", func(c echo.Context) error {
		board.CreateThread(c)
		return nil
	})

	e.POST("/board/:b/:t", func(c echo.Context) error {
		board.CreateThreadPost(c)
		return nil
	})

	// admin

	e.GET("/admin", func(c echo.Context) error {
		return home.AdminHandler(c)
	})

	e.POST("/admin/board", func(c echo.Context) error {
		admin.CreateBoard(c)
		return nil
	})

	e.DELETE("/admin/delete/:b", func(c echo.Context) error {
		admin.DeleteBoard(c)
		return nil
	})

	e.DELETE("/admin/delete/:b/:t", func(c echo.Context) error {
		admin.DeleteThread(c)
		return nil
	})

	e.DELETE("/admin/delete/:b/:t/:p", func(c echo.Context) error {
		admin.DeletePost(c)
		return nil
	})

	e.POST("/admin/ban", func(c echo.Context) error {
		bans.BanIP(c)
		return nil
	})
	// janny
	e.POST("/janny/delete/:b/:t", func(c echo.Context) error {
		admin.JannyDeleteThread(c)
		return nil
	})

	e.POST("/janny/delete/:b/:t/:p", func(c echo.Context) error {
		admin.JannyDeletePost(c)
		return nil
	})

	// static files
	e.Static("/assets", "assets")
	e.GET("/robots.txt", func(c echo.Context) error {
		return c.File("static/robots.txt")
	})
	e.GET("/sitemap.xml", func(c echo.Context) error {
		return c.File("static/sitemap.xml")
	})
	e.GET("/file/:b/:f", func(c echo.Context) error {
		board := c.Param("b")
		file := c.Param("f")
		// Construct the path to the file based on the parameters
		filePath := fmt.Sprintf("boards/%s/%s", board, file)
		// Serve the file
		return c.File(filePath)
	})
	e.GET("/banner/:b/:f", func(c echo.Context) error {
		board := c.Param("b")
		file := c.Param("f")
		filepath := fmt.Sprintf("boards/%s/banners/%s", board, file)
		return c.File(filepath)
	})

	//auth
	e.GET("/login", func(c echo.Context) error {
		return auth.LoginHandler(c)
	})
	e.GET("/auth/callback", func(c echo.Context) error {
		return auth.CallbackHandler(c)
	})
	e.GET("/logout", func(c echo.Context) error {
		return auth.LogoutHandler(c)
	})
}