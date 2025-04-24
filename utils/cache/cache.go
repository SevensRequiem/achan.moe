package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"achan.moe/auth"
	"achan.moe/boardimages"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/blocker"
	"achan.moe/utils/stats"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/valkey-io/valkey-go"
)

var BoardModel models.Board
var ThreadModel models.ThreadPost
var PostModel models.Posts

var On = true
var b = blocker.NewBlocker()
var ctx = context.Background()

var ClientData valkey.Client
var ClientMain valkey.Client
var ClientRatelimit valkey.Client
var ClientSession valkey.Client
var ClientCache valkey.Client
var ClientQueue valkey.Client
var ClientBans valkey.Client
var ClientLog valkey.Client
var ClientEmail valkey.Client
var ClientAnnouncement valkey.Client
var ClientWebhook valkey.Client
var ClientCaptcha valkey.Client

func init() {
	InitClientData()
	InitClientMain()
	InitClientRatelimit()
	InitClientSession()
	InitClientCache()
	InitClientQueue()
	InitClientBans()
	InitClientLog()
	InitClientEmail()
	InitClientAnnouncement()
	InitClientWebhook()
	InitClientCaptcha()
	ClearCache()
}
func InitCaches() {

	CacheBoards()
	boards := GetBoards()
	threadcount := 0
	postcount := 0
	for _, board := range boards {
		fmt.Println("Caching threads for board: ", board)
		CacheThreads(board.BoardID)
		threads := models.GetThreads(board.BoardID)
		for _, thread := range threads {
			fmt.Println("counting thread: ", thread.ThreadID)
			threadcount++
			for _, post := range thread.Posts {
				fmt.Println("counting post: ", post.PostID)
				postcount++
			}
		}
	}
	count := threadcount + postcount
	stats.SetTotalPosts(count)
	news, err := models.GetAllNews()
	if err != nil {
		logs.Error("Error getting news: ", err)
		return
	}
	for _, n := range news {
		fmt.Println("Caching news: ", n)
		CacheNews(n)
	}

	CacheAnnouncements()
	CacheBans()
	SetStatsToCache()
}

func InitClientData() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_DATA")
	if db == "" {
		logs.Fatal("VALKEY_DB_DATA not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_DATA: %v", err)
	}
	ClientData, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Data",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}

func InitClientMain() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_MAIN")
	if db == "" {
		logs.Fatal("VALKEY_DB_MAIN not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_MAIN: %v", err)
	}
	ClientMain, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Main",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}

func InitClientRatelimit() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_RATELIMIT")
	if db == "" {
		logs.Fatal("VALKEY_DB_RATELIMIT not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_RATELIMIT: %v", err)
	}
	ClientRatelimit, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Ratelimit",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientSession() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_SESSION")
	if db == "" {
		logs.Fatal("VALKEY_DB_SESSION not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_SESSION: %v", err)
	}
	ClientSession, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Session",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientCache() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_CACHE")
	if db == "" {
		logs.Fatal("VALKEY_DB_CACHE not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_CACHE: %v", err)
	}
	ClientCache, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Cache",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientQueue() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_QUEUE")
	if db == "" {
		logs.Fatal("VALKEY_DB_QUEUE not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_QUEUE: %v", err)
	}
	ClientQueue, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Queue",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientBans() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_BANS")
	if db == "" {
		logs.Fatal("VALKEY_DB_BANS not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_BANS: %v", err)
	}
	ClientBans, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "IPBans",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}

