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
		bans.BanIP(c, c.FormValue("ip"), c.FormValue("reason"), c.FormValue("username"), c.FormValue("admin"), c.FormValue("timestamp"))
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
	e.GET("/img/:b/:f", func(c echo.Context) error {
		board := c.Param("b")
		file := c.Param("f")
		// Construct the path to the file based on the parameters
		filePath := fmt.Sprintf("boards/%s/%s", board, file)
		// Serve the file
		return c.File(filePath)
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
