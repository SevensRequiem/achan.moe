package banners

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"time"

	"achan.moe/logs"
)

// GetRandomBanner returns a random banner based on the board ID.
func GetRandomBanner(boardid string) (string, error) {
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(2) == 0 {
		logs.Debug("Getting global banner")
		return GetRandomGlobalBanner()
	}
	logs.Debug("Getting local banner for board %s", boardid)
	return GetRandomLocalBanner(boardid)
}

// GetRandomGlobalBanner returns a random global banner.
func GetRandomGlobalBanner() (string, error) {
	dir := "banners/global/"
	return GetRandomFile(dir)
}

// GetRandomLocalBanner returns a random local banner for a given board ID.
func GetRandomLocalBanner(boardid string) (string, error) {
	boardDir := filepath.Join("boards", boardid, "banners")
	return GetRandomFile(boardDir)
}

// GetRandomFile returns a random file from the specified directory.
func GetRandomFile(dir string) (string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logs.Error("failed to read directory %s: %v", dir, err)
		return "", err
	}
	if len(files) == 0 {
		logs.Error("no files found in %s", dir)
		return "", fmt.Errorf("no files found in %s", dir)
	}
	rand.Seed(time.Now().UnixNano())
	file := files[rand.Intn(len(files))]
	return filepath.Join(dir, file.Name()), nil
}
