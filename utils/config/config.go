package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/labstack/echo/v4"
)

type GlobalConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Fork        string `json:"fork"`
	Minecraft   string `json:"minecraft"`
}

type BoardConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Locked      bool   `json:"locked"`
	AllowImages bool   `json:"allowimages"`
	ImageOnly   bool   `json:"imageonly"`
	AllowLinks  bool   `json:"allowlinks"`
	RateLimit   int    `json:"ratelimit"`
	MaxThreads  int    `json:"maxthreads"`
	MaxSize     int    `json:"maxsize"`
}

var globalConfig GlobalConfig

func init() {
	// Create directories if they don't exist
	if err := os.MkdirAll("config/boards", 0755); err != nil {
		log.Fatal(err)
	}

	// Load default global config into memory
	file, err := os.Open("config/global.json")
	if err != nil {
		log.Println("No global config found, creating default")
		file, err = os.Create("config/global.json")
		if err != nil {
			log.Fatal(err)
		}
		// Initialize with default values and write to the file
		defaultConfig := GlobalConfig{
			Name:        "achan",
			Version:     "1.0.0",
			Description: "a simple imageboard written in go",
			Fork:        "https://github.com/SevensRequiem/achan.moe",
			Minecraft:   "1234567890",
		}
		if err := WriteJSON(file, defaultConfig); err != nil {
			log.Fatal(err)
		}
		file.Close()
		// Reopen the file for reading
		file, err = os.Open("config/global.json")
		if err != nil {
			log.Fatal(err)
		}
	}
	defer file.Close()

	// Decode the JSON file into the GlobalConfig struct
	if err := ReadJSON(file, &globalConfig); err != nil {
		log.Fatal(err)
	}
}

func ReadJSON(file *os.File, config interface{}) error {
	decoder := json.NewDecoder(file)
	err := decoder.Decode(config)
	if err != nil {
		return err
	}
	return nil
}

func WriteJSON(file *os.File, config interface{}) error {
	encoder := json.NewEncoder(file)
	err := encoder.Encode(config)
	if err != nil {
		return err
	}
	return nil
}

func ReadGlobalConfig() GlobalConfig {
	file, err := os.Open("config/global.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var config GlobalConfig
	err = ReadJSON(file, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func WriteGlobalConfig(c echo.Context) error {
	// Parse multipart form
	err := c.Request().ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}

	// Update globalConfig with form values
	globalConfig.Name = c.FormValue("name")
	globalConfig.Description = c.FormValue("description")
	globalConfig.Fork = c.FormValue("fork")
	globalConfig.Minecraft = c.FormValue("minecraft")

	// Open the global config file for writing
	file, err := os.OpenFile("config/global.json", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open global config file for writing: %w", err)
	}
	defer file.Close()

	if err := WriteJSON(file, globalConfig); err != nil {
		return fmt.Errorf("failed to write JSON to config file: %w", err)
	}

	return nil
}

func ReadBoardConfig(id string) BoardConfig {
	file, err := os.Open("config/boards/" + id + ".json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var config BoardConfig
	err = ReadJSON(file, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}
