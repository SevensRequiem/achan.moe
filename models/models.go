package models

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"achan.moe/database"
	"achan.moe/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Board struct {
	BoardID     string `bson:"boardid"`
	Name        string `bson:"name"`
	Description string `bson:"description"`
	PostCount   int64  `bson:"post_count"`
	ImageOnly   bool   `bson:"image_only"`
	Locked      bool   `bson:"locked"`
	Archived    bool   `bson:"archived"`
	LatestPosts bool   `bson:"latest_posts"`
}

type ThreadPost struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	BoardID        string             `bson:"boardid"`
	ThreadID       string             `bson:"thread_id"`
	PostID         string             `bson:"post_id"`
	Content        string             `bson:"content"`
	PartialContent string             `bson:"partial_content"`
	Image          string             `bson:"image"`
	Thumbnail      string             `bson:"thumbnail"`
	Subject        string             `bson:"subject"`
	Author         string             `bson:"author"`
	TrueUser       string             `bson:"true_user"`
	Timestamp      int64              `bson:"timestamp"`
	IP             string             `bson:"ip"`
	Sticky         bool               `bson:"sticky"`
	Locked         bool               `bson:"locked"`
	PostCount      int                `bson:"post_count"`
	ReportCount    int                `bson:"report_count"`
	PostNumber     int64              `bson:"post_number"`
	Posts          []Posts            `bson:"posts"`
	PostsMin       []PostsMin         `bson:"posts_min"`
}

type Posts struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	BoardID        string             `bson:"boardid"`
	ParentID       string             `bson:"parent_id"`
	PostID         string             `bson:"post_id"`
	Content        string             `bson:"content"`
	PartialContent string             `bson:"partial_content"`
	Image          string             `bson:"image"`
	Thumbnail      string             `bson:"thumb"`
	Subject        string             `bson:"subject"`
	Author         string             `bson:"author"`
	TrueUser       string             `bson:"true_user"`
	Timestamp      int64              `bson:"timestamp"`
	IP             string             `bson:"ip"`
	ReportCount    int                `bson:"report_count"`
	PostNumber     int64              `bson:"post_number"`
}

type PostsMin struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	BoardID        string             `bson:"boardid"`
	ParentID       string             `bson:"parent_id"`
	PostID         string             `bson:"post_id"`
	PartialContent string             `bson:"partial_content"`
	Thumbnail      string             `bson:"thumb"`
	Timestamp      int64              `bson:"timestamp"`
}

type RecentThreads []ThreadPost

type Image struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Image    []byte             `bson:"image"`
	Filetype string             `bson:"filetype"`
	Size     int64              `bson:"size"`
	Height   int                `bson:"height"`
	Width    int                `bson:"width"`
}

type Recents struct {
	ID     int64  `bson:"_id,omitempty"`
	PostID string `bson:"post_id"`
}

type Stats struct {
	ID          int `bson:"_id,omitempty"`
	PostCount   int `bson:"post_count"`
	ThreadCount int `bson:"thread_count"`
	BoardCount  int `bson:"board_count"`
	BanCount    int `bson:"ban_count"`
	TotalSize   int `bson:"total_size"`
	TotalUsers  int `bson:"total_users"`
	TotalHits   int `bson:"hit_count"`
	TotalImages int `bson:"image_count"`
	TotalFiles  int `bson:"file_count"`
}

// USER MODEL

type User struct {
	ID              uint   `bson:"_id"`
	Username        string `bson:"username"`
	Password        string `bson:"password"`
	Groups          Group  `bson:"groups"`
	DateCreated     string `bson:"date_created"`
	LastLogin       string `bson:"last_login"`
	DoesExist       bool   `bson:"does_exist"`
	Premium         bool   `bson:"premium"`
	Permanent       bool   `bson:"permanent"`
	Banned          bool   `bson:"banned"`
	Email           string `bson:"email"`
	TransactionID   string `bson:"transaction_id"`
	PlusReputation  int    `bson:"plus_reputation"`
	MinusReputation int    `bson:"minus_reputation"`
	Posts           int    `bson:"posts"`
	Threads         int    `bson:"threads"`
	DisplayName     string `bson:"display_name"`
}

