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
	"achan.moe/utils/actions"
	"achan.moe/utils/announcements"
	"achan.moe/utils/cache"
	captcha "achan.moe/utils/captcha"
	"achan.moe/utils/hitcounter"
	"achan.moe/utils/news"
	"achan.moe/utils/ratelimit"
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
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminHandler(c)
	})

	e.GET("/admin/dashboard", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminDashboardHandler(c)
	})

	e.GET("/admin/boards", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminBoardsHandler(c)
	})

	e.GET("/admin/users", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminUsersHandler(c)
	})

	e.POST("/admin/user/edit", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return auth.EditUser(c)
	})

	e.GET("/admin/config", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminConfigHandler(c)
	})

	e.GET("/admin/bans", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminBansHandler(c)
	})

	e.GET("/admin/update", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminUpdateHandler(c)
	})

	e.POST("/admin/board", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.CreateBoard(c)
		return nil
	})
	e.GET("/admin/news", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminNewsHandler(c)
	})
	e.POST("/admin/addnews", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		news.NewNews(c)
		return nil
	})
	e.DELETE("/admin/delete/:b", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		admin.DeleteBoard(c)
		return nil
	})

	e.POST("/admin/ban", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		bans.ManualBanIP(c)
		return nil
	})
	e.POST("/admin/unban", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		bans.UnbanIP(c)
		return nil
	})

	e.POST("/admin/announcement", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return announcements.AddAnnouncement(c)
	})
	e.GET("/admin/announcement", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminAnnouncementsHandler(c)
	})

	e.GET("/admin/actions", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminActionsHandler(c)
	})
	e.GET("/admin/reports", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminReportsHandler(c)
	})
	e.GET("/admin/update", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return home.AdminUpdateHandler(c)
	})

	e.DELETE("/admin/announcement", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return announcements.DeleteAnnouncement(c)
	})

	e.DELETE("/admin/news", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return news.DeleteNews(c)
	})

	e.DELETE("/admin/banner", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return banners.DeleteBannerHandler(c)
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
	e.GET("/api/banner/global/:id", func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, "Missing ID")
		}
		return banners.GetGlobalBanner(c, id)
	})

	e.GET("/api/banner/local/:b/:id", func(c echo.Context) error {
		boardID := c.Param("b")
		id := c.Param("id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, "Missing ID")
		}
		if boardID == "" {
			return c.JSON(http.StatusBadRequest, "Missing board ID")
		}
		return banners.GetLocalBanner(c, boardID, id)
	})
	e.GET("/banner/:b", func(c echo.Context) error {
		boardID := c.Param("b")
		return banners.GetRandomBanner(c, boardID)
	})

	e.GET("/api/banners/global", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return banners.ListGlobalBanners(c)
	})
	e.GET("/api/banners/local/:boardid", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		return banners.ListLocalBanners(c)
	})

	e.DELETE("/api/banner/global/:id", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		id := c.Param("id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, "Missing ID")
		}
		return banners.DeleteGlobalBanner(c, id)
	})
	e.DELETE("/api/banner/local/:b/:id", func(c echo.Context) error {
		if !auth.AdminCheck(c) {
			return c.JSON(http.StatusUnauthorized, "Unauthorized")
		}
		boardID := c.Param("b")
		id := c.Param("id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, "Missing ID")
		}
		if boardID == "" {
			return c.JSON(http.StatusBadRequest, "Missing board ID")
		}
		return banners.DeleteLocalBanner(c, boardID, id)
	})
	//auth
	e.GET("/login", func(c echo.Context) error {
		return auth.LoginHandler(c)
	})
	e.POST("/login", func(c echo.Context) error {
		exceeded, err := ratelimit.LoginHandler(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "Internal server error")
		}
		if exceeded {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "You are being rate limited. Please try again later.",
			})
		}
		return auth.LoginHandler(c)
	})
	e.GET("/logout", func(c echo.Context) error {
		return auth.LogoutHandler(c)
	})

	e.GET("/register", func(c echo.Context) error {
		if auth.AuthCheck(c) {
			return c.Redirect(http.StatusSeeOther, "/")
		}
		return home.RegisterHandler(c)
	})
	e.POST("/register", func(c echo.Context) error {
		exceeded, err := ratelimit.RegisterHandler(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "Internal server error")
		}
		if exceeded {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "You are being rate limited. Please try again later.",
			})
		}
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
	// latest
	e.GET("/api/latest", func(c echo.Context) error {
		threads := cache.GetLatestThreadsHandler()
		return c.JSON(http.StatusOK, threads)
	})

	// news
	e.GET("/api/news", func(c echo.Context) error {
		news := cache.GetAllNewsHandler(c)
		return c.JSON(http.StatusOK, news)
	})
	// captcha
	e.GET("/api/gencaptcha", func(c echo.Context) error {
		exceeded, err := ratelimit.CaptchaHandler(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "Internal server error")
		}
		if exceeded {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "You are being rate limited. Please try again later.",
			})
		}
		return captcha.GetCaptcha(c)
	})

	e.GET("/api/debug/captcha", func(c echo.Context) error {
		return captcha.GetCurrentCaptcha(c)
	})

	// server status

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
		exceeded, err := ratelimit.ThreadHandler(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "Internal server error")
		}
		if exceeded {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "You are being rate limited. Please try again later.",
			})
		}
		board.CreateThread(c)
		return nil
	})

	e.POST("/post/:b/:t", func(c echo.Context) error {
		exceeded, err := ratelimit.PostHandler(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "Internal server error")
		}
		if exceeded {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "You are being rate limited. Please try again later.",
			})
		}
		board.CreatePost(c)
		return nil
	})

	e.GET("/api/boards", func(c echo.Context) error {
		boards := cache.GetBoards()
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
	actions.Routes(e)
}
