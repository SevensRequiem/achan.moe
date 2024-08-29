package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// config is a package that handles config file reading and writing
// layout:
// config/global.json
// config/boards/{boardid}.json
// config/users/{userid}.json

// no struct

// GetConfig reads a config file and returns a map of the config
var (
	globalConfig map[string]interface{}
	boardConfigs map[string]map[string]interface{}
	userConfigs  map[string]map[string]interface{}
)

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
func ReadGlobalConfig() map[string]interface{} {
	config := make(map[string]interface{})
	file, err := os.Open("config/global.json")
	if err != nil {
		log.Printf("Error opening config file: %v", err)
		return config
	}
	defer file.Close()
	err = ReadJSON(file, &config)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
	}
	return make(map[string]interface{})
}
func ReadBoardConfig(boardid string) map[string]interface{} {
	config := make(map[string]interface{})
	file, err := os.Open("config/boards/" + boardid + ".json")
	if err != nil {
		log.Printf("Error opening config file: %v", err)
		return config
	}
	defer file.Close()
	err = ReadJSON(file, &config)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
	}
	return make(map[string]interface{})
}
func ReadUserConfig(userid string) map[string]interface{} {
	config := make(map[string]interface{})
	file, err := os.Open("config/users/" + userid + ".json")
	if err != nil {
		log.Printf("Error opening config file: %v", err)
		return config
	}
	defer file.Close()
	err = ReadJSON(file, &config)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
	}
	return make(map[string]interface{})
}

func WriteGlobalConfig(config map[string]interface{}) {
	file, err := os.Create("config/global.json")
	if err != nil {
		log.Printf("Error creating config file: %v", err)
		return
	}
	defer file.Close()
	err = WriteJSON(file, config)
	if err != nil {
		log.Printf("Error writing config file: %v", err)
	}
}

func WriteBoardConfig(boardid string, config map[string]interface{}) {
	file, err := os.Create("config/boards/" + boardid + ".json")
	if err != nil {
		log.Printf("Error creating config file: %v", err)
		return
	}
	defer file.Close()
	err = WriteJSON(file, config)
	if err != nil {
		log.Printf("Error writing config file: %v", err)
	}
}

func WriteUserConfig(userid string, config map[string]interface{}) {
	file, err := os.Create("config/users/" + userid + ".json")
	if err != nil {
		log.Printf("Error creating config file: %v", err)
		return
	}
	defer file.Close()
	err = WriteJSON(file, config)
	if err != nil {
		log.Printf("Error writing config file: %v", err)
	}
}

func LoadConfigsInMemory() {
	globalConfig = ReadGlobalConfig()
	boardConfigs = make(map[string]map[string]interface{})
	userConfigs = make(map[string]map[string]interface{})

	// Load board configurations
	boardFiles, err := ioutil.ReadDir("config/boards/")
	if err != nil {
		log.Printf("Error reading boards directory: %v", err)
		return
	}
	for _, file := range boardFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			boardID := strings.TrimSuffix(file.Name(), ".json")
			boardConfigs[boardID] = ReadBoardConfig(boardID)
		}
	}

	// Load user configurations
	userFiles, err := ioutil.ReadDir("config/users/")
	if err != nil {
		log.Printf("Error reading users directory: %v", err)
		return
	}
	for _, file := range userFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			userID := strings.TrimSuffix(file.Name(), ".json")
			userConfigs[userID] = ReadUserConfig(userID)
		}
	}
}
