package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"achan.moe/auth"
	"achan.moe/boardimages"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/blocker"
	"achan.moe/utils/config"
	"achan.moe/utils/news"
	"achan.moe/utils/stats"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

var BoardModel models.Board
var ThreadModel models.ThreadPost
var PostModel models.Posts

var On = true
var b = blocker.NewBlocker()
var ctx = context.Background()

var Client *redis.Client

func Connect() *redis.Client {
	if Client == nil {
		InitCache()
	}
	return Client
}
func InitCache() {
	uri := os.Getenv("REDIS_URI")
	if uri == "" {
		logs.Fatal("REDIS_URI not set")
	}
	Client = redis.NewClient(&redis.Options{
		Addr:     uri,
		Password: "",
		DB:       0,
	})
	ping, err := Client.Ping(ctx).Result()
	if err != nil {
		logs.Fatal("Failed to connect to Redis: %v", err)
	}
	logs.Info("Connected to Redis: %v", ping)
	ClearCache()

	CacheBoards()
	boards := GetBoards()
	for _, board := range boards {
		CacheThreads(board.BoardID)
		threads := models.GetThreads(board.BoardID)
		for _, thread := range threads {
			CacheImage(board.BoardID, thread.Image)
			CacheThumbnail(board.BoardID, thread.Thumbnail)
			fmt.Println("Caching posts: ", thread.ThreadID)
			for _, post := range thread.Posts {
				CacheImage(board.BoardID, post.Image)
				CacheThumbnail(board.BoardID, post.Thumbnail)
			}
		}
	}
	news := news.GetNews()
	for _, n := range news {
		CacheNews(n)
	}
	CacheGlobalPostCount()
	CacheTotalUserCount()
	CacheGlobalConfig()
	CacheTotalSize()
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
	err = Client.Set(ctx, "board:"+board.BoardID, data, 0).Err()
	if err != nil {
		logs.Error("Failed to set board key: ", err)
	}
}

func GetBoards() []models.Board {
	boards := []models.Board{}
	keys := Client.Keys(ctx, "board:*").Val()
	for _, key := range keys {
		board := models.Board{}
		data, err := Client.Get(ctx, key).Bytes()
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
	data, err := Client.Get(ctx, "board:"+boardID).Bytes()
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
		CacheThread(boardID, thread)
	}
}

func CacheThread(boardID string, thread models.ThreadPost) {
	data, err := json.Marshal(thread)
	if err != nil {
		logs.Error("Failed to marshal thread: ", err)
		return
	}
	err = Client.Set(ctx, "thread:"+boardID+":"+thread.ThreadID, data, 0).Err()
	if err != nil {
		logs.Error("Failed to set thread key: ", err)
	}
}

func CacheImage(boardID string, imageid string) {
	image, err := boardimages.GetImage(boardID, imageid)
	if err != nil {
		logs.Error("Error getting image: ", err)
	}
	Client.Set(ctx, "image:"+boardID+":*:"+imageid, image, 0)
}

func CacheThumbnail(boardID string, thumbid string) {
	thumbnail, err := boardimages.GetThumb(boardID, thumbid)
	if err != nil {
		logs.Error("Error getting thumbnail: ", err)
	}
	Client.Set(ctx, "thumbnail:"+boardID+":*:"+thumbid, thumbnail, 0)
}

func CacheNews(news models.News) {
	Client.Set(ctx, "news:"+news.ID, news, 0)
}

func CacheGlobalPostCount() {
	count := stats.GetGlobalPostCount()
	Client.Set(ctx, "global:postcount", count, 0)
}

func GetGlobalPostCount() int {
	count := 0
	Client.Get(ctx, "global:postcount").Scan(&count)
	return count
}

func CacheTotalUserCount() {
	count := stats.GetTotalUserCount()
	Client.Set(ctx, "global:usercount", count, 0)
}

func GetTotalUserCount() int {
	count := 0
	Client.Get(ctx, "global:usercount").Scan(&count)
	return count
}

func CacheGlobalConfig() {
	config := config.ReadGlobalConfig()
	json, err := json.Marshal(config)
	if err != nil {
		logs.Error("Failed to marshal global config: ", err)
		return
	}
	Client.Set(ctx, "global:config", json, 0)
}

func GetGlobalConfig() config.GlobalConfig {
	config := config.GlobalConfig{}
	data, err := Client.Get(ctx, "global:config").Bytes()
	if err != nil {
		logs.Error("Error getting global config: ", err)
		return config
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		logs.Error("Failed to unmarshal global config: ", err)
		return config
	}
	return config
}

func CacheTotalSize() {
	size := stats.GetTotalSize()
	Client.Set(ctx, "global:size", size, 0)
}

func GetTotalSize() float64 {
	size := 0.0
	Client.Get(ctx, "global:size").Scan(&size)
	return size
}

