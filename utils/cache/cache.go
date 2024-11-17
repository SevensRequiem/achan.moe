package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"achan.moe/auth"
	"achan.moe/board"
	"achan.moe/logs"
	"achan.moe/models"
	"achan.moe/utils/blocker"
	"achan.moe/utils/config"
	"achan.moe/utils/news"
	"github.com/allegro/bigcache/v3"
	"github.com/labstack/echo/v4"
)

var BoardModel models.Board
var ThreadModel models.ThreadPost
var PostModel models.Posts

var Cache *bigcache.BigCache

var On = true
var b = blocker.NewBlocker()

func init() {
	InitCache()
	CacheBoards()
	boards := GetBoards()
	for _, board := range boards {
		CacheThreads(board.BoardID)
		fmt.Println("Caching board: ", board)
		threads := models.GetThreads(board.BoardID)
		fmt.Println("Caching threads: ", len(threads))
		for _, thread := range threads {
			CacheImage(board.BoardID, thread.ThreadID, thread.PostID)
			CacheThumbnail(board.BoardID, thread.ThreadID, thread.PostID)
			fmt.Println("Caching posts: ", thread.ThreadID)
			CachePosts(board.BoardID, thread.ThreadID)
			for _, post := range thread.Posts {
				CacheImage(board.BoardID, thread.ThreadID, post.PostID)
				CacheThumbnail(board.BoardID, thread.ThreadID, post.PostID)
			}
		}
	}
	news := news.GetNews()
	for _, n := range news {
		CacheNews(n)
	}
}

func InitCache() {
	ctx := context.Background()
	config := bigcache.DefaultConfig(0)
	config.CleanWindow = 0
	config.HardMaxCacheSize = 16384
	config.MaxEntrySize = 11000000
	config.Verbose = true

	var err error
	Cache, err = bigcache.New(ctx, config)
	if err != nil {
		logs.Error("Error creating cache: %v", err)
		os.Exit(1)
	}
}

func CacheConfig() {
	config := config.ReadGlobalConfig()
	data, err := json.Marshal(config)
	if err != nil {
		logs.Error("Error marshalling config: %v", err)
		return
	}
	Cache.Set("config:", data)
}
func CacheImage(boardID string, threadID string, postID string) {
	postdata := GetPost(threadID, postID)
	if postdata.Image != "" {
		imagedata, err := board.GetImage(boardID, postdata.Image)
		if err != nil {
			logs.Error("Error getting image: %v", err)
			return
		}
		imageBytes, err := json.Marshal(imagedata)
		if err != nil {
			logs.Error("Error marshalling image: %v", err)
			return
		}
		Cache.Set("image:"+postdata.Image, imageBytes)
	}
}
func GetImage(imageID string) models.Image {
	data, err := Cache.Get("image:" + imageID)
	if err != nil {
		logs.Error("Error getting image: %v", err)
		return models.Image{}
	}

	var image models.Image
	err = json.Unmarshal(data, &image)
	if err != nil {
		logs.Error("Error unmarshalling image: %v", err)
		return models.Image{}
	}

	return image
}

func CacheThumbnail(boardID string, threadID string, postID string) {
	postdata := GetPost(threadID, postID)
	if postdata.Thumbnail != "" {
		imagedata, err := board.GetThumb(boardID, postdata.Thumbnail)
		if err != nil {
			logs.Error("Error getting thumbnail: %v", err)
			return
		}
		imageBytes, err := json.Marshal(imagedata)
		if err != nil {
			logs.Error("Error marshalling thumbnail: %v", err)
			return
		}
		Cache.Set("thumbnail:"+postdata.Thumbnail, imageBytes)
	}
}

func GetThumbnail(thumbID string) models.Image {
	data, err := Cache.Get("thumbnail:" + thumbID)
	if err != nil {
		logs.Error("Error getting thumbnail: %v", err)
		return models.Image{}
	}

	var thumb models.Image
	err = json.Unmarshal(data, &thumb)
	if err != nil {
		logs.Error("Error unmarshalling thumbnail: %v", err)
		return models.Image{}
	}

	return thumb
}
func CacheBoards() {
	boards := models.GetBoards()

	for _, board := range boards {
		data, err := json.Marshal(board)
		if err != nil {
			logs.Error("Error marshalling board: %v", err)
			continue
		}
		Cache.Set("board:"+board.BoardID, data)
	}
}

