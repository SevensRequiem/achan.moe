package banners

import (
	"context"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"achan.moe/database"
	"achan.moe/logs"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

type Banner struct {
	Filetype string `bson:"filetype"`
	Size     int64  `bson:"size"`
	Image    []byte `bson:"image"`
}

// GetRandomBanner returns a random banner based on the board ID.
func GetRandomBanner(c echo.Context, boardid string) error {
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(2) == 0 {
		logs.Debug("Getting global banner")
		err := GetRandomGlobalBanner(c)
		if err != nil {
			logs.Error("Error getting global banner: %v", err)
			return err
		}
		return nil
	}
	logs.Debug("Getting local banner for board %s", boardid)
	err := GetRandomLocalBanner(c, boardid)
	if err != nil {
		logs.Error("Error getting local banner: %v", err)
		return err
	}
	return nil
}

// GetRandomGlobalBanner returns a random global banner.
func GetRandomGlobalBanner(c echo.Context) error {
	db := database.Client.Database("achan")
	collection := db.Collection("banners")

	// Use MongoDB aggregation to randomly select one document
	pipeline := []bson.M{
		{"$sample": bson.M{"size": 1}},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		logs.Error("Error aggregating global banners: %v", err)
		return err
	}
	defer cursor.Close(context.Background())

	var banners []Banner
	if err = cursor.All(context.Background(), &banners); err != nil {
		logs.Error("Error fetching global banners: %v", err)
		return err
	}
	if len(banners) == 0 {
		logs.Error("No global banners found")
		return errors.New("no global banners found")
	}

	banner := banners[0]
	c.Response().Header().Set("Content-Type", banner.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(banner.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	_, err = c.Response().Write(banner.Image)
	if err != nil {
		logs.Error("Error writing global banner image: %v", err)
	}
	return err
}

// GetRandomLocalBanner returns a random local banner for a given board ID.
func GetRandomLocalBanner(c echo.Context, boardid string) error {
	db := database.Client.Database(boardid)
	collection := db.Collection("banners")

	// Use MongoDB aggregation to randomly select one document
	pipeline := []bson.M{
		{"$sample": bson.M{"size": 1}},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		logs.Error("Error aggregating local banners: %v", err)
		return err
	}
	defer cursor.Close(context.Background())

	var banners []Banner
	if err = cursor.All(context.Background(), &banners); err != nil {
		logs.Error("Error fetching local banners: %v", err)
		return err
	}
	if len(banners) == 0 {
		logs.Error("No local banners found")
		return errors.New("no local banners found")
	}

	banner := banners[0]
	c.Response().Header().Set("Content-Type", banner.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(banner.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	_, err = c.Response().Write(banner.Image)
	if err != nil {
		logs.Error("Error writing local banner image: %v", err)
	}
	return err
}

func UploadBanner(c echo.Context) error {
	boardid := c.FormValue("boardid")
	filename := c.FormValue("filename")
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(400, "Error uploading file")
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(500, "Error opening file")
	}
	defer src.Close()

	data, err := ioutil.ReadAll(src)
	if err != nil {
		return c.JSON(500, "Error reading file")
	}

	if boardid == "global" {
		return UploadGlobalBanner(filename, data)
	}
	return UploadLocalBanner(boardid, filename, data)
}

func UploadGlobalBanner(filename string, data []byte) error {
	db := database.Client.Database("achan")
	collection := db.Collection("banners")
	_, err := collection.InsertOne(context.Background(), bson.M{"filetype": http.DetectContentType(data), "size": int64(len(data)), "image": data})
	return err
}

func UploadLocalBanner(boardid string, filename string, data []byte) error {
	db := database.Client.Database(boardid)
	collection := db.Collection("banners")
	_, err := collection.InsertOne(context.Background(), bson.M{"filetype": http.DetectContentType(data), "size": int64(len(data)), "image": data})
	return err
}
