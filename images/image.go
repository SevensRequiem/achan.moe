package images

import (
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func GenerateThumbnail(inputPath string, outputPath string, maxWidth, maxHeight int) error {
	log.Printf("Generating thumbnail for %s", inputPath)

	// Open the input image file
	file, err := os.Open(inputPath)
	if err != nil {
		log.Printf("Error opening input file: %v", err)
		return err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Error decoding image: %v", err)
		return err
	}

	// Calculate the new dimensions while preserving the aspect ratio
	originalWidth := img.Bounds().Dx()
	originalHeight := img.Bounds().Dy()

	var newWidth, newHeight int
	if originalWidth > originalHeight {
		newWidth = maxWidth
		newHeight = (originalHeight * maxWidth) / originalWidth
	} else {
		newHeight = maxHeight
		newWidth = (originalWidth * maxHeight) / originalHeight
	}

	// Create a new image with the desired size
	thumbnail := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Resize the image
	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Create the output file
	out, err := os.Create(outputPath)
	if err != nil {
		log.Printf("Error creating output file: %v", err)
		return err
	}
	defer out.Close()

	// Encode the thumbnail as a PNG
	switch filepath.Ext(outputPath) {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(out, thumbnail, nil)
	case ".png":
		err = png.Encode(out, thumbnail)
	default:
		log.Printf("Unsupported output file format: %s", filepath.Ext(outputPath))
		return err
	}

	if err != nil {
		log.Printf("Error encoding thumbnail: %v", err)
		return err
	}

	log.Printf("Thumbnail generated successfully for %s", inputPath)
	return nil
}

func CompressImage(inputPath string, outputPath string, quality int) error {
	// Open the input image file
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Create the output file
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Encode the image as JPEG with the specified quality
	options := jpeg.Options{Quality: quality}
	err = jpeg.Encode(out, img, &options)
	if err != nil {
		return err
	}

	return nil
}
