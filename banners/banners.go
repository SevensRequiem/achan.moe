package banners

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"
)

func GetRandomBanner(boardid string) string {
	int := rand.Intn(2)
	if int == 0 {
		return GetRandomGlobalBanner()
	} else {
		return GetRandomLocalBanner(boardid)
	}
}

func GetRandomGlobalBanner() string {
	dir := "banners/global/"
	// Get random banner from global banners folder
	banner, err := GetRandomFile(dir)
	if err != nil {
		log.Printf("Error getting random file: %v", err)
		return ""
	}
	return banner
}

func GetRandomLocalBanner(boardid string) string {
	boardDir := "boards/" + boardid + "/"
	bannerDir := boardDir + "banners/"
	// Get random banner from banners folder
	banner, err := GetRandomFile(bannerDir)
	if err != nil {
		log.Printf("Error getting random file: %v", err)
		return ""
	}
	return banner
}

func GetRandomFile(dir string) (string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no files in directory: %s", dir)
	}
	rand.Seed(time.Now().UnixNano())
	randNum := rand.Intn(len(files))
	return files[randNum].Name(), nil
}
