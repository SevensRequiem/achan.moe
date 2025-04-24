package ratelimit

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"achan.moe/logs"
	"achan.moe/utils/cache"
	"github.com/labstack/echo/v4"
)

var On = true
var ctx = context.Background()

type ThreadRateLimit struct {
	ID          int
	clientIP    string
	DateCreated string
	TimeLeft    int
	Requests    int
}

type PostRateLimit struct {
	ID          int
	clientIP    string
	DateCreated string
	TimeLeft    int
	Requests    int
}

type RegisterRateLimit struct {
	ID          int
	clientIP    string
	DateCreated string
	TimeLeft    int
	Requests    int
}

type LoginRateLimit struct {
	ID          int
	clientIP    string
	DateCreated string
	TimeLeft    int
	Requests    int
}

type CaptchaRateLimit struct {
	ID          int
	clientIP    string
	DateCreated string
	TimeLeft    int
	Requests    int
}

func generateRandomID() int {
	max := new(big.Int).SetInt64(999999999999999999)
	min := new(big.Int).SetInt64(0)
	randomBigInt, _ := rand.Int(rand.Reader, new(big.Int).Sub(max, min))
	return int(randomBigInt.Int64())
}

func IsThreadRateLimited(clientIP string) bool {
	ratelimit, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Exists().Key("thread_rate_limit:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to get captcha verification status from Redis: " + err.Error())
		return false
	}
	if ratelimit {
		return true
	}
	return false
}

func IsPostRateLimited(clientIP string) bool {
	ratelimit, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Exists().Key("post_rate_limit:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to get captcha verification status from Redis: " + err.Error())
		return false
	}
	if ratelimit {
		return true
	}
	return false
}
func IsRegisterRateLimited(clientIP string) bool {
	ratelimit, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Exists().Key("register_rate_limit:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to get captcha verification status from Redis: " + err.Error())
		return false
	}
	if ratelimit {
		return true
	}
	return false
}
func IsLoginRateLimited(clientIP string) bool {
	ratelimit, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Exists().Key("login_rate_limit:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to get captcha verification status from Redis: " + err.Error())
		return false
	}
	if ratelimit {
		return true
	}
	return false
}

func IsCaptchaRateLimited(clientIP string) bool {
	ratelimit, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Exists().Key("captcha_rate_limit:"+clientIP).Build()).AsBool()
	if err != nil {
		logs.Error("Failed to get captcha verification status from Redis: " + err.Error())
		return false
	}
	if ratelimit {
		return true
	}
	return false
}
func ThreadHandler(c echo.Context) (bool, error) {
	clientIP := c.RealIP()
	if IsThreadRateLimited(clientIP) {
		data, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Get().Key("thread_rate_limit:"+clientIP).Build()).AsBytes()
		if err != nil {
			return false, fmt.Errorf("failed to get thread rate limit: %w", err)
		}

		var threadRateLimit ThreadRateLimit
		err = json.Unmarshal(data, &threadRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal thread rate limit: %w", err)
		}

		currentTime := time.Now()
		createdTime, err := time.Parse("2006-01-02 15:04:05", threadRateLimit.DateCreated)
		if err != nil {
			return false, fmt.Errorf("failed to parse date created: %w", err)
		}

		if currentTime.Sub(createdTime).Seconds() > float64(threadRateLimit.TimeLeft) {
			_, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Del().Key("thread_rate_limit:"+clientIP).Build()).AsBool()
			if err != nil {
				return false, fmt.Errorf("failed to delete expired thread rate limit: %w", err)
			}
			threadRateLimit = ThreadRateLimit{}
		}

		threadRateLimit.ID = generateRandomID()
		threadRateLimit.clientIP = clientIP
		threadRateLimit.DateCreated = time.Now().Format("2006-01-02 15:04:05")
		if threadRateLimit.Requests > 3 {
			threadRateLimit.TimeLeft = 300 * threadRateLimit.Requests
		} else {
			threadRateLimit.TimeLeft = 300
		}
		threadRateLimit.Requests++
		dataBytes, err := json.Marshal(threadRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal thread rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("thread_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set thread rate limit: %w", err)
		}
		if threadRateLimit.Requests > 3 {
			return true, nil
		}
	} else {
		threadRateLimit := ThreadRateLimit{
			ID:          generateRandomID(),
			clientIP:    clientIP,
			DateCreated: time.Now().Format("2006-01-02 15:04:05"),
			TimeLeft:    300,
			Requests:    1,
		}
		dataBytes, err := json.Marshal(threadRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal thread rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("thread_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set thread rate limit: %w", err)
		}
	}
	return false, nil
}

