package home

import (
	"fmt"
	"io"
	"log"
	"strconv"

	"html/template"
	"net/http"

	"achan.moe/auth"
	"achan.moe/banners"
	"achan.moe/bans"
	"achan.moe/board"
	"achan.moe/utils"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

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
func HomeHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/base.html", "views/home.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Home"

	data["PostCount"] = board.GetGlobalPostCount()
	data["UserCount"] = auth.GetTotalUsers()
	latestPosts, err := board.GetLatestPosts(10)
	if err != nil {
		// Handle the error, for example, log it or return it
		log.Fatalf("Error fetching latest posts: %v", err)
	}
	data["LatestPosts"] = latestPosts

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func BoardHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/base.html", "views/board.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	boardname := board.GetBoardName(c.Param("b"))
	data["Pagename"] = boardname
	data["Board"] = board.GetBoard(c.Param("b"))
	data["BoardID"] = board.GetBoardID(c.Param("b"))
	data["BoardDesc"] = board.GetBoardDescription(c.Param("b"))
	boardid := board.GetBoardID(c.Param("b"))
	data["Threads"] = board.GetThreads(boardid)
	data["IsJanny"] = auth.JannyCheck(c, boardid)
	data["Banner"] = banners.GetRandomBanner(boardid)

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func ThreadHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/base.html", "views/thread.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	// Assuming globaldata is a function that populates some global data for the template
	data = globaldata(c)
	t := c.Param("t") // Get the parameter as string
	b := c.Param("b") // Get the parameter as string

	// Convert t from string to int
	tInt, err := strconv.Atoi(t)
	if err != nil {
		// Handle the error, maybe return or log it
		return c.String(http.StatusBadRequest, "Invalid thread ID")
	}

	// Assuming you want to display posts in the thread, and each post has a Title field
	data["Thread"] = board.GetThread(b, tInt)
	data["ThreadID"] = tInt
	data["BoardID"] = board.GetBoardID(b)
	data["BoardDesc"] = board.GetBoardDescription(b)
	posts := board.GetPosts(b, tInt)
	data["Posts"] = posts
	data["IsJanny"] = auth.JannyCheck(c, b)
	boardid := board.GetBoardID(c.Param("b"))
	data["Banner"] = banners.GetRandomBanner(boardid)

	// Execute the template once with all the data prepared
	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return nil
}
func PostHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/base.html", "views/post.html")
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
		c.Logger().Error(err)
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

	user, ok := userSessionValue.(auth.User)
	if !ok {
		data["User"] = ""
	}
	data["IsAdmin"] = auth.AdminCheck(c)
	data["User"] = user.Username
	data["Boards"] = board.GetBoards()
	data["IP"] = c.RealIP()
	data["Country"] = c.Request().Header.Get("CF-IPCountry")
	data["user"] = "Anonymous"
	data["TotalSize"] = utils.GetProjectSize(".")
	latestPosts, err := board.GetLatestPosts(1)
	if err != nil {
		// Handle the error, for example, log it or return it
		log.Fatalf("Error fetching latest posts: %v", err)
	}
	data["LatestPosts"] = latestPosts
	return data
}

func AdminHandler(c echo.Context) error {
	if !auth.AdminCheck(c) {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}
	tmpl, err := template.ParseFiles("views/base.html", "views/admin/admin.html")
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	data := map[string]interface{}{}
	data = globaldata(c)
	data["Pagename"] = "Admin"
	data["Boards"] = board.GetBoards()
	data["Bans"] = bans.GetBans(c)

	err = tmpl.ExecuteTemplate(c.Response().Writer, "base.html", data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func DonateHandler(c echo.Context) error {
	tmpl, err := template.ParseFiles("views/base.html", "views/donate.html")
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
	tmpl, err := template.ParseFiles("views/base.html", "views/terms.html")
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
	tmpl, err := template.ParseFiles("views/base.html", "views/privacy.html")
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
	tmpl, err := template.ParseFiles("views/base.html", "views/contact.html")
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