func CacheBans() {
	if ClientBans == nil {
		logs.Error("ClientBans is not initialized")
		return
	}

	bans, err := models.GetAllBans()
	if err != nil {
		logs.Error("Error getting bans: ", err)
		return
	}

	for _, ban := range bans {
		data, err := json.Marshal(ban)
		if err != nil {
			logs.Error("Failed to marshal ban: ", err)
			continue
		}
		err = ClientBans.Do(ctx, ClientBans.B().Set().Key(ban.IP).Value(string(data)).Build()).Error()
		if err != nil {
			logs.Error("Failed to set ban key: ", err)
			continue
		}
		fmt.Println("Caching ban: ", ban)
	}
}
func InitClientLog() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_LOG")
	if db == "" {
		logs.Fatal("VALKEY_DB_LOG not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_LOG: %v", err)
	}
	ClientLog, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Log",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientEmail() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_EMAIL")
	if db == "" {
		logs.Fatal("VALKEY_DB_EMAIL not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_EMAIL: %v", err)
	}
	ClientMain, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Email",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientAnnouncement() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_ANNOUNCEMENT")
	if db == "" {
		logs.Fatal("VALKEY_DB_ANNOUNCEMENT not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_ANNOUNCEMENT: %v", err)
	}
	ClientAnnouncement, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Announcement",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientWebhook() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_WEBHOOK")
	if db == "" {
		logs.Fatal("VALKEY_DB_WEBHOOK not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_WEBHOOK: %v", err)
	}
	ClientWebhook, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Webhook",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func InitClientCaptcha() {
	godotenv.Load()
	uri := os.Getenv("VALKEY_URI")
	if uri == "" {
		logs.Fatal("VALKEY_URI not set")
	}
	db := os.Getenv("VALKEY_DB_CAPTCHA")
	if db == "" {
		logs.Fatal("VALKEY_DB_CAPTCHA not set")
	}
	dbNum, err := strconv.Atoi(db)
	if err != nil {
		logs.Fatal("Invalid VALKEY_DB_CAPTCHA: %v", err)
	}
	ClientCaptcha, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{uri},
		ClientName:  "Captcha",
		SelectDB:    dbNum,
	})
	if err != nil {
		logs.Fatal("Failed to create Redis client: %v", err)
	}
}
func CacheBoards() {
	boards := models.GetBoards()
	for _, board := range boards {
		fmt.Println("Caching board: ", board)
		CacheBoard(board)
	}
}

func CacheBoard(board models.Board) {
	data, err := json.Marshal(board)
	if err != nil {
		logs.Error("Failed to marshal board: ", err)
		return
	}
	err = ClientMain.Do(ctx, ClientMain.B().Set().Key("board:"+board.BoardID).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to cache board: ", err)
	}
}
func GetBoards() []models.Board {
	boards := []models.Board{}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("board:*").Build()).AsStrSlice()
	if err != nil {
		logs.Error("Error fetching keys for boards: ", err)
		return boards
	}
	if len(keys) == 0 {
		logs.Error("No keys found for boards")
		return boards
	}
	for _, key := range keys {
		board := models.Board{}
		data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
		if err != nil {
			logs.Error("Error getting board: ", err)
			continue
		}
		err = json.Unmarshal(data, &board)
		if err != nil {
			logs.Error("Failed to unmarshal board: ", err)
			continue
		}
		boards = append(boards, board)
	}

	sort.Slice(boards, func(i, j int) bool {
		return boards[i].Name < boards[j].Name
	})

	return boards
}
func GetBoard(boardID string) models.Board {
	board := models.Board{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("board:"+boardID).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting board: ", err)
		return board
	}
	err = json.Unmarshal(data, &board)
	if err != nil {
		logs.Error("Failed to unmarshal board: ", err)
	}
	return board
}
func CacheThreads(boardID string) {
	threads := models.GetThreads(boardID)
	for _, thread := range threads {
		fmt.Println("Caching thread: ", thread.ThreadID)
		CacheThread(boardID, thread)
	}
}

func CacheThread(boardID string, thread models.ThreadPost) {
	data, err := json.Marshal(thread)
	if err != nil {
		logs.Error("Failed to marshal thread: ", err)
		return
	}
	err = ClientMain.Do(ctx, ClientMain.B().Set().Key("thread:"+boardID+":"+thread.ThreadID).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to cache thread: ", err)
	}
}

