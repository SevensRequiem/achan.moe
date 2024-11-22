package database

import (
	"context"
	"os"
	"time"

	"achan.moe/logs"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB_Main *mongo.Database
var DB_Boards *mongo.Database
var DB_Actions *mongo.Database
var DB_Archive *mongo.Database
var DB_Users *mongo.Database
var Client *mongo.Client
var MySQL *gorm.DB

func init() {
	err := godotenv.Load()
	if err != nil {
		logs.Error("Error loading .env file")
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		logs.Error("Error creating MongoDB client")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		logs.Error("Error connecting to MongoDB")
		return
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		logs.Error("Error pinging MongoDB")
		return
	}

	Client = client
	DB_Main = client.Database("achan")
	DB_Actions = client.Database("actions")

	logs.Info("Connected to MongoDB")

	// create collections
	if DB_Main != nil {
		DB_Main.CreateCollection(context.Background(), "users")
		DB_Main.CreateCollection(context.Background(), "reports")
		DB_Main.CreateCollection(context.Background(), "sessions")
		DB_Main.CreateCollection(context.Background(), "settings")
		DB_Main.CreateCollection(context.Background(), "bans")
		DB_Main.CreateCollection(context.Background(), "old_bans")
		DB_Main.CreateCollection(context.Background(), "boards")
		DB_Main.CreateCollection(context.Background(), "recent_posts")
		DB_Main.CreateCollection(context.Background(), "data")

		filter := bson.M{"post_count": bson.M{"$exists": true}}
		var result bson.M
		err := DB_Main.Collection("data").FindOne(context.Background(), filter).Decode(&result)
		if err == mongo.ErrNoDocuments {
			_, err = DB_Main.Collection("data").InsertOne(context.Background(), bson.M{"post_count": 0})
			if err != nil {
				logs.Fatal("Error inserting post count: %v", err)
			}
		} else if err != nil {
			logs.Fatal("Error checking post count: %v", err)
		}

		DB_Main.CreateCollection(context.Background(), "news")
		DB_Main.CreateCollection(context.Background(), "hits")
	} else {
		logs.Error("DB_Main is nil")
	}

	if DB_Actions != nil {
		DB_Actions.CreateCollection(context.Background(), "actions")
		DB_Actions.CreateCollection(context.Background(), "reports")
		DB_Actions.CreateCollection(context.Background(), "sessions")
		DB_Actions.CreateCollection(context.Background(), "settings")
		DB_Actions.CreateCollection(context.Background(), "bans")
		DB_Actions.CreateCollection(context.Background(), "old_bans")
		DB_Actions.CreateCollection(context.Background(), "boards")
		DB_Actions.CreateCollection(context.Background(), "recent_posts")
		DB_Actions.CreateCollection(context.Background(), "data")
		DB_Actions.CreateCollection(context.Background(), "news")
		DB_Actions.CreateCollection(context.Background(), "hits")
		DB_Actions.CreateCollection(context.Background(), "users")
	} else {
		logs.Error("DB_Actions is nil")
	}

	// mysql connection
	dsn := os.Getenv("MYSQL_URI")
	if dsn == "" {
		logs.Error("No MySQL DSN provided")
		return
	}
	MySQL, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logs.Error("Error opening MySQL connection: %v", err)
		return
	}

	sqlDB, err := MySQL.DB()
	if err != nil {
		logs.Error("Error getting SQL DB from GORM: %v", err)
		return
	}

	err = sqlDB.Ping()
	if err != nil {
		logs.Error("Error pinging MySQL: %v", err)
		MySQL = nil
		return
	}
}

func GetCollection(collection string) *mongo.Collection {
	return DB_Main.Collection(collection)
}

func Createboards() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := DB_Main.Collection("boards").Find(ctx, bson.M{})
	if err != nil {
		logs.Error("Error finding boards")
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var board string
		if err := cursor.Decode(&board); err != nil {
			logs.Error("Error decoding board")
			continue
		}
		Client.Database(board)
	}
	if err := cursor.Err(); err != nil {
		logs.Error("Cursor error")
	}
}

