package captcha

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Captcha struct {
	Text  string `json:"text"`
	Image string `json:"image"`
}

var (
	CaptchaWidth  = 200
	CaptchaHeight = 100
	CaptchaLength = 6
	CaptchaChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	fontFace      font.Face
)

func init() {
	fontBytes, err := ioutil.ReadFile("assets/fonts/TerminalVector.ttf")
	if err != nil {
		panic(err)
	}
	ttf, err := opentype.Parse(fontBytes)
	if err != nil {
		panic(err)
	}
	const fontSize = 36
	fontFace, err = opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
}

func GenerateCaptcha() (string, *bytes.Buffer) {
	rand.Seed(time.Now().UnixNano())
	img := image.NewRGBA(image.Rect(0, 0, CaptchaWidth, CaptchaHeight))
	for x := 0; x < CaptchaWidth; x++ {
		for y := 0; y < CaptchaHeight; y++ {
			img.Set(x, y, color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255})
		}
	}
	text := GenerateCaptchaText()
	addLabel(img, 10, 50, text)
	buf := new(bytes.Buffer)
	png.Encode(buf, img)
	return text, buf
}

func GenerateCaptchaText() string {
	text := ""
	for i := 0; i < CaptchaLength; i++ {
		text += string(CaptchaChars[rand.Intn(len(CaptchaChars))])
	}
	return text
}

func VerifyCaptcha(text string, input string) bool {
	return text == input
}

func VerifyCaptchaHandler(c echo.Context) error {
	sess, _ := session.Get("session", c)
	text := sess.Values["captcha"].(string)
	input := c.QueryParam("input")
	return c.JSON(http.StatusOK, VerifyCaptcha(text, input))
}

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{0, 0, 0, 255} // black color
	point := fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: fontFace,
		Dot:  point,
	}
	d.DrawString(label)
}

func GenerateCaptchaHandler(c echo.Context) error {
	text, imgBuf := GenerateCaptcha()
	sess, _ := session.Get("session", c)
	sess.Values["captcha"] = text
	sess.Save(c.Request(), c.Response())
	c.Response().Header().Set(echo.HeaderContentType, "image/png")
	return c.Blob(http.StatusOK, "image/png", imgBuf.Bytes())
}