func CacheImage(boardID string, imageid string) {
	image, err := boardimages.GetImage(boardID, imageid)
	if err != nil {
		logs.Error("Error getting image: ", err)
		return
	}

	data, err := json.Marshal(image)
	if err != nil {
		logs.Error("Failed to marshal image: ", err)
		return
	}

	err = ClientMain.Do(ctx, ClientMain.B().Set().Key("image:"+boardID+":*:"+imageid).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to cache image: ", err)
	}
}

func CacheThumbnail(boardID string, thumbid string) {
	thumbnail, err := boardimages.GetThumb(boardID, thumbid)
	if err != nil {
		logs.Error("Error getting thumbnail: ", err)
		return
	}

	data, err := json.Marshal(thumbnail)
	if err != nil {
		logs.Error("Failed to marshal thumbnail: ", err)
		return
	}
	err = ClientMain.Do(ctx, ClientMain.B().Set().Key("thumbnail:"+boardID+":*:"+thumbid).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to cache thumbnail: ", err)
	}
}

func CacheNews(news models.News) {
	data, err := json.Marshal(news)
	if err != nil {
		logs.Error("Failed to marshal news: ", err)
		return
	}
	err = ClientMain.Do(ctx, ClientMain.B().Set().Key("news:"+news.ID).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to cache news: ", err)
	}
}

func ClearNews() {
	err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("news:*").Build()).Error()
	if err != nil {
		logs.Error("Error clearing news: ", err)
		return
	}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("news:*").Build()).AsStrSlice()
	if err != nil {
		logs.Error("No keys found for news: ", err)
		return
	}
	for _, key := range keys {
		err = ClientMain.Do(ctx, ClientMain.B().Del().Key(key).Build()).Error()
		if err != nil {
			logs.Error("Failed to delete news key: ", err)
		}
	}
}

func DeleteNews(newsID string) {
	err := ClientMain.Do(ctx, ClientMain.B().Del().Key("news:"+newsID).Build()).Error()
	if err != nil {
		logs.Error("Failed to delete news: ", err)
	}
}
func CacheAnnouncements() {
	announcements := models.GetAllAnnouncements()
	for _, announcement := range announcements {
		CacheAnnouncement(announcement)
	}
}

func CacheAnnouncement(announcement models.Announcement) {
	data, err := json.Marshal(announcement)
	if err != nil {
		logs.Error("Failed to marshal announcement: ", err)
		return
	}
	err = ClientMain.Do(ctx, ClientMain.B().Set().Key("announcement:"+announcement.BoardID).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to cache announcement: ", err)
	}
}

func GetAnnouncements() []models.Announcement {
	announcementsList := []models.Announcement{}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("announcement:*").Build()).AsStrSlice()
	if err != nil {
		logs.Error("No keys found for announcements: ", err)
		return nil
	}
	for _, key := range keys {
		var announcement models.Announcement
		data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
		if err != nil {
			logs.Error("Error getting announcement: ", err)
			continue
		}
		err = json.Unmarshal(data, &announcement)
		if err != nil {
			logs.Error("Failed to unmarshal announcement: ", err)
			continue
		}
		announcementsList = append(announcementsList, announcement)
	}
	return announcementsList
}
func GetAnnouncement(boardID string) models.Announcement {
	announcement := models.Announcement{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("announcement:"+boardID).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting announcement: ", err)
		return announcement
	}
	err = json.Unmarshal(data, &announcement)
	if err != nil {
		logs.Error("Failed to unmarshal announcement: ", err)
	}
	return announcement
}

func GetGlobalAnnouncement() models.Announcement {
	announcements := GetAnnouncements()
	announcement := models.Announcement{}
	for _, a := range announcements {
		if a.BoardID == "global" {
			announcement = a
			break
		}
	}

	return announcement
}

func GetAnnouncementHandler(boardID string) models.Announcement {
	announcement := GetAnnouncement(boardID)
	return announcement
}

