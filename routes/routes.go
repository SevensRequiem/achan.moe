package routes

import (
	"fmt"
	"net/http"

	"achan.moe/admin"
	"achan.moe/auth"
	"achan.moe/bans"
	"achan.moe/board"
	"achan.moe/home"
	captcha "achan.moe/utils/captcha"
	"achan.moe/utils/config"
	"achan.moe/utils/hitcounter"
	"achan.moe/utils/minecraft"
	"achan.moe/utils/stats"
	"achan.moe/utils/websocket"
	"github.com/labstack/echo/v4"
)

func Routes(e *echo.Echo) {
	hc := hitcounter.NewHitCounter()

	e.GET("/", func(c echo.Context) error {
		hc.Hit(c.RealIP())
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
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminHandler(c)
	})

	e.GET("/admin/dashboard", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminDashboardHandler(c)
	})

	e.GET("/admin/boards", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminBoardsHandler(c)
	})

	e.GET("/admin/users", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminUsersHandler(c)
	})

	e.POST("/admin/user/edit", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return auth.EditUser(c)
	})

	e.GET("/admin/config", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminConfigHandler(c)
	})

	e.POST("/admin/config", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return config.WriteGlobalConfig(c)
	})

	e.GET("/admin/bans", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminBansHandler(c)
	})

	e.GET("/admin/update", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminUpdateHandler(c)
	})

	e.POST("/admin/board", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.CreateBoard(c)
		return nil
	})

	e.POST("/admin/user", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.UpdateUserRole(c)
		return nil
	})

	e.DELETE("/admin/delete/:b", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeleteBoard(c)
		return nil
	})

	e.DELETE("/admin/delete/:b/:t", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeleteThread(c)
		return nil
	})

	e.DELETE("/admin/delete/:b/:t/:p", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeletePost(c)
		return nil
	})

	e.POST("/admin/ban", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		bans.BanIP(c)
		return nil
	})
	e.POST("/admin/unban", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		bans.UnbanIP(c)
		return nil
	})
	// moderator
	e.POST("/mod/ban", func(c echo.Context) error {
		if !auth.ModCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		bans.BanIP(c)
		return nil
	})

	e.POST("/mod/delete/:b/:t", func(c echo.Context) error {
		if !auth.ModCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeleteThread(c)
		return nil
	})

	e.POST("/mod/delete/:b/:t/:p", func(c echo.Context) error {
		if !auth.ModCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeletePost(c)
		return nil
	})
	// janny
	e.DELETE("/janny/delete/:b/:t", func(c echo.Context) error {
		board := c.Param("b")
		if !auth.JannyCheck(c, board) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.JannyDeleteThread(c)
		return nil
	})

	e.DELETE("/janny/delete/:b/:t/:p", func(c echo.Context) error {
		board := c.Param("b")
		if !auth.JannyCheck(c, board) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
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
	e.GET("/thumb/:f", func(c echo.Context) error {
		file := c.Param("f")
		filepath := fmt.Sprintf("thumbs/%s", file)
		return c.File(filepath)
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
	e.POST("/login", func(c echo.Context) error {
		return auth.LoginHandler(c)
	})
	e.GET("/logout", func(c echo.Context) error {
		return auth.LogoutHandler(c)
	})

	e.GET("/register", func(c echo.Context) error {
		return home.RegisterHandler(c)
	})
	e.POST("/register", func(c echo.Context) error {
		return auth.NewUser(c)
	})

	// bans
	e.GET("/api/bans", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return bans.GetBans(c)
	})
	e.GET("/api/bans/old", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return bans.GetBansOld(c)
	})
	e.GET("/api/bans/active", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return bans.GetBansActive(c)
	})
	e.GET("/api/bans/expired", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return bans.GetBansExpired(c)
	})
	e.GET("/api/bans/deleted", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return bans.GetBansDeleted(c)
	})

	// statistics
	e.GET("/api/admin/stats", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(401, "Unauthorized")
		}
		return stats.GetStats(c)
	})
	e.GET("/api/minecraft", func(c echo.Context) error {
		return minecraft.JSONStatus(c)
	})

	// captcha
	e.GET("/api/gencaptcha", func(c echo.Context) error {
		return captcha.GenerateCaptchaHandler(c)
	})
	e.POST("/api/verifycaptcha", func(c echo.Context) error {
		return captcha.VerifyCaptchaHandler(c)
	})

	// server status
	e.GET("/api/status", func(c echo.Context) error {
		return stats.ServerStatus(c)
	})

	// websocket
	e.GET("/ws", func(c echo.Context) error {
		return websocket.WebsocketHandler(c)
	})

	// store
	e.GET("/store", func(c echo.Context) error {
		return home.StoreHandler(c)
	})

	// profile
	e.GET("/profile", func(c echo.Context) error {
		return home.ProfileHandler(c)
	})

	e.POST("/profile/edit", func(c echo.Context) error {
		return auth.UpdateUser(c)
	})

}