func CacheThreads(boardID string) {
	threads := models.GetThreads(boardID)

	for _, thread := range threads {
		data, err := json.Marshal(thread)
		if err != nil {
			logs.Error("Error marshalling thread: %v", err)
			continue
		}
		Cache.Set("thread:"+thread.ThreadID, data)
	}
}

func CachePosts(boardID string, threadID string) {
	posts := models.GetThreadPosts(boardID, threadID)

	for _, post := range posts {
		data, err := json.Marshal(post)
		if err != nil {
			logs.Error("Error marshalling post: %v", err)
			continue
		}
		Cache.Set("post:"+threadID+"#"+post.PostID, data)
	}
}

func GetBoard(boardID string) models.Board {
	data, err := Cache.Get("board:" + boardID)
	if err != nil {
		logs.Error("Error getting board: %v", err)
		return models.Board{}
	}

	var board models.Board
	err = json.Unmarshal(data, &board)
	if err != nil {
		logs.Error("Error unmarshalling board: %v", err)
		return models.Board{}
	}

	return board
}

func GetThread(threadID string) models.ThreadPost {
	data, err := Cache.Get("thread:" + threadID)
	if err != nil {
		logs.Error("Error getting thread: %v", err)
		return models.ThreadPost{}
	}

	var thread models.ThreadPost
	err = json.Unmarshal(data, &thread)
	if err != nil {
		logs.Error("Error unmarshalling thread: %v", err)
		return models.ThreadPost{}
	}

	// Fetch all posts related to the thread
	posts := GetPosts(thread.BoardID, thread.ThreadID)
	thread.Posts = posts

	return thread
}

func GetPost(threadID, postID string) models.Posts {
	data, err := Cache.Get("post:" + threadID + "#" + postID)
	if err != nil {
		logs.Error("Error getting post: %v", err)
		return models.Posts{}
	}

	var post models.Posts
	err = json.Unmarshal(data, &post)
	if err != nil {
		logs.Error("Error unmarshalling post: %v", err)
		return models.Posts{}
	}

	return post
}
func GetBoards() []models.Board {
	boards := []models.Board{}

	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 5 && key[:5] == "board" {
			board := GetBoard(key[6:])
			boards = append(boards, board)
		}
	}

	return boards
}

func GetThreadsHandler(c echo.Context) error {
	boardID := c.Param("b")
	threads := GetThreads(boardID, c)
	if !auth.AdminCheck(c) || !auth.ModeratorCheck(c) || !auth.JannyCheck(c, boardID) {
		for i := range threads {
			threads[i].IP = ""
		}
	}
	return c.JSON(200, threads)
}

func GetThreadHandler(c echo.Context) error {
	threadID := c.Param("t")
	thread := GetThread(threadID)
	if !auth.AdminCheck(c) || !auth.ModeratorCheck(c) || !auth.JannyCheck(c, thread.BoardID) {
		thread.IP = ""
		for i := range thread.Posts {
			thread.Posts[i].IP = ""
		}
	}
	return c.JSON(200, thread)
}

func GetThreads(boardID string, e echo.Context) []models.ThreadPost {
	threads := []models.ThreadPost{}

	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 6 && key[:6] == "thread" {
			thread := GetThread(key[7:])
			if thread.BoardID == boardID {
				for i := range thread.Posts {
					postsmin := models.PostsMin{
						ID:             thread.Posts[i].ID,
						BoardID:        thread.Posts[i].BoardID,
						ParentID:       thread.Posts[i].ParentID,
						PostID:         thread.Posts[i].PostID,
						PartialContent: thread.Posts[i].PartialContent,
						Thumbnail:      thread.Posts[i].Thumbnail,
						Timestamp:      thread.Posts[i].Timestamp,
					}
					thread.PostsMin = append(thread.PostsMin, postsmin)
				}
				thread.Posts = nil
				threads = append(threads, thread)
			}
		}
	}

	return threads
}