func GetNewsByID(newsID string) models.News {
	news := models.News{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("news:"+newsID).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting news: ", err)
	}
	err = json.Unmarshal(data, &news)
	if err != nil {
		logs.Error("Failed to unmarshal news: ", err)
	}
	return news
}

func GetImage(boardID string, imageid string) models.Image {
	image := models.Image{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("image:"+boardID+":*:"+imageid).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting image: ", err)
	}
	err = json.Unmarshal(data, &image)
	if err != nil {
		logs.Error("Failed to unmarshal image: ", err)
	}
	return image
}

func GetThumbnail(boardID string, thumbid string) models.Image {
	thumbnail := models.Image{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("thumbnail:"+boardID+":*:"+thumbid).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting thumbnail: ", err)
	}
	err = json.Unmarshal(data, &thumbnail)
	if err != nil {
		logs.Error("Failed to unmarshal thumbnail: ", err)
	}
	return thumbnail
}
func GetThread(boardID string, threadID string) models.ThreadPost {
	thread := models.ThreadPost{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("thread:"+boardID+":"+threadID).Build()).AsBytes()
	if err != nil {
		logs.Debug("Error getting thread: ", err)
		return thread
	}
	err = json.Unmarshal([]byte(data), &thread)
	if err != nil {
		logs.Debug("Failed to unmarshal thread: ", err)
		return thread
	}
	return thread
}
func GetLatestThreadsHandler() []models.ThreadPost {
	threads := []models.ThreadPost{}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("thread:*").Build()).AsStrSlice()
	if err != nil {
		logs.Error("No keys found for threads: ", err)
		return nil
	}
	for _, key := range keys {
		thread := models.ThreadPost{}
		data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
		if err != nil {
			logs.Error("Failed to get thread for key:", key, "Error:", err)
			continue
		}
		err = json.Unmarshal(data, &thread)
		if err != nil {
			logs.Error("Failed to unmarshal thread for key:", key, "Error:", err)
			continue
		}
		threads = append(threads, thread)
	}

	sort.Slice(threads, func(i, j int) bool {
		return threads[i].Timestamp > threads[j].Timestamp
	})

	if len(threads) > 10 {
		threads = threads[:10]
	}

	return threads
}
func GetBoardsHandler() []models.Board {
	boards := []models.Board{}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("board:*").Build()).AsStrSlice()
	if err == nil {
		logs.Error("No keys found for boards")
		return nil
	}
	for _, key := range keys {
		board := models.Board{}
		data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
		if err != nil {
			logs.Error("Error getting board: ", err)
			continue
		}
		err = json.Unmarshal(data, &board)
		if err != nil {
			logs.Error("Failed to unmarshal board: ", err)
			continue
		}
		boards = append(boards, board)
	}
	return boards
}

func GetBoardHandler(boardID string) models.Board {
	board := models.Board{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("board:"+boardID).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting board: ", err)
	}
	err = json.Unmarshal(data, &board)
	if err != nil {
		logs.Error("Failed to unmarshal board: ", err)
	}
	return board
}

func GetThreadsHandler(c echo.Context, boardID string) []models.ThreadPost {
	threads := []models.ThreadPost{}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("thread:"+boardID+":*").Build()).AsStrSlice()
	if err != nil {
		logs.Error("No keys found for threads: ", err)
		return nil
	}
	for _, key := range keys {
		thread := models.ThreadPost{}
		data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
		if err != nil {
			logs.Error("Error getting thread: ", err)
			continue
		}
		err = json.Unmarshal(data, &thread)
		if err != nil {
			logs.Error("Failed to unmarshal thread: ", err)
			continue
		}

		if !auth.AdminCheck(c) {
			thread.IP = ""
			thread.TrueUser = ""
			for i := range thread.Posts {
				thread.Posts[i].IP = ""
				thread.Posts[i].TrueUser = ""
			}
		}

		threads = append(threads, thread)
	}

	sort.Slice(threads, func(i, j int) bool {
		if threads[i].Sticky && !threads[j].Sticky {
			return true
		}
		if !threads[i].Sticky && threads[j].Sticky {
			return false
		}

		var timeI, timeJ time.Time
		if len(threads[i].Posts) > 0 {
			lastPostI := threads[i].Posts[len(threads[i].Posts)-1]
			timeI = time.Unix(lastPostI.Timestamp, 0)
		} else {
			timeI = time.Unix(threads[i].Timestamp, 0)
		}

		if len(threads[j].Posts) > 0 {
			lastPostJ := threads[j].Posts[len(threads[j].Posts)-1]
			timeJ = time.Unix(lastPostJ.Timestamp, 0)
		} else {
			timeJ = time.Unix(threads[j].Timestamp, 0)
		}

		return timeI.After(timeJ)
	})

	return threads
}

