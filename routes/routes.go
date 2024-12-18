package routes

import (
	"net/http"

	"achan.moe/admin"
	"achan.moe/auth"
	"achan.moe/banners"
	"achan.moe/bans"
	"achan.moe/board"
	"achan.moe/boardimages"
	"achan.moe/home"
	"achan.moe/user"
	"achan.moe/utils/cache"
	captcha "achan.moe/utils/captcha"
	"achan.moe/utils/config"
	"achan.moe/utils/hitcounter"
	"achan.moe/utils/minecraft"
	"achan.moe/utils/news"
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
		bans.ManualBanIP(c)
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
	e.GET("/image/:b/:i", func(c echo.Context) error {
		imageID := c.Param("i")
		boardID := c.Param("b")
		return boardimages.ReturnImage(c, boardID, imageID)
	})

	e.GET("/thumb/:b/:i", func(c echo.Context) error {
		thumbID := c.Param("i")
		boardID := c.Param("b")
		return boardimages.ReturnThumb(c, boardID, thumbID)
	})
	e.GET("/banner/:b/global", func(c echo.Context) error {
		return banners.GetRandomGlobalBanner(c)
	})

	e.GET("/banner/:b/local", func(c echo.Context) error {
		boardID := c.Param("b")
		return banners.GetRandomLocalBanner(c, boardID)
	})
	e.GET("/banner/:b", func(c echo.Context) error {
		boardID := c.Param("b")
		return banners.GetRandomBanner(c, boardID)
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
	e.POST("/api/ban", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return bans.BanIP(c)
	})

	// statistics
	e.GET("/api/contentsize", func(c echo.Context) error {
		return stats.ReturnContentSizeFromDB(c)
	})
	e.GET("/api/admin/stats", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(401, "Unauthorized")
		}
		return stats.GetStats(c)
	})
	e.GET("/api/minecraft", func(c echo.Context) error {
		return minecraft.JSONStatus(c)
	})

	// latest
	e.GET("/api/latest", func(c echo.Context) error {
		threads := cache.GetLatestThreadsHandler()
		return c.JSON(http.StatusOK, threads)
	})

	// news
	e.GET("/api/news", func(c echo.Context) error {
		news := cache.GetAllNewsHandler()
		return c.JSON(http.StatusOK, news)
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

	e.POST("/post/:b/:t", func(c echo.Context) error {
		board.CreatePost(c)
		return nil
	})

	e.GET("/api/boards", func(c echo.Context) error {
		boards := cache.GetBoardsHandler()
		return c.JSON(http.StatusOK, boards)
	})

	e.GET("/api/board/:b", func(c echo.Context) error {
		board := cache.GetBoardHandler(c.Param("b"))
		return c.JSON(http.StatusOK, board)
	})

	e.GET("/api/board/:b/threads", func(c echo.Context) error {
		threads := cache.GetThreadsHandler(c, c.Param("b"))
		return c.JSON(http.StatusOK, threads)
	})

	e.GET("/api/board/:b/thread/:t", func(c echo.Context) error {
		thread := cache.GetThreadHandler(c, c.Param("b"), c.Param("t"))
		return c.JSON(http.StatusOK, thread)
	})

	e.GET("/test/thumb/:b/:t", func(c echo.Context) error {
		thumbnail := cache.GetThumbnail(c.Param("b"), c.Param("t"))
		return c.JSON(http.StatusOK, thumbnail)
	})

	e.DELETE("/board/:b/:t", func(c echo.Context) error {
		boardID := c.Param("b")
		threadID := c.Param("t")
		if !(auth.AdminCheck(c) || auth.ModeratorCheck(c) || auth.JannyCheck(c, boardID)) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeleteThread(c, boardID, threadID)
		return nil
	})

	e.DELETE("/board/:b/:t/:p", func(c echo.Context) error {
		boardID := c.Param("b")
		threadID := c.Param("t")
		postID := c.Param("p")

		if !(auth.AdminCheck(c) || auth.ModeratorCheck(c) || auth.JannyCheck(c, boardID)) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeletePost(c, boardID, threadID, postID)
		return nil
	})

	e.POST("/board/:b/report", func(c echo.Context) error {
		board.ReportPost(c)
		return nil
	})

	e.POST("/admin/banners", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return banners.UploadBanner(c)
	})
	e.GET("/admin/banners", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminBannersHandler(c)
	})
	user.Routes(e)
}