func PostHandler(c echo.Context) (bool, error) {
	clientIP := c.RealIP()
	if IsPostRateLimited(clientIP) {
		data, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Get().Key("post_rate_limit:"+clientIP).Build()).AsBytes()
		if err != nil {
			return false, fmt.Errorf("failed to get post rate limit: %w", err)
		}

		var postRateLimit PostRateLimit
		err = json.Unmarshal(data, &postRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal post rate limit: %w", err)
		}

		currentTime := time.Now()
		createdTime, err := time.Parse("2006-01-02 15:04:05", postRateLimit.DateCreated)
		if err != nil {
			return false, fmt.Errorf("failed to parse date created: %w", err)
		}

		if currentTime.Sub(createdTime).Seconds() > float64(postRateLimit.TimeLeft) {
			_, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Del().Key("post_rate_limit:"+clientIP).Build()).AsBool()
			if err != nil {
				return false, fmt.Errorf("failed to delete expired post rate limit: %w", err)
			}
			postRateLimit = PostRateLimit{}
		}

		postRateLimit.ID = generateRandomID()
		postRateLimit.clientIP = clientIP
		postRateLimit.DateCreated = time.Now().Format("2006-01-02 15:04:05")
		if postRateLimit.Requests > 3 {
			postRateLimit.TimeLeft = 300 * postRateLimit.Requests
		} else {
			postRateLimit.TimeLeft = 300
		}
		postRateLimit.Requests++
		dataBytes, err := json.Marshal(postRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal post rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("post_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set post rate limit: %w", err)
		}
		if postRateLimit.Requests > 3 {
			return true, nil
		}
	} else {
		postRateLimit := PostRateLimit{
			ID:          generateRandomID(),
			clientIP:    clientIP,
			DateCreated: time.Now().Format("2006-01-02 15:04:05"),
			TimeLeft:    300,
			Requests:    1,
		}
		dataBytes, err := json.Marshal(postRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal post rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("post_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set post rate limit: %w", err)
		}
	}
	return false, nil
}