func GetThreadHandler(c echo.Context, boardID string, threadID string) models.ThreadPost {
	thread := models.ThreadPost{}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("thread:"+boardID+":"+threadID).Build()).AsStrSlice()
	if err != nil {
		logs.Error("Error getting keys: ", err)
		return thread
	}
	if len(keys) == 0 {
		logs.Error("No keys found for thread:", boardID, threadID)
		return thread
	}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(keys[0]).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting thread: ", err)
		return thread
	}
	err = json.Unmarshal([]byte(data), &thread)
	if err != nil {
		logs.Error("Failed to unmarshal thread: ", err)
	}
	thread.Posts = thread.Posts[1:]
	if !auth.AdminCheck(c) {
		thread = models.ThreadPost{
			ID:             thread.ID,
			BoardID:        thread.BoardID,
			ThreadID:       thread.ThreadID,
			Content:        thread.Content,
			PartialContent: thread.PartialContent,
			Image:          thread.Image,
			Thumbnail:      thread.Thumbnail,
			Subject:        thread.Subject,
			Author:         thread.Author,
			Timestamp:      thread.Timestamp,
			Sticky:         thread.Sticky,
			Locked:         thread.Locked,
			PostCount:      thread.PostCount,
			PostNumber:     thread.PostNumber,
			Posts:          thread.Posts,
		}
		for i := range thread.Posts {
			thread.Posts[i] = models.Posts{
				ID:             thread.Posts[i].ID,
				BoardID:        thread.Posts[i].BoardID,
				ParentID:       thread.Posts[i].ParentID,
				PostID:         thread.Posts[i].PostID,
				Content:        thread.Posts[i].Content,
				PartialContent: thread.Posts[i].PartialContent,
				Image:          thread.Posts[i].Image,
				Thumbnail:      thread.Posts[i].Thumbnail,
				Subject:        thread.Posts[i].Subject,
				Author:         thread.Posts[i].Author,
				Timestamp:      thread.Posts[i].Timestamp,
				PostNumber:     thread.Posts[i].PostNumber,
			}
		}
	}

	return thread
}
func GetThumbnailHandler(boardID string, thumbid string) models.Image {
	thumbnail := models.Image{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("thumbnail:"+boardID+":*:"+thumbid).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting thumbnail: ", err)
	}
	err = json.Unmarshal(data, &thumbnail)
	if err != nil {
		logs.Error("Failed to unmarshal thumbnail: ", err)
	}
	return thumbnail
}

func GetImageHandler(boardID string, imageid string) models.Image {
	image := models.Image{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("image:"+boardID+":*:"+imageid).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting image: ", err)
	}
	err = json.Unmarshal(data, &image)
	if err != nil {
		logs.Error("Failed to unmarshal image: ", err)
	}
	return image
}

func GetNewsByIDHandler(newsID string) models.News {
	news := models.News{}
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key("news:"+newsID).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting news: ", err)
	}
	err = json.Unmarshal(data, &news)
	if err != nil {
		logs.Error("Failed to unmarshal news: ", err)
	}
	return news
}