func GetPosts(boardID string, threadID string) []models.Posts {
	posts := []models.Posts{}

	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 5 && key[:5] == "post:" {
			parts := strings.Split(key[5:], "#")
			if len(parts) == 2 && parts[0] == threadID {
				post := GetPost(parts[0], parts[1])
				posts = append(posts, post)
			}
		}
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Timestamp < posts[j].Timestamp
	})

	return posts
}
func AddThreadToCache(thread models.ThreadPost) {
	data, err := json.Marshal(thread)
	if err != nil {
		logs.Error("Error marshalling thread: %v", err)
		return
	}
	Cache.Set("thread:"+thread.ThreadID, data)
}

func AddPostToCache(post models.Posts) {
	data, err := json.Marshal(post)
	if err != nil {
		logs.Error("Error marshalling post: %v", err)
		return
	}
	Cache.Set("post:"+post.ParentID, data)
}

func AddBoardToCache(board models.Board) {
	data, err := json.Marshal(board)
	if err != nil {
		logs.Error("Error marshalling board: %v", err)
		return
	}
	Cache.Set("board:"+board.BoardID, data)
}

func CacheNews(news models.News) {
	data, err := json.Marshal(news)
	if err != nil {
		logs.Error("Error marshalling news: %v", err)
		return
	}
	Cache.Set("news:"+news.ID, data)
}

func GetNews(id string) models.News {
	data, err := Cache.Get("news:" + id)
	if err != nil {
		logs.Error("Error getting news: %v", err)
		return models.News{}
	}

	var news models.News
	err = json.Unmarshal(data, &news)
	if err != nil {
		logs.Error("Error unmarshalling news: %v", err)
		return models.News{}
	}

	return news
}

func GetAllNews() []models.News {
	news := []models.News{}

	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 5 && key[:5] == "news:" {
			n := GetNews(key[6:])
			news = append(news, n)
		}
	}

	return news
}

func GetNewsHandler(c echo.Context) error {
	news := GetAllNews()
	return c.JSON(200, news)
}

func CacheLatestPosts(posts []models.RecentPosts) {
	recents := models.GetLatestPosts(10)
	for _, post := range recents {
		data, err := json.Marshal(post)
		if err != nil {
			logs.Error("Error marshalling post: %v", err)
			continue
		}
		Cache.Set("recent:"+post.PostID, data)
	}
}
func GetLatestPost(id string) models.RecentPosts {
	data, err := Cache.Get("recent:" + id)
	if err != nil {
		logs.Error("Error getting recent post: %v", err)
		return models.RecentPosts{}
	}

	var post models.RecentPosts
	err = json.Unmarshal(data, &post)
	if err != nil {
		logs.Error("Error unmarshalling recent post: %v", err)
		return models.RecentPosts{}
	}

	return post
}

func GetAllLatestPosts() []models.RecentPosts {
	posts := []models.RecentPosts{}

	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 7 && key[:7] == "recent:" {
			p := GetLatestPost(key[8:])
			posts = append(posts, p)
		}
	}

	return posts
}

func GetLatestPostsHandler(c echo.Context) error {
	posts := GetAllLatestPosts()
	return c.JSON(200, posts)
}

func ResetBoards() {
	b.Start()
	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 5 && key[:5] == "board" {
			Cache.Delete(key)
		}
	}
	b.Close()
}

func ResetThreads() {
	b.Start()
	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 6 && key[:6] == "thread" {
			Cache.Delete(key)
		}
	}
	b.Close()
}

func ResetPosts() {
	b.Start()
	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 5 && key[:5] == "post:" {
			Cache.Delete(key)
		}
	}
	b.Close()
}

func ResetNews() {
	b.Start()
	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 5 && key[:5] == "news:" {
			Cache.Delete(key)
		}
	}
	b.Close()
}

func ResetLatestPosts() {
	b.Start()
	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		if len(key) > 7 && key[:7] == "recent:" {
			Cache.Delete(key)
		}
	}
	b.Close()
}

func ResetCache() {
	b.Start()
	iterator := Cache.Iterator()
	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			continue
		}
		key := string(entry.Key())
		Cache.Delete(key)
	}
	b.Close()
}
