package boardimages

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"achan.moe/database"
	"achan.moe/logs"
	"achan.moe/models"
	"github.com/labstack/echo/v4"
	"github.com/nfnt/resize"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var client = database.Client

func GetImage(boardID string, imageID string) (models.Image, error) {
	db := client.Database(boardID)
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(imageID)
	if err != nil {
		logs.Error("Invalid image ID '%s': %v", imageID, err)
		return models.Image{}, errors.New("Invalid image ID")
	}

	var image models.Image
	err = db.Collection("images").FindOne(ctx, bson.M{"_id": objectID}).Decode(&image)
	if err != nil {
		logs.Error("Error finding image: %v", err)
		return models.Image{}, errors.New("Image not found")
	}

	return image, nil
}

func GetThumb(boardID string, thumbID string) (models.Image, error) {
	db := client.Database(boardID)
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(thumbID)
	if err != nil {
		logs.Error("Invalid thumbnail ID '%s': %v", thumbID, err)
		return models.Image{}, errors.New("Invalid thumbnail ID")
	}

	var image models.Image
	err = db.Collection("thumbs").FindOne(ctx, bson.M{"_id": objectID}).Decode(&image)
	if err != nil {
		logs.Error("Error finding thumbnail: %v", err)
		return models.Image{}, errors.New("Thumbnail not found")
	}

	return image, nil
}

func SaveThumb(boardID string, imageFile *multipart.FileHeader) (string, error) {
	db := client.Database(boardID)
	file, err := imageFile.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		logs.Error("Error decoding image: %v", err)
		return "", err
	}

	thumb := resize.Thumbnail(250, 250, img, resize.Lanczos3)
	var thumbBuffer bytes.Buffer
	err = png.Encode(&thumbBuffer, thumb)
	if err != nil {
		logs.Error("Error encoding thumbnail: %v", err)
		return "", err
	}

	imageDoc := models.Image{
		Image:    thumbBuffer.Bytes(),
		Filetype: "image/png",
		Size:     int64(thumbBuffer.Len()),
		Height:   thumb.Bounds().Dy(),
		Width:    thumb.Bounds().Dx(),
	}

	result, err := db.Collection("thumbs").InsertOne(context.Background(), imageDoc)
	if err != nil {
		logs.Error("Error inserting thumbnail: %v", err)
		return "", err
	}

	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func SaveImage(boardID string, imageFile *multipart.FileHeader) (string, error) {
	db := client.Database(boardID)
	file, err := imageFile.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the file into a byte slice
	imageData, err := io.ReadAll(file)
	if err != nil {
		logs.Error("Error reading image file: %v", err)
		return "", err
	}

	imageDoc := models.Image{
		Image:    imageData,
		Filetype: imageFile.Header.Get("Content-Type"),
		Size:     imageFile.Size,
		// Height and Width can be set if you decode the image
	}

	result, err := db.Collection("images").InsertOne(context.Background(), imageDoc)
	if err != nil {
		logs.Error("Error inserting image: %v", err)
		return "", err
	}

	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func ReturnThumb(c echo.Context, boardID string, thumbID string) error {
	db := client.Database(boardID)
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(thumbID)
	if err != nil {
		logs.Error("Invalid thumbnail ID '%s': %v", thumbID, err)
		return c.String(http.StatusBadRequest, "Invalid thumbnail ID")
	}

	var image models.Image
	err = db.Collection("thumbs").FindOne(ctx, bson.M{"_id": objectID}).Decode(&image)
	if err != nil {
		logs.Error("Error finding thumbnail: %v", err)
		return c.String(http.StatusNotFound, "Thumbnail not found")
	}

	c.Response().Header().Set("Content-Type", image.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(image.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Write(image.Image)

	return nil
}

func ReturnImage(c echo.Context, boardID string, imageID string) error {
	db := client.Database(boardID)
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(imageID)
	if err != nil {
		logs.Error("Invalid image ID '%s': %v", imageID, err)
		return c.String(http.StatusBadRequest, "Invalid image ID")
	}

	var image models.Image
	err = db.Collection("images").FindOne(ctx, bson.M{"_id": objectID}).Decode(&image)
	if err != nil {
		logs.Error("Error finding image: %v", err)
		return c.String(http.StatusNotFound, "Image not found")
	}

	c.Response().Header().Set("Content-Type", image.Filetype)
	c.Response().Header().Set("Content-Length", strconv.FormatInt(image.Size, 10))
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Write(image.Image)

	return nil
}