func GetAllNewsHandler(c echo.Context) []models.News {
	news := []models.News{}
	keys, err := ClientMain.Do(ctx, ClientMain.B().Keys().Pattern("news:*").Build()).AsStrSlice()
	if err != nil {
		logs.Error("No keys found for news: ", err)
		return nil
	}
	for _, key := range keys {
		newsItem := models.News{}
		data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
		if err != nil {
			logs.Error("Error getting news: ", err)
			continue
		}
		err = json.Unmarshal(data, &newsItem)
		if err != nil {
			logs.Error("Failed to unmarshal news: ", err)
			continue
		}
		news = append(news, newsItem)
	}
	sort.Slice(news, func(i, j int) bool {
		timeI := news[i].Date
		timeJ := news[j].Date
		return timeI > timeJ
	})
	for i := range news {
		if !auth.AdminCheck(c) {
			news[i].Content = ""
			news[i].Author = ""
		}
	}
	return news
}

func AddPostToThreadCache(boardID string, threadID string, post models.Posts) {
	key := "thread:" + boardID + ":" + threadID
	logs.Info("Fetching thread with key: ", key)
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting thread: ", err)
		return
	}

	logs.Info("Successfully fetched thread data: ", string(data))

	thread := models.ThreadPost{}
	err = json.Unmarshal(data, &thread)
	if err != nil {
		logs.Error("Failed to unmarshal thread: ", err)
		return
	}

	logs.Info("Successfully unmarshalled thread: ", thread)

	thread.Posts = append(thread.Posts, post)
	logs.Info("Appended post to thread: ", thread)

	updatedData, err := json.Marshal(thread)
	if err != nil {
		logs.Error("Failed to marshal updated thread: ", err)
		return
	}

	logs.Info("Successfully marshalled updated thread: ", string(updatedData))

	err = ClientMain.Do(ctx, ClientMain.B().Set().Key(key).Value(string(updatedData)).Build()).Error()
	if err != nil {
		logs.Error("Failed to update thread in cache: ", err)
		return
	}

	logs.Info("Successfully updated thread in cache")
}

func AddThreadToCache(boardID string, thread models.ThreadPost) {
	key := "thread:" + boardID + ":" + thread.ThreadID
	data, err := json.Marshal(thread)
	if err != nil {
		logs.Error("Failed to marshal thread: ", err)
		return
	}
	err = ClientMain.Do(ctx, ClientMain.B().Set().Key(key).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to set thread key: ", err)
	}
}

func AddThreadPostCountToCache(boardID string, threadID string) {
	key := "thread:" + boardID + ":" + threadID
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting thread: ", err)
		return
	}

	thread := models.ThreadPost{}
	err = json.Unmarshal(data, &thread)
	if err != nil {
		logs.Error("Failed to unmarshal thread: ", err)
		return
	}

	thread.PostCount = thread.PostCount + 1
	updatedData, err := json.Marshal(thread)
	if err != nil {
		logs.Error("Failed to marshal updated thread: ", err)
		return
	}

	err = ClientMain.Do(ctx, ClientMain.B().Set().Key(key).Value(string(updatedData)).Build()).Error()
	if err != nil {
		logs.Error("Failed to update thread in cache: ", err)
		return
	}
}
func AddToRecentThreadsCache(thread models.ThreadPost) {
	key := "recent:" + thread.ThreadID
	data, err := json.Marshal(thread)
	if err != nil {
		logs.Error("Failed to marshal thread: ", err)
		return
	}
	err = ClientMain.Do(ctx, ClientMain.B().Set().Key(key).Value(string(data)).Build()).Error()
	if err != nil {
		logs.Error("Failed to set recent thread key: ", err)
	}
}
func GetTotalThreadPostCount(boardID string, threadID string) int {
	key := "thread:" + boardID + ":" + threadID
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting thread: ", err)
		return 0
	}

	thread := models.ThreadPost{}
	err = json.Unmarshal(data, &thread)
	if err != nil {
		logs.Error("Failed to unmarshal thread: ", err)
		return 0
	}

	return len(thread.Posts)
}

