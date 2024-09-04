package captcha

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"time"
)

var (
	CaptchaWidth  = 200
	CaptchaHeight = 100
	CaptchaLength = 6
	CaptchaChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func GenerateCaptcha() (string, image.Image) {
	rand.Seed(time.Now().UnixNano())
	img := image.NewRGBA(image.Rect(0, 0, CaptchaWidth, CaptchaHeight))
	for x := 0; x < CaptchaWidth; x++ {
		for y := 0; y < CaptchaHeight; y++ {
			img.Set(x, y, color.RGBA{uint8(rand.Intn(256)), uint8(rand.Intn(256)), uint8(rand.Intn(256)), 255})
		}
	}
	var buffer bytes.Buffer
	png.Encode(&buffer, img)
	return GenerateCaptchaText(), img
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
