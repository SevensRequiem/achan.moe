package images

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/image/draw"
)

// GenerateThumbnail creates a thumbnail for the given input image or video.
func GenerateThumbnail(inputPath, outputPath string, maxWidth, maxHeight int) error {
	// Get file extension
	ext := filepath.Ext(inputPath)
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".tiff", ".bmp", ".webp", ".avif":
		return imageThumbnail(inputPath, outputPath, maxWidth, maxHeight)
	case ".webm", ".mp4", ".mov":
		return videoThumbnail(inputPath, outputPath, maxWidth, maxHeight)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}

func imageThumbnail(inputPath, outputPath string, maxWidth, maxHeight int) error {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open the input file: %w", err)
	}
	defer inputFile.Close()

	inputImage, _, err := image.Decode(inputFile)
	if err != nil {
		return fmt.Errorf("failed to decode the input image: %w", err)
	}

	bounds := inputImage.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > maxWidth || height > maxHeight {
		width, height = resize(width, height, maxWidth, maxHeight)
	}

	thumbnail := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), inputImage, bounds, draw.Over, nil)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create the output file: %w", err)
	}
	defer outputFile.Close()

	if err := jpeg.Encode(outputFile, thumbnail, nil); err != nil {
		return fmt.Errorf("failed to encode the thumbnail: %w", err)
	}

	return nil
}

func resize(width, height, maxWidth, maxHeight int) (int, int) {
	if maxWidth <= 0 || maxHeight <= 0 {
		return width, height // No resizing if max dimensions are invalid
	}
	if width > height {
		height = height * maxWidth / width
		width = maxWidth
	} else {
		width = width * maxHeight / height
		height = maxHeight
	}
	return width, height
}

func videoThumbnail(inputPath, outputPath string, maxWidth, maxHeight int) error {
	ext := filepath.Ext(inputPath)
	if ext != ".webm" && ext != ".mp4" && ext != ".mov" {
		return fmt.Errorf("unsupported video format: %s", ext)
	}
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", fmt.Sprintf("thumbnail,scale=%d:%d", maxWidth, maxHeight), "-frames:v", "1", outputPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate the video thumbnail: %w", err)
	}
	return nil
}