type Group struct {
	Admin     bool        `json:"admin"`
	Moderator bool        `json:"moderator"`
	Janny     JannyBoards `json:"janny"`
}

type JannyBoards struct {
	Boards []string `json:"boards"`
}

type News struct {
	ID      string `json:"id" bson:"_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Date    int64  `json:"date"`
	Author  string `json:"author"`
}

// bans

type Bans struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Status    string             `bson:"status"`
	IP        string             `bson:"ip"`
	Reason    string             `bson:"reason"`
	Username  string             `bson:"username"`
	UserID    uint               `bson:"userid"`
	Timestamp string             `bson:"timestamp"`
	Expires   string             `bson:"expires"`
}

// actions

type AnnouncementActions struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	BoardID   string             `json:"boardid" bson:"boardid"`
	Content   string             `json:"content" bson:"content"`
	Timestamp int64              `json:"timestamp" bson:"timestamp"`
	UserID    uint               `json:"user" bson:"user"`
	IP        string             `json:"ip" bson:"ip"`
	Action    string             `json:"action" bson:"action"`
}

type BanActions struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Status    string             `bson:"status" json:"status"`
	IP        string             `bson:"ip" json:"ip"`
	Reason    string             `bson:"reason" json:"reason"`
	Username  string             `bson:"username" json:"username"`
	UserID    uint               `bson:"userid" json:"userid"`
	Timestamp string             `bson:"timestamp" json:"timestamp"`
	Expires   string             `bson:"expires" json:"expires"`
}
type UnbanActions struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Status    string             `bson:"status" json:"status"`
	IP        string             `bson:"ip" json:"ip"`
	Reason    string             `bson:"reason" json:"reason"`
	Username  string             `bson:"username" json:"username"`
	UserID    uint               `bson:"userid" json:"userid"`
	Timestamp string             `bson:"timestamp" json:"timestamp"`
	Expires   string             `bson:"expires" json:"expires"`
}

type BoardActions struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	BoardID   string             `json:"boardid" bson:"boardid"`
	BoardName string             `json:"boardname" bson:"boardname"`
	Username  string             `json:"user" bson:"user"`
	UserID    uint               `json:"userid" bson:"userid"`
	Action    string             `json:"action" bson:"action"`
	Timestamp int64              `json:"timestamp" bson:"timestamp"`
	IP        string             `json:"ip" bson:"ip"`
}
type ThreadActions struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	BoardID        string             `json:"boardid" bson:"boardid"`
	ThreadID       string             `json:"threadid" bson:"threadid"`
	PostID         string             `json:"postid" bson:"postid"`
	Username       string             `json:"user" bson:"user"`
	UserID         uint               `json:"userid" bson:"userid"`
	Action         string             `json:"action" bson:"action"`
	Timestamp      int64              `json:"timestamp" bson:"timestamp"`
	IP             string             `json:"ip" bson:"ip"`
	Subject        string             `json:"subject" bson:"subject"`
	PartialContent string             `json:"partial_content" bson:"partial_content"`
	Image          string             `json:"image" bson:"image"`
	Thumbnail      string             `json:"thumb" bson:"thumb"`
}
type PostActions struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	BoardID        string             `json:"boardid" bson:"boardid"`
	ThreadID       string             `json:"threadid" bson:"threadid"`
	PostID         string             `json:"postid" bson:"postid"`
	Username       string             `json:"user" bson:"user"`
	UserID         uint               `json:"userid" bson:"userid"`
	Action         string             `json:"action" bson:"action"`
	Timestamp      int64              `json:"timestamp" bson:"timestamp"`
	IP             string             `json:"ip" bson:"ip"`
	Subject        string             `json:"subject" bson:"subject"`
	PartialContent string             `json:"partial_content" bson:"partial_content"`
	Image          string             `json:"image" bson:"image"`
	Thumbnail      string             `json:"thumb" bson:"thumb"`
}
type ConfigActions struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	AdminUsername string             `json:"admin_user" bson:"admin_user"`
	Action        string             `json:"action" bson:"action"`
	Timestamp     int64              `json:"timestamp" bson:"timestamp"`
	IP            string             `json:"ip" bson:"ip"`
}

type DataActions struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Action    string             `json:"action" bson:"action"`
	Timestamp int64              `json:"timestamp" bson:"timestamp"`
	IP        string             `json:"ip" bson:"ip"`
}

type NewsActions struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Title     string             `json:"title" bson:"title"`
	Content   string             `json:"content" bson:"content"`
	Timestamp int64              `json:"timestamp" bson:"timestamp"`
	Author    string             `json:"author" bson:"author"`
	IP        string             `json:"ip" bson:"ip"`
}

type UserActions struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Username  string             `json:"username" bson:"username"`
	UserID    uint               `json:"userid" bson:"userid"`
	Action    string             `json:"action" bson:"action"`
	Timestamp int64              `json:"timestamp" bson:"timestamp"`
	IP        string             `json:"ip" bson:"ip"`
}

// FUNCS

func GetBoards() []Board {
	db := database.DB_Main.Collection("boards")
	ctx := context.Background()
	cursor, err := db.Find(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding boards: %v", err)
		return nil
	}
	defer cursor.Close(ctx)

	var boards []Board
	for cursor.Next(ctx) {
		var board Board
		if err := cursor.Decode(&board); err != nil {
			logs.Error("Error decoding board: %v", err)
			return nil
		}
		boards = append(boards, board)
	}

	if err := cursor.Err(); err != nil {
		logs.Error("Cursor error: %v", err)
		return nil
	}

	return boards
}

func GetLatestPosts(n int) RecentThreads {
	db := database.DB_Main.Collection("recent_posts")
	ctx := context.Background()
	cursor, err := db.Find(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding recent posts: %v", err)
		return nil
	}
	defer cursor.Close(ctx)
	var recentThreads RecentThreads
	for cursor.Next(ctx) {
		var post ThreadPost
		if err := cursor.Decode(&post); err != nil {
			logs.Error("Error decoding post: %v", err)
			return nil
		}
		recentThreads = append(recentThreads, post)
	}
	return recentThreads
}
func GetThreads(boardID string) []ThreadPost {
	db := database.Client.Database(boardID)
	ctx := context.Background()
	cursor, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding collections: %v", err)
		return nil
	}
	defer cursor.Close(ctx)
	var threads []struct {
		ThreadPost        ThreadPost
		LastPostTimestamp int64
	}
	for cursor.Next(ctx) {
		var collection struct {
			Name string `bson:"name"`
		}
		if err := cursor.Decode(&collection); err != nil {
			logs.Error("Error decoding collection: %v", err)
			return nil
		}
		if collection.Name == "thumbs" || collection.Name == "images" || collection.Name == "banners" {
			continue
		}
		var threadPost ThreadPost
		err := db.Collection(collection.Name).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", 1}})).Decode(&threadPost)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			logs.Error("Error finding first document in collection %s: %v", collection.Name, err)
			return nil
		}
		var lastPost ThreadPost
		err = db.Collection(collection.Name).FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{"timestamp", -1}})).Decode(&lastPost)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			logs.Error("Error finding last document in collection %s: %v", collection.Name, err)
			return nil
		}

		// Fetch posts for the thread
		var posts []Posts
		cursor, err := db.Collection(collection.Name).Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{"timestamp", 1}}))
		if err != nil {
			logs.Error("Error finding posts in collection %s: %v", collection.Name, err)
			return nil
		}
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var post Posts
			if err := cursor.Decode(&post); err != nil {
				logs.Error("Error decoding post: %v", err)
				return nil
			}
			posts = append(posts, post)
		}
		threadPost.Posts = posts

		threads = append(threads, struct {
			ThreadPost        ThreadPost
			LastPostTimestamp int64
		}{
			ThreadPost:        threadPost,
			LastPostTimestamp: lastPost.Timestamp,
		})
	}

	sort.SliceStable(threads, func(i, j int) bool {
		if threads[i].ThreadPost.Sticky != threads[j].ThreadPost.Sticky {
			return threads[i].ThreadPost.Sticky
		}
		return threads[i].LastPostTimestamp > threads[j].LastPostTimestamp
	})

	var sortedThreads []ThreadPost
	for _, thread := range threads {
		sortedThreads = append(sortedThreads, thread.ThreadPost)
	}

	return sortedThreads
}

type Announcement struct {
	ID        string `json:"id" bson:"_id"`
	BoardID   string `json:"boardid" bson:"boardid"`
	Content   string `json:"content" bson:"content"`
	Timestamp int64  `json:"timestamp" bson:"timestamp"`
	User      string `json:"user" bson:"user"`
	UserID    string `json:"userid" bson:"userid"`
}

func GetAllAnnouncements() []Announcement {
	var announcements []Announcement
	db := database.DB_Main
	cursor, err := db.Collection("announcements").Find(context.Background(), bson.M{})
	if err != nil {
		return nil
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var announcement Announcement
		err := cursor.Decode(&announcement)
		if err != nil {
			return nil
		}
		announcements = append(announcements, announcement)
	}
	return announcements
}

func GetAnnouncementsByBoard(boardID string) ([]Announcement, error) {
	var announcements []Announcement
	db := database.DB_Main
	cursor, err := db.Collection("announcements").Find(context.Background(), bson.M{"boardid": boardID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var announcement Announcement
		err := cursor.Decode(&announcement)
		if err != nil {
			return nil, err
		}
		announcements = append(announcements, announcement)
	}
	return announcements, nil
}

func GetAnnouncementByType(announcementType string) *Announcement {
	var announcement Announcement
	db := database.DB_Main
	err := db.Collection("announcements").FindOne(context.Background(), bson.M{"type": announcementType}).Decode(&announcement)
	if err != nil {
		return nil
	}
	return &announcement
}

func AddAnnouncement(boardID, content, user string) error {
	announcement := Announcement{
		BoardID:   boardID,
		Content:   content,
		Timestamp: time.Now().Unix(),
		User:      user,
	}
	_, err := database.DB_Main.Collection("announcements").InsertOne(context.Background(), announcement)
	return err
}

func DeleteAnnouncement(id string) error {
	_, err := database.DB_Main.Collection("announcements").DeleteOne(context.Background(), bson.M{"_id": id})
	return err
}

func GetAllNews() ([]News, error) {
	cursor, err := database.DB_Main.Collection("news").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error finding news: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var allNews []News
	for cursor.Next(context.Background()) {
		var news News
		if err := cursor.Decode(&news); err != nil {
			logs.Error("Error decoding news: %v", err)
			continue
		}
		allNews = append(allNews, news)
	}

	if err := cursor.Err(); err != nil {
		logs.Error("Cursor error: %v", err)
		return nil, err
	}

	return allNews, nil
}

func GetAllBans() ([]Bans, error) {
	cursor, err := database.DB_Main.Collection("bans").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error finding bans: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var allBans []Bans
	for cursor.Next(context.Background()) {
		var ban Bans
		if err := cursor.Decode(&ban); err != nil {
			logs.Error("Error decoding ban: %v", err)
			continue
		}
		allBans = append(allBans, ban)
	}

	if err := cursor.Err(); err != nil {
		logs.Error("Cursor error: %v", err)
		return nil, err
	}

	return allBans, nil
}

func GetAllStats() ([]byte, error) {
	var stats Stats
	err := database.DB_Main.Collection("stats").FindOne(context.Background(), bson.M{"_id": 1}).Decode(&stats)
	if err != nil {
		logs.Error("Error finding stats: %v", err)
		return nil, err
	}
	fmt.Println("Stats: ", stats)
	jsonData, err := json.Marshal(stats)
	if err != nil {
		logs.Error("Error marshalling stats: %v", err)
		return nil, err
	}
	return jsonData, nil
}
