package stats

import (
	"context"
	"fmt"
	"log"

	"achan.moe/auth"
	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Stats = models.Stats{}
var db = database.DB_Main
var statsCollection = db.Collection("stats")

func init() {
	SetTotalSize(CalcTotalSize())
	SetTotalUsers()
}

func SetTotalSize(size int) {
	_, err := statsCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": 1},
		bson.M{"$set": bson.M{"total_size": size}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Fatal(err)
	}
}
func CalcTotalSize() int {
	boards := models.GetBoards()
	var grandTotalSize int

	for _, board := range boards {
		boardID := board.BoardID
		db := database.Client.Database(boardID)

		collections, err := db.ListCollectionNames(context.TODO(), bson.D{})
		if err != nil {
			logs.Info("Error listing collections for board %s: %v\n", boardID, err)
			continue
		}

		for _, collName := range collections {
			coll := db.Collection(collName)

			pipeline := mongo.Pipeline{
				{{"$project", bson.D{
					{"_id", 0},
					{"size", bson.D{{"$bsonSize", "$$ROOT"}}},
				}}},
			}

			cursor, err := coll.Aggregate(context.TODO(), pipeline)
			if err != nil {
				continue
			}

			defer cursor.Close(context.TODO())

			for cursor.Next(context.TODO()) {
				var result struct {
					Size int64 `bson:"size"`
				}
				if err := cursor.Decode(&result); err != nil {
					logs.Info("Error decoding BSON size for collection %s: %v\n", collName, err)
					continue
				}
				if result.Size < 0 {
					logs.Info("Negative BSON size for collection %s: %d\n", collName, result.Size)
					continue
				}
				if result.Size > 0 {
					logs.Info("BSON size for collection %s: %d\n", collName, result.Size)
					grandTotalSize += int(result.Size)
				} else {
				}
			}
		}
	}

	fmt.Printf("Total BSON size across all boards: %d bytes\n", grandTotalSize)

	return grandTotalSize
}

func SetTotalUsers() {
	users := auth.GetTotalUsers()
	_, err := statsCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": 1},
		bson.M{"$set": bson.M{"total_users": users}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func SetTotalPosts(count int) {
	_, err := statsCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": 1},
		bson.M{"$set": bson.M{"post_count": count}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Fatal(err)
	}
}
