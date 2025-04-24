package captcha

import (
	"bytes"
	"context"
	"crypto/rand"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"net/http"
	"os"

	"achan.moe/logs"
	"achan.moe/utils/cache"
	"github.com/labstack/echo/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var ctx = context.Background()

func generateRandomString(length int) string {
	const maxLength = 7
	if length > maxLength {
		length = maxLength
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := range b {
		randomBigInt, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			panic("failed to generate random number: " + err.Error())
		}
		b[i] = charset[randomBigInt.Int64()]
	}
	return string(b)
}

func GenerateCaptcha(c echo.Context) error {
	captchaStr := generateRandomString(7)
	clientIP := c.RealIP()
	exists, err := cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Exists().Key("captcha:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to check if captcha exists in Redis: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "message": "Internal server error"})
	}
	if exists {
		logs.Error("Captcha already exists for IP: " + clientIP)
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "message": "Captcha already exists"})
	}
	_, err = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Set().Key("captcha:"+clientIP).Value(captchaStr).ExSeconds(60).Build()).AsBool()
	if err != nil {
		logs.Fatal("Failed to set captcha in Redis: " + err.Error())
	}
	return nil
}

func GetCaptcha(c echo.Context) error {
	clientIP := c.RealIP()
	GenerateCaptcha(c)
	captchaData, err := cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Get().Key("captcha:"+clientIP).Build()).AsBytes()
	if err != nil {
		logs.Error("Failed to get captcha from Redis: " + err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "message": "Captcha not found"})
	}
	captcha := string(captchaData)

	// Create a new image with a white background
	captchaImage := image.NewRGBA(image.Rect(0, 0, 200, 100))
	draw.Draw(captchaImage, captchaImage.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	// Load a font
	fontPath := "assets/fonts/gohu.ttf"
	fontFile, err := os.Open(fontPath)
	if err != nil {
		logs.Error("Failed to open font file: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "message": "Internal server error"})
	}
	defer fontFile.Close()
	face := &font.Drawer{
		Dst:  captchaImage,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
	}
	if err != nil {
		logs.Error("Failed to create font face: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "message": "Internal server error"})
	}
	borderColor := color.RGBA{0, 0, 0, 255}
	draw.Draw(captchaImage, captchaImage.Bounds(), &image.Uniform{borderColor}, image.Point{}, draw.Src)
	draw.Draw(captchaImage, image.Rect(1, 1, captchaImage.Bounds().Max.X-1, captchaImage.Bounds().Max.Y-1), &image.Uniform{color.White}, image.Point{}, draw.Src)
	// Draw the captcha text
	face.Dot = fixed.P(10, 50)
	face.Src = image.NewUniform(color.RGBA{0, 0, 0, 255})
	face.DrawString(captcha)
	var buf bytes.Buffer
	err = png.Encode(&buf, captchaImage)
	if err != nil {
		logs.Error("Failed to encode captcha image: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "message": "Internal server error"})
	}
	_, err = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Exists().Key("captcha_image:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to delete existing captcha image: " + err.Error())
	}
	data := string(buf.Bytes())
	_, err = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Set().Key("captcha_image:"+clientIP).Value(data).ExSeconds(60).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to set captcha image in Redis: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "message": "Internal server error"})
	}

	c.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Expires", "0")
	c.Response().Header().Set("Content-Disposition", "inline; filename=captcha.png")
	c.Response().Header().Set("Content-Type", "image/png")
	c.Response().WriteHeader(http.StatusOK)
	_, err = c.Response().Write(buf.Bytes())
	if err != nil {
		logs.Error("Failed to write captcha image to response: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "message": "Internal server error"})
	}

	return nil
}
func VerifyCaptcha(c echo.Context, captchaStr string) bool {
	clientIP := c.RealIP()
	captchaData, err := cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Get().Key("captcha:"+clientIP).Build()).AsBytes()
	if err != nil {
		logs.Error("Failed to get captcha from Redis: " + err.Error())
		_ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha:"+clientIP).Build())
		_ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha_image:"+clientIP).Build())
		_ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha_verified:"+clientIP).Build())
		return false
	}
	captcha := string(captchaData)
	if captcha == "" {
		logs.Error("Captcha not found for IP: " + clientIP)
		_ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha:"+clientIP).Build())
		_ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha_image:"+clientIP).Build())
		_ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha_verified:"+clientIP).Build())
		return false
	}

	if captchaStr == captcha {
		logs.Info("Captcha verification successful for IP: " + clientIP)
		_, _ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha:"+clientIP).Build()).AsBool()
		_, _ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha_image:"+clientIP).Build()).AsBool()
		_, _ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Set().Key("captcha_verified:"+clientIP).Value("true").ExSeconds(60).Build()).AsBool()
		return true
	} else {
		logs.Error("Captcha verification failed for IP: " + clientIP)
		_, _ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha:"+clientIP).Build()).AsBool()
		_, _ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha_image:"+clientIP).Build()).AsBool()
		_, _ = cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Del().Key("captcha_verified:"+clientIP).Build()).AsBool()
		return false
	}
}

func VerifyCaptchaHandler(c echo.Context) error {
	captchaStr := c.QueryParam("captcha")
	if VerifyCaptcha(c, captchaStr) {
		return c.JSON(200, map[string]string{"status": "success"})
	} else {
		return c.JSON(400, map[string]string{"status": "error", "message": "Invalid captcha"})
	}
}

func IsCaptchaVerified(c echo.Context) bool {
	clientIP := c.RealIP()
	captchaVerified, err := cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Get().Key("captcha_verified:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to get captcha verification status from Redis: " + err.Error())
		return false
	}
	if captchaVerified == true {
		logs.Info("Captcha already verified for IP: " + clientIP)
		return true
	}
	return false
}

func GetCurrentCaptcha(c echo.Context) error {
	clientIP := c.RealIP()
	captcha, err := cache.ClientCaptcha.Do(ctx, cache.ClientCaptcha.B().Get().Key("captcha:"+clientIP).Build()).AsStrSlice()
	captchaData := captcha[0]
	if err != nil {
		logs.Error("Failed to get captcha from Redis: " + err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "message": "Captcha not found"})
	}
	return c.JSON(http.StatusOK, map[string]string{"captcha": captchaData})
}
