package routes

import (
	"fmt"
	"net/http"
	"time"

	"achan.moe/admin"
	"achan.moe/auth"
	"achan.moe/banners"
	"achan.moe/bans"
	"achan.moe/board"
	"achan.moe/home"
	"achan.moe/user"
	captcha "achan.moe/utils/captcha"
	"achan.moe/utils/config"
	"achan.moe/utils/hitcounter"
	"achan.moe/utils/minecraft"
	"achan.moe/utils/news"
	"achan.moe/utils/stats"
	"achan.moe/utils/websocket"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"
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
	e.GET("/admin/news", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminNewsHandler(c)
	})
	e.POST("/admin/addnews", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		news.NewNews(c)
		return nil
	})
	e.DELETE("/admin/delete/:b", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeleteBoard(c)
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
		filePath := fmt.Sprintf("boards/%s/%s", board, file)
		return c.File(filePath)
	})
	e.GET("/thumb/:f", func(c echo.Context) error {
		file := c.Param("f")
		filepath := fmt.Sprintf("thumbs/%s", file)
		return c.File(filepath)
	})
	e.GET("/banner/:b", func(c echo.Context) error {
		boardid := c.Param("b")
		rand.Seed(uint64(time.Now().UnixNano()))
		var banner string
		var err error

		if rand.Intn(2) == 1 {
			banner, err = banners.GetRandomGlobalBanner()
		} else {
			banner, err = banners.GetRandomLocalBanner(boardid)
		}

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		return c.File(banner)
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
		board.CreateThread(c)
		return nil
	})

	e.POST("/board/:b/:t/:p", func(c echo.Context) error {
		board.CreatePost(c)
		return nil
	})

	e.DELETE("/board/:b/:t", func(c echo.Context) error {
		boardID := c.Param("b")
		if !(auth.AdminCheck(c) || auth.ModeratorCheck(c) || auth.JannyCheck(c, boardID)) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		board.DeleteThread(c)
		return nil
	})

	e.DELETE("/board/:b/:t/:p", func(c echo.Context) error {
		boardID := c.Param("b")
		if !(auth.AdminCheck(c) || auth.ModeratorCheck(c) || auth.JannyCheck(c, boardID)) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		board.DeletePost(c)
		return nil
	})

	e.POST("/board/:b/:t/:p/report", func(c echo.Context) error {
		board.ReportPost(c)
		return nil
	})

	e.POST("/board/:b/:t/report", func(c echo.Context) error {
		board.ReportThread(c)
		return nil
	})

	e.POST("/board/:b/:t/:p/delete", func(c echo.Context) error {
		boardID := c.Param("b")
		if !(auth.AdminCheck(c) || auth.ModeratorCheck(c) || auth.JannyCheck(c, boardID)) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		board.DeletePost(c)
		return nil
	})
	user.Routes(e)
}
