package banners

import (
	"context"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"

	"achan.moe/database"
	"achan.moe/logs"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Banner struct {
	ID       string `bson:"_id"`
	Filetype string `bson:"filetype"`
	Size     int64  `bson:"size"`
	Image    []byte `bson:"image"`
}

func GetGlobalBanner(c echo.Context, fileid string) error {
	objectID, err := primitive.ObjectIDFromHex(fileid)
	if err != nil {
		logs.Error("Invalid fileid: %v", err)
		return c.JSON(http.StatusBadRequest, "Invalid fileid")
	}

	db := database.Client.Database("achan")
	collection := db.Collection("banners")

	var banner Banner
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&banner)
	if err != nil {
		logs.Error("Error fetching global banner: %v", err)
		return err
	}

	c.Response().Header().Set("Content-Type", banner.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(banner.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	_, err = c.Response().Write(banner.Image)
	if err != nil {
		logs.Error("Error writing global banner image: %v", err)
	}
	return err
}

func GetLocalBanner(c echo.Context, boardid string, fileid string) error {
	objectID, err := primitive.ObjectIDFromHex(fileid)
	if err != nil {
		logs.Error("Invalid fileid: %v", err)
		return c.JSON(http.StatusBadRequest, "Invalid fileid")
	}

	db := database.Client.Database(boardid)
	collection := db.Collection("banners")

	var banner Banner
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&banner)
	if err != nil {
		logs.Error("Error fetching local banner: %v", err)
		return err
	}

	c.Response().Header().Set("Content-Type", banner.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(banner.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	_, err = c.Response().Write(banner.Image)
	if err != nil {
		logs.Error("Error writing local banner image: %v", err)
	}
	return err
}

// GetRandomBanner returns a random banner based on the board ID.
func GetRandomBanner(c echo.Context, boardid string) error {
	randNum := rand.Intn(2)
	var err error
	var errnum int

	for errnum <= 3 {
		if randNum == 0 {
			err = GetRandomLocalBanner(c, boardid)
			if err != nil {
				logs.Error("Failed to get random local banner, trying global: %v", err)
				errnum++
				randNum = 1
			} else {
				return nil
			}
		} else {
			err = GetRandomGlobalBanner(c)
			if err != nil {
				logs.Error("Failed to get random global banner, trying local: %v", err)
				errnum++
				randNum = 0
			} else {
				return nil
			}
		}
	}

	logs.Error("Exceeded maximum retries for getting a random banner")
	return errors.New("failed to get a random banner after multiple attempts")
}

func GetRandomGlobalBanner(c echo.Context) error {
	db := database.Client.Database("achan")
	collection := db.Collection("banners")

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

func ListGlobalBanners(c echo.Context) error {
	db := database.Client.Database("achan")
	collection := db.Collection("banners")

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error fetching global banners: %v", err)
		return err
	}
	defer cursor.Close(context.Background())

	var banners []Banner
	if err = cursor.All(context.Background(), &banners); err != nil {
		logs.Error("Error decoding global banners: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, banners)
}
func ListLocalBanners(c echo.Context) error {
	boardid := c.Param("boardid")
	db := database.Client.Database(boardid)
	collection := db.Collection("banners")

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		logs.Error("Error fetching local banners: %v", err)
		return err
	}
	defer cursor.Close(context.Background())

	var banners []Banner
	if err = cursor.All(context.Background(), &banners); err != nil {
		logs.Error("Error decoding local banners: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, banners)
}

func DeleteGlobalBanner(c echo.Context, fileid string) error {
	objectID, err := primitive.ObjectIDFromHex(fileid)
	if err != nil {
		logs.Error("Invalid fileid: %v", err)
		return c.JSON(http.StatusBadRequest, "Invalid fileid")
	}

	db := database.Client.Database("achan")
	collection := db.Collection("banners")

	result, err := collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		logs.Error("Error deleting global banner: %v", err)
		return err
	}
	if result.DeletedCount == 0 {
		logs.Error("No global banner found with ID: %s", fileid)
		return c.JSON(http.StatusNotFound, "No global banner found")
	}

	return c.JSON(http.StatusOK, "Global banner deleted successfully")
}
func DeleteLocalBanner(c echo.Context, boardid string, fileid string) error {
	objectID, err := primitive.ObjectIDFromHex(fileid)
	if err != nil {
		logs.Error("Invalid fileid: %v", err)
		return c.JSON(http.StatusBadRequest, "Invalid fileid")
	}

	db := database.Client.Database(boardid)
	collection := db.Collection("banners")

	result, err := collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		logs.Error("Error deleting local banner: %v", err)
		return err
	}
	if result.DeletedCount == 0 {
		logs.Error("No local banner found with ID: %s", fileid)
		return c.JSON(http.StatusNotFound, "No local banner found")
	}

	return c.JSON(http.StatusOK, "Local banner deleted successfully")
}

func DeleteBannerHandler(c echo.Context) error {
	boardid := c.FormValue("board_id")
	fileid := c.FormValue("banner_id")

	if boardid == "global" {
		return DeleteGlobalBanner(c, fileid)
	}
	return DeleteLocalBanner(c, boardid, fileid)
}
