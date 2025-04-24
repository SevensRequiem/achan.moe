package home

import (
	"fmt"
	"io"
	"os"

	"html/template"
	"net/http"

	"achan.moe/auth"
	"achan.moe/models"
	"achan.moe/utils/cache"
	config "achan.moe/utils/config"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

var baseUrl = ""

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func NewTemplateRenderer(glob string) *TemplateRenderer {
	tmpl := template.Must(template.ParseGlob(glob))
	return &TemplateRenderer{
		templates: tmpl,
	}
}

func init() {
	godotenv.Load()
	baseUrl = os.Getenv("BASE_URL")
}

func HomeHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/home.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Home"
	data["Hits"] = cache.GetTotalHitCount()
	data["News"] = cache.GetAllNewsHandler(c)

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func BoardHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/board.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	board := cache.GetBoard(c.Param("b"))
	data["Pagename"] = board.Name
	data["Board"] = board
	data["BoardID"] = board.BoardID
	description := board.Description
	data["BoardDesc"] = description
	boardid := board.BoardID
	data["IsJanny"] = auth.JannyCheck(c, boardid)

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func ThreadHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/thread.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	t := c.Param("t")
	b := c.Param("b")

	session, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	selfposts := session.Values["self_post_id"]

	data["ThreadID"] = t
	data["BoardID"] = cache.GetBoard(b).BoardID
	description := cache.GetBoard(b).Description
	data["BoardDesc"] = description
	data["IsJanny"] = auth.JannyCheck(c, b)
	data["SelfPosts"] = selfposts

	// Execute the template once with all the data prepared
	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return nil
}
func PostHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/post.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Post"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func ErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	data := map[string]interface{}{
		"code": code,
	}

	if err := c.Render(code, "error.html", data); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
}
func globaldata(c echo.Context) map[string]interface{} {
	data := map[string]interface{}{}
	sess, err := session.Get("session", c)
	if err != nil {
		return nil
	}

	userSessionValue, ok := sess.Values["user"]
	if !ok {
		data["User"] = ""
	}

	user, ok := userSessionValue.(models.User)
	if !ok {
		data["User"] = ""
	}

	data["IsAdmin"] = auth.AdminCheck(c)
	data["IsModerator"] = auth.ModeratorCheck(c)
	data["User"] = user
	data["Boards"] = cache.GetBoards()
	data["IP"] = c.RealIP()
	data["Country"] = c.Request().Header.Get("CF-IPCountry")
	data["user"] = "Anonymous"
	data["PostCount"] = cache.GetGlobalPostCount()
	data["TotalSize"] = cache.GetTotalSize()
	data["TotalUsers"] = cache.GetTotalUserCount()
	data["GlobalConfig"] = config.GetGlobalConfig()
	data["Country"] = c.Request().Header.Get("CF-IPCountry")
	var globalannouncement string
	announcement := cache.GetGlobalAnnouncement()
	if announcement.Content == "" {
		globalannouncement = ""
	} else {
		globalannouncement = announcement.Content
	}
	data["GlobalAnnouncement"] = globalannouncement
	data["BaseURL"] = baseUrl
	return data
}
func RegisterHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/register.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Register"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return nil
}
func AdminHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Admin"
	data["Boards"] = models.GetBoards()

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func AdminDashboardHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/dashboard.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Dashboard"
	boards := models.GetBoards()

	data["Boards"] = boards
	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func AdminBoardsHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/boards.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Boards"
	data["Boards"] = models.GetBoards()
	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func AdminUsersHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/users.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Users"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func AdminConfigHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/config.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Config"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func AdminBansHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/bans.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Bans"
	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func AdminUpdateHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/update.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Update"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func AdminNewsHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/news.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin News"
	data["News"] = cache.GetAllNewsHandler(c)

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return nil
}

func AdminBannersHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/banners.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Banners"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return nil
}
func AdminReportsHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/reports.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Reports"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}
func AdminAnnouncementsHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/announcements.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Announcements"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}
func AdminActionsHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/admin/admin.html", "views/dst/admin/actions.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	data := globaldata(c)
	data["Pagename"] = "Admin Actions"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}
func DonateHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/donate.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Donate"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func TermsHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/terms.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Terms"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func PrivacyHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/privacy.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Privacy"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func ContactHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/contact.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Contact"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func StoreHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/store.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Store"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func SuccessHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/success.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Success"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func ProfileHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/dst/base.html", "views/dst/profile.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Profile"

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}