func GetTotalThreadCount(c echo.Context, boardID string) int {
	threads := GetThreadsHandler(c, boardID)
	return len(threads)
}
func CheckDuplicatePostContent(boardID string, threadID string, content string) bool {
	key := "thread:" + boardID + ":" + threadID
	data, err := ClientMain.Do(ctx, ClientMain.B().Get().Key(key).Build()).AsBytes()
	if err != nil {
		logs.Error("Error getting thread: ", err)
		return false
	}

	thread := models.ThreadPost{}
	err = json.Unmarshal(data, &thread)
	if err != nil {
		logs.Error("Failed to unmarshal thread: ", err)
		return false
	}

	posts := thread.Posts
	for _, post := range posts {
		if post.Content == content {
			return true
		}
	}
	return false
}
func CheckDuplicateThreadContent(c echo.Context, boardID string, content string) bool {
	threads := GetThreadsHandler(c, boardID)
	for _, thread := range threads {
		if thread.Content == content {
			return true
		}
	}
	return false
}
func CheckBoardExists(boardID string) bool {
	key := "board:" + boardID
	check, err := ClientMain.Do(ctx, ClientMain.B().Exists().Key(key).Build()).AsIntSlice()
	if err != nil {
		logs.Error("Error checking if board exists: ", err)
		return false
	}
	if check[0] == 1 {
		return true
	}
	logs.Debug("Board does not exist: ", boardID)
	return false
}

func CheckBoardLocked(boardID string) bool {
	board := GetBoard(boardID)
	return board.Locked
}

func CheckBoardImageOnly(boardID string) bool {
	board := GetBoard(boardID)
	return board.ImageOnly
}

func CheckBoardArchived(boardID string) bool {
	board := GetBoard(boardID)
	return board.Archived
}

func DeleteThreadFromCache(boardID string, threadID string) {
	b.Start()
	err := ClientMain.Do(ctx, ClientMain.B().Del().Key("thread:"+boardID+":"+threadID).Build()).Error()
	if err != nil {
		logs.Error("Failed to delete thread from cache: ", err)
	}
	b.Close()
}
func DeletePostFromCache(boardID string, threadID string, postID string) {
	b.Start()
	thread := GetThread(boardID, threadID)
	for i, post := range thread.Posts {
		if post.PostID == postID {
			thread.Posts = append(thread.Posts[:i], thread.Posts[i+1:]...)
			break
		}
	}
	AddThreadToCache(boardID, thread)
	b.Close()
}
func GetGlobalPostCount() int {
	countStr, err := ClientData.Do(ctx, ClientData.B().Get().Key("post_count").Build()).AsBytes()
	if err != nil || len(countStr) == 0 {
		logs.Error("Error getting post count or key is empty: ", err)
		return 0
	}
	count, err := strconv.Atoi(string(countStr))
	if err != nil {
		logs.Error("Error converting post count to int: ", err)
		return 0
	}
	return count
}

func SetGlobalPostCount(count int) {
	err := ClientData.Do(ctx, ClientData.B().Set().Key("post_count").Value(strconv.Itoa(count)).Build()).Error()
	if err != nil {
		logs.Error("Error setting post count: ", err)
	}
}

func GetTotalSize() float32 {
	countStr, err := ClientData.Do(ctx, ClientData.B().Get().Key("total_size").Build()).AsBytes()
	if err != nil || len(countStr) == 0 {
		logs.Error("Error getting total size or key is empty: ", err)
		return 0
	}
	size, err := strconv.ParseInt(string(countStr), 10, 64)
	if err != nil {
		logs.Error("Error converting total size to int: ", err)
		return 0
	}
	returnedSize := float32(size) / 1024 / 1024 / 1024
	roundedSize := math.Round(float64(returnedSize)*100) / 100
	returnedSize = float32(roundedSize)
	return returnedSize
}