func RegisterHandler(c echo.Context) (bool, error) {
	clientIP := c.RealIP()
	if IsRegisterRateLimited(clientIP) {
		data, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Get().Key("register_rate_limit:"+clientIP).Build()).AsBytes()
		if err != nil {
			return false, fmt.Errorf("failed to get register rate limit: %w", err)
		}

		var registerRateLimit RegisterRateLimit
		err = json.Unmarshal(data, &registerRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal register rate limit: %w", err)
		}

		currentTime := time.Now()
		createdTime, err := time.Parse("2006-01-02 15:04:05", registerRateLimit.DateCreated)
		if err != nil {
			return false, fmt.Errorf("failed to parse date created: %w", err)
		}

		if currentTime.Sub(createdTime).Seconds() > float64(registerRateLimit.TimeLeft) {
			_, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Del().Key("register_rate_limit:"+clientIP).Build()).AsBool()
			if err != nil {
				return false, fmt.Errorf("failed to delete expired register rate limit: %w", err)
			}
			registerRateLimit = RegisterRateLimit{}
		}

		registerRateLimit.ID = generateRandomID()
		registerRateLimit.clientIP = clientIP
		registerRateLimit.DateCreated = time.Now().Format("2006-01-02 15:04:05")
		if registerRateLimit.Requests > 3 {
			registerRateLimit.TimeLeft = 300 * registerRateLimit.Requests
		} else {
			registerRateLimit.TimeLeft = 300
		}
		registerRateLimit.Requests++
		dataBytes, err := json.Marshal(registerRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal register rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("register_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set register rate limit: %w", err)
		}
		if registerRateLimit.Requests > 3 {
			return true, nil
		}
	} else {
		registerRateLimit := RegisterRateLimit{
			ID:          generateRandomID(),
			clientIP:    clientIP,
			DateCreated: time.Now().Format("2006-01-02 15:04:05"),
			TimeLeft:    300,
			Requests:    1,
		}
		dataBytes, err := json.Marshal(registerRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal register rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("register_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set register rate limit: %w", err)
		}
	}
	return false, nil
}
func LoginHandler(c echo.Context) (bool, error) {
	clientIP := c.RealIP()
	if IsLoginRateLimited(clientIP) {
		data, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Get().Key("login_rate_limit:"+clientIP).Build()).AsBytes()
		if err != nil {
			return false, fmt.Errorf("failed to get login rate limit: %w", err)
		}

		var loginRateLimit LoginRateLimit
		err = json.Unmarshal(data, &loginRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal login rate limit: %w", err)
		}

		currentTime := time.Now()
		createdTime, err := time.Parse("2006-01-02 15:04:05", loginRateLimit.DateCreated)
		if err != nil {
			return false, fmt.Errorf("failed to parse date created: %w", err)
		}

		if currentTime.Sub(createdTime).Seconds() > float64(loginRateLimit.TimeLeft) {
			_, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Del().Key("login_rate_limit:"+clientIP).Build()).AsBool()
			if err != nil {
				return false, fmt.Errorf("failed to delete expired login rate limit: %w", err)
			}
			loginRateLimit = LoginRateLimit{}
		}

		loginRateLimit.ID = generateRandomID()
		loginRateLimit.clientIP = clientIP
		loginRateLimit.DateCreated = time.Now().Format("2006-01-02 15:04:05")
		if loginRateLimit.Requests > 3 {
			loginRateLimit.TimeLeft = 300 * loginRateLimit.Requests
		} else {
			loginRateLimit.TimeLeft = 300
		}
		loginRateLimit.Requests++
		dataBytes, err := json.Marshal(loginRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal login rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("login_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set login rate limit: %w", err)
		}
		if loginRateLimit.Requests > 3 {
			return true, nil
		}
	} else {
		loginRateLimit := LoginRateLimit{
			ID:          generateRandomID(),
			clientIP:    clientIP,
			DateCreated: time.Now().Format("2006-01-02 15:04:05"),
			TimeLeft:    300,
			Requests:    1,
		}
		dataBytes, err := json.Marshal(loginRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal login rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("login_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set login rate limit: %w", err)
		}
	}
	return false, nil
}
func CaptchaHandler(c echo.Context) (bool, error) {
	clientIP := c.RealIP()
	if IsCaptchaRateLimited(clientIP) {
		data, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Get().Key("captcha_rate_limit:"+clientIP).Build()).AsBytes()
		if err != nil {
			return false, fmt.Errorf("failed to get captcha rate limit: %w", err)
		}

		var captchaRateLimit CaptchaRateLimit
		err = json.Unmarshal(data, &captchaRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal captcha rate limit: %w", err)
		}

		currentTime := time.Now()
		createdTime, err := time.Parse("2006-01-02 15:04:05", captchaRateLimit.DateCreated)
		if err != nil {
			return false, fmt.Errorf("failed to parse date created: %w", err)
		}

		if currentTime.Sub(createdTime).Seconds() > float64(captchaRateLimit.TimeLeft) {
			_, err := cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Del().Key("captcha_rate_limit:"+clientIP).Build()).AsBool()
			if err != nil {
				return false, fmt.Errorf("failed to delete expired captcha rate limit: %w", err)
			}
			captchaRateLimit = CaptchaRateLimit{}
		}

		captchaRateLimit.ID = generateRandomID()
		captchaRateLimit.clientIP = clientIP
		captchaRateLimit.DateCreated = time.Now().Format("2006-01-02 15:04:05")
		if captchaRateLimit.Requests > 3 {
			captchaRateLimit.TimeLeft = 300 * captchaRateLimit.Requests
		} else {
			captchaRateLimit.TimeLeft = 300
		}
		captchaRateLimit.Requests++
		dataBytes, err := json.Marshal(captchaRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal captcha rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("captcha_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set captcha rate limit: %w", err)
		}
		if captchaRateLimit.Requests > 3 {
			return true, nil
		}
	} else {
		captchaRateLimit := CaptchaRateLimit{
			ID:          generateRandomID(),
			clientIP:    clientIP,
			DateCreated: time.Now().Format("2006-01-02 15:04:05"),
			TimeLeft:    300,
			Requests:    1,
		}
		dataBytes, err := json.Marshal(captchaRateLimit)
		if err != nil {
			return false, fmt.Errorf("failed to marshal captcha rate limit: %w", err)
		}
		_, err = cache.ClientRatelimit.Do(ctx, cache.ClientRatelimit.B().Set().Key("captcha_rate_limit:"+clientIP).Value(string(dataBytes)).Build()).AsBool()
		if err != nil {
			return false, fmt.Errorf("failed to set captcha rate limit: %w", err)
		}
	}
	return false, nil
}