func Migrateboardsfromsql() {
	if MySQL == nil {
		logs.Error("MySQL connection is not initialized")
		return
	}

	var boards []struct {
		BoardID     string `gorm:"column:board_id"`
		Name        string
		Description string
		PostCount   int
		ImageOnly   bool
		Locked      bool
		Archived    bool
		LatestPosts int
		Pages       int
	}
	result := MySQL.Table("boards").Find(&boards)
	if result.Error != nil {
		logs.Error("Error querying MySQL: %v", result.Error)
		return
	}

	for _, board := range boards {
		// Check if the board already exists in MongoDB
		var existingBoard bson.M
		err := DB_Main.Collection("boards").FindOne(context.Background(), bson.M{"board_id": board.BoardID}).Decode(&existingBoard)
		if err != nil && err != mongo.ErrNoDocuments {
			logs.Error("Error checking if board exists: %v", err)
			continue
		}
		if err == nil {
			logs.Info("Board already exists: %v", board.BoardID)
			continue
		}

		// Create collections for the board
		err = Client.Database(board.BoardID).CreateCollection(context.Background(), "thumbs")
		if err != nil {
			logs.Error("Error creating thumbs collection: %v", err)
			continue
		}
		err = Client.Database(board.BoardID).CreateCollection(context.Background(), "images")
		if err != nil {
			logs.Error("Error creating images collection: %v", err)
			continue
		}

		// Insert the board into MongoDB
		_, err = DB_Main.Collection("boards").InsertOne(context.Background(), bson.M{
			"boardid":      board.BoardID,
			"name":         board.Name,
			"description":  board.Description,
			"post_count":   board.PostCount,
			"image_only":   board.ImageOnly,
			"locked":       board.Locked,
			"archived":     board.Archived,
			"latest_posts": board.LatestPosts,
			"pages":        board.Pages,
		})
		if err != nil {
			logs.Error("Error inserting board: %v", err)
		}
	}
}

func Migratebansfromsql() {
	if MySQL == nil {
		logs.Error("MySQL connection is not initialized")
		return
	}

	var bans []struct {
		ID     string
		IP     string
		Reason string
		Time   int64
		Board  string
	}
	result := MySQL.Table("bans").Find(&bans)
	if result.Error != nil {
		logs.Error("Error querying MySQL: %v", result.Error)
		return
	}

	for _, ban := range bans {
		// Check if the ban already exists in MongoDB
		var existingBan bson.M
		err := DB_Main.Collection("bans").FindOne(context.Background(), bson.M{"id": ban.ID}).Decode(&existingBan)
		if err == nil {
			logs.Info("Ban already exists: %v", ban.ID)
			continue
		}

		// Insert the ban into MongoDB
		_, err = DB_Main.Collection("bans").InsertOne(context.Background(), bson.M{
			"id":     ban.ID,
			"ip":     ban.IP,
			"reason": ban.Reason,
			"time":   ban.Time,
			"board":  ban.Board,
		})
		if err != nil {
			logs.Error("Error inserting ban: %v", err)
		}
	}
}

func Migratemisc() {
	if MySQL == nil {
		logs.Error("MySQL connection is not initialized")
		return
	}

	var visits []struct {
		ID   string `gorm:"column:id"`
		Hits int    `gorm:"column:hits"`
	}
	hits := MySQL.Table("hit_counters").Find(&visits)
	if hits.Error != nil {
		logs.Error("Error querying MySQL: %v", hits.Error)
		return
	}

	var pcount []struct {
		ID        string `gorm:"column:id"`
		PostCount int    `gorm:"column:post_count"`
	}
	postcount := MySQL.Table("post_counters").Find(&pcount)
	if postcount.Error != nil {
		logs.Error("Error querying MySQL: %v", postcount.Error)
		return
	}

	DB_Main.Collection("hits").Drop(context.Background())
	DB_Main.CreateCollection(context.Background(), "hits")
	if len(visits) > 0 {
		DB_Main.Collection("hits").InsertOne(context.Background(), bson.M{"hits": visits[0].Hits})
	}

	DB_Main.Collection("data").Drop(context.Background())
	DB_Main.CreateCollection(context.Background(), "data")
	if len(pcount) > 0 {
		DB_Main.Collection("data").InsertOne(context.Background(), bson.M{"post_count": pcount[0].PostCount})
	}
}

func Drops() {
	// get all boards
	cursor, err := DB_Main.Collection("boards").Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error finding boards")
		return
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var board bson.M
		if err := cursor.Decode(&board); err != nil {
			logs.Error("Error decoding board")
			continue
		}
		Client.Database(board["boardid"].(string)).Drop(context.Background())
	}
	if err := cursor.Err(); err != nil {
		logs.Error("Cursor error")
	}

	Client.Database("achan").Collection("boards").Drop(context.Background())
	DB_Main.Collection("users").Drop(context.Background())
	DB_Main.Collection("reports").Drop(context.Background())
	DB_Main.Collection("sessions").Drop(context.Background())
	DB_Main.Collection("settings").Drop(context.Background())
	DB_Main.Collection("bans").Drop(context.Background())
	DB_Main.Collection("old_bans").Drop(context.Background())
	DB_Main.Collection("boards").Drop(context.Background())
	DB_Main.Collection("recent_posts").Drop(context.Background())
	DB_Main.Collection("data").Drop(context.Background())
	DB_Main.Collection("news").Drop(context.Background())
	DB_Main.Collection("hits").Drop(context.Background())
}