func SetTotalSize(size int) {
	sizeStr := strconv.FormatFloat(float64(size), 'f', -1, 32)
	err := ClientData.Do(ctx, ClientData.B().Set().Key("total_size").Value(sizeStr).Build()).Error()
	if err != nil {
		logs.Error("Error setting total size: ", err)
	}
}
func GetTotalUserCount() int {
	countStr, err := ClientData.Do(ctx, ClientData.B().Get().Key("user_count").Build()).AsBytes()
	if err != nil || len(countStr) == 0 {
		logs.Error("Error getting user count or key is empty: ", err)
		return 0
	}
	count, err := strconv.Atoi(string(countStr))
	if err != nil {
		logs.Error("Error converting user count to int: ", err)
		return 0
	}
	return count
}
func SetTotalUserCount(count int) {
	err := ClientData.Do(ctx, ClientData.B().Set().Key("user_count").Value(strconv.Itoa(count)).Build()).Error()
	if err != nil {
		logs.Error("Error setting user count: ", err)
	}
}
func GetTotalHitCount() int {
	countStr, err := ClientData.Do(ctx, ClientData.B().Get().Key("hit_count").Build()).AsBytes()
	if err != nil || len(countStr) == 0 {
		logs.Error("Error getting hit count or key is empty: ", err)
		return 0
	}
	count, err := strconv.Atoi(string(countStr))
	if err != nil {
		logs.Error("Error converting hit count to int: ", err)
		return 0
	}
	return count
}
func SetTotalHitCount(count int) {
	err := ClientData.Do(ctx, ClientData.B().Set().Key("hit_count").Value(strconv.Itoa(count)).Build()).Error()
	if err != nil {
		logs.Error("Error setting hit count: ", err)
	}
}
func SetStatsToCache() {
	statsData, err := models.GetAllStats()
	if err != nil {
		logs.Error("Error getting stats: ", err)
		return
	}

	var stats models.Stats
	err = json.Unmarshal(statsData, &stats)
	if err != nil {
		logs.Error("Failed to unmarshal stats: ", err)
		return
	}

	logs.Info("Stats fetched: ", stats)

	SetGlobalPostCount(stats.PostCount)
	logs.Info("SetGlobalPostCount completed: ", stats.PostCount)

	SetTotalSize(stats.TotalSize)
	logs.Info("SetTotalSize completed: ", stats.TotalSize)

	SetTotalUserCount(stats.TotalUsers)
	logs.Info("SetTotalUserCount completed: ", stats.TotalUsers)

	SetTotalHitCount(stats.TotalHits)
	logs.Info("SetTotalHitCount completed: ", stats.TotalHits)
}

func ClearCache() {
	if ClientData != nil {
		ClientData.Do(ctx, ClientData.B().Flushdb().Build()).Error()
	}
	if ClientMain != nil {
		ClientMain.Do(ctx, ClientMain.B().Flushdb().Build()).Error()
	}
	if ClientBans != nil {
		ClientBans.Do(ctx, ClientBans.B().Flushdb().Build()).Error()
	}
	if ClientQueue != nil {
		ClientQueue.Do(ctx, ClientQueue.B().Flushdb().Build()).Error()
	}
	if ClientLog != nil {
		ClientLog.Do(ctx, ClientLog.B().Flushdb().Build()).Error()
	}
	if ClientEmail != nil {
		ClientEmail.Do(ctx, ClientEmail.B().Flushdb().Build()).Error()
	}
	if ClientAnnouncement != nil {
		ClientAnnouncement.Do(ctx, ClientAnnouncement.B().Flushdb().Build()).Error()
	}
	if ClientWebhook != nil {
		ClientWebhook.Do(ctx, ClientWebhook.B().Flushdb().Build()).Error()
	}
	if ClientCaptcha != nil {
		ClientCaptcha.Do(ctx, ClientCaptcha.B().Flushdb().Build()).Error()
	}
	if ClientQueue != nil {
		ClientQueue.Do(ctx, ClientQueue.B().Flushdb().Build()).Error()
	}
}

func ReloadEntireCache() {
	b.Start()
	ClearCache()
	InitClientMain()
	b.Close()
}
