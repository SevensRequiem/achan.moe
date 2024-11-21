package models

import (
	"context"
	"sort"

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
	Pages       int    `bson:"pages"`
}

type ThreadPost struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	BoardID        string             `bson:"boardid"`
	ThreadID       string             `bson:"thread_id"`
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

type PostCounter struct {
	ID        int   `bson:"_id,omitempty"`
	PostCount int64 `bson:"post_count"`
}

// USER MODEL

type User struct {
	ID              uint   `bson:"_id"`
	UUID            string `bson:"UUID"`
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
	Date    string `json:"date"`
	Author  string `json:"author"`
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