func GetNews(newsID string) models.News {
	news := models.News{}
	Client.Get(ctx, "news:"+newsID).Scan(&news)
	return news
}

func GetImage(boardID string, imageid string) models.Image {
	image := models.Image{}
	Client.Get(ctx, "image:"+boardID+":*:"+imageid).Scan(&image)
	return image
}

func GetThumbnail(boardID string, thumbid string) models.Image {
	thumbnail := models.Image{}
	Client.Get(ctx, "thumbnail:"+boardID+":*:"+thumbid).Scan(&thumbnail)
	return thumbnail
}
func GetThread(boardID string, threadID string) models.ThreadPost {
	thread := models.ThreadPost{}
	data, err := Client.Get(ctx, "thread:"+boardID+":"+threadID).Result()
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
	keys := Client.Keys(ctx, "recent:*").Val()
	for _, key := range keys {
		thread := models.ThreadPost{}
		Client.Get(ctx, key).Scan(&thread)
		threads = append(threads, thread)
	}
	return threads
}

func GetBoardsHandler() []models.Board {
	boards := []models.Board{}
	keys := Client.Keys(ctx, "board:*").Val()
	for _, key := range keys {
		board := models.Board{}
		Client.Get(ctx, key).Scan(&board)
		boards = append(boards, board)
	}
	return boards
}

func GetBoardHandler(boardID string) models.Board {
	board := models.Board{}
	Client.Get(ctx, "board:"+boardID).Scan(&board)
	return board
}

func GetThreadsHandler(c echo.Context, boardID string) []models.ThreadPost {
	threads := []models.ThreadPost{}
	keys := Client.Keys(ctx, "thread:"+boardID+":*").Val()
	for _, key := range keys {
		thread := models.ThreadPost{}
		data, err := Client.Get(ctx, key).Bytes()
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
	keys := Client.Keys(ctx, "thread:"+boardID+":"+threadID).Val()
	if len(keys) == 0 {
		logs.Error("No keys found for thread:", boardID, threadID)
		return thread
	}
	data := Client.Get(ctx, keys[0]).Val()
	err := json.Unmarshal([]byte(data), &thread)
	if err != nil {
		logs.Error("Failed to unmarshal thread: ", err)
	}

	if !auth.AdminCheck(c) {
		thread.IP = ""
		thread.TrueUser = ""
		for i := range thread.Posts {
			thread.Posts[i].IP = ""
			thread.Posts[i].TrueUser = ""
		}
	}

	return thread
}
func GetThumbnailHandler(boardID string, thumbid string) models.Image {
	thumbnail := models.Image{}
	Client.Get(ctx, "thumbnail:"+boardID+":"+thumbid).Scan(&thumbnail)
	return thumbnail
}

func GetImageHandler(boardID string, imageid string) models.Image {
	image := models.Image{}
	Client.Get(ctx, "image:"+boardID+":*:"+imageid).Scan(&image)
	return image
}

func GetNewsHandler(newsID string) models.News {
	news := models.News{}
	Client.Get(ctx, "news:"+newsID).Scan(&news)
	return news
}

func GetAllNewsHandler() []models.News {
	news := []models.News{}
	keys := Client.Keys(ctx, "news:*").Val()
	for _, key := range keys {
		n := models.News{}
		Client.Get(ctx, key).Scan(&n)
		news = append(news, n)
	}
	return news
}

func AddPostToThreadCache(boardID string, threadID string, post models.Posts) {
	key := "thread:" + boardID + ":" + threadID
	logs.Info("Fetching thread with key: ", key)
	data, err := Client.Get(ctx, key).Bytes()
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

	err = Client.Set(ctx, key, updatedData, 0).Err()
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
	err = Client.Set(ctx, key, data, 0).Err()
	if err != nil {
		logs.Error("Failed to set thread key: ", err)
	}
}

func AddThreadPostCountToCache(boardID string, threadID string) {
	key := "thread:" + boardID + ":" + threadID
	data, err := Client.Get(ctx, key).Bytes()
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

	err = Client.Set(ctx, key, updatedData, 0).Err()
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
	err = Client.Set(ctx, key, data, 0).Err()
	if err != nil {
		logs.Error("Failed to set recent thread key: ", err)
	}

}
func GetTotalThreadPostCount(boardID string, threadID string) int {
	key := "thread:" + boardID + ":" + threadID
	data, err := Client.Get(ctx, key).Bytes()
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
	data, err := Client.Get(ctx, key).Bytes()
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
	return Client.Exists(ctx, key).Val() == 1
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
	key := "thread:" + boardID + ":" + threadID
	Client.Del(ctx, key)
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

func ClearCache() {
	b.Start()
	Client.FlushAll(ctx)
	b.Close()
}

func ReloadEntireCache() {
	b.Start()
	ClearCache()
	InitCache()
	b.Close()
}
