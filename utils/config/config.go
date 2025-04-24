package config

import (
	"context"
	"sync"

	"achan.moe/database"
	"achan.moe/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	globalConfig *GlobalConfig
	configMutex  sync.RWMutex // To ensure thread-safe access to the globalConfig
)

type GlobalConfig struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Description   string `json:"description"`
	Fork          string `json:"fork"`
	MinecraftIP   string `json:"minecraft-ip"`
	MinecraftPort int    `json:"minecraft-port"`
}

func init() {
	// Ensure the database connection is established
	if database.DB_Main == nil {
		logs.Error("Database connection is not initialized")
		return
	}

	// Create an index on the "name" field
	_, err := database.DB_Main.Collection("config").Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{"name": 1},
	})
	if err != nil {
		logs.Error("Failed to create index on config collection: %v", err)
	}

	// Check if the config exists in the database
	if err := database.DB_Main.Collection("config").FindOne(context.Background(), bson.M{}).Err(); err != nil {
		// Insert a default configuration if none exists
		config := GlobalConfig{
			Name:          "achan",
			Version:       "0.0.4",
			Description:   "a simple imageboard written in Go",
			Fork:          "achan",
			MinecraftIP:   "69.164.202.38",
			MinecraftPort: 25565,
		}
		_, err := database.DB_Main.Collection("config").InsertOne(context.Background(), config)
		if err != nil {
			logs.Error("Failed to insert initial config: %v", err)
			return
		}
		logs.Info("Inserted initial config: %+v", config)
	}

	// Load the configuration into memory
	config, err := GetConfig()
	if err != nil {
		logs.Error("Failed to get config: %v", err)
		return
	}
	setGlobalConfig(config)
	logs.Info("Config loaded into memory: %+v", config)
}

func GetConfig() (*GlobalConfig, error) {
	var config GlobalConfig
	err := database.DB_Main.Collection("config").FindOne(context.Background(), bson.M{}).Decode(&config)
	if err != nil {
		logs.Error("Failed to get config from database: %v", err)
		return nil, err
	}
	logs.Debug("Config retrieved from database: %+v", config)
	return &config, nil
}

func SetConfig(config *GlobalConfig) error {
	_, err := database.DB_Main.Collection("config").UpdateOne(context.Background(), bson.M{}, bson.M{"$set": config})
	if err != nil {
		logs.Error("Failed to set config: %v", err)
		return err
	}
	setGlobalConfig(config) // Update the in-memory config
	return nil
}

func UpdateConfig(key string, value interface{}) error {
	_, err := database.DB_Main.Collection("config").UpdateOne(context.Background(), bson.M{}, bson.M{"$set": bson.M{key: value}})
	if err != nil {
		logs.Error("Failed to update config: %v", err)
		return err
	}

	configMutex.Lock()
	defer configMutex.Unlock()
	switch key {
	case "name":
		globalConfig.Name = value.(string)
	case "version":
		globalConfig.Version = value.(string)
	case "description":
		globalConfig.Description = value.(string)
	case "fork":
		globalConfig.Fork = value.(string)
	case "minecraft-ip":
		globalConfig.MinecraftIP = value.(string)
	case "minecraft-port":
		globalConfig.MinecraftPort = value.(int)
	}
	return nil
}

func GetConfigValue(key string) (interface{}, error) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	switch key {
	case "name":
		return globalConfig.Name, nil
	case "version":
		return globalConfig.Version, nil
	case "description":
		return globalConfig.Description, nil
	case "fork":
		return globalConfig.Fork, nil
	case "minecraft-ip":
		return globalConfig.MinecraftIP, nil
	case "minecraft-port":
		return globalConfig.MinecraftPort, nil
	default:
		logs.Error("Invalid config key: %s", key)
		return nil, nil
	}
}
func setGlobalConfig(config *GlobalConfig) {
	if config == nil {
		logs.Error("Attempted to set a nil globalConfig")
		return
	}
	configMutex.Lock()
	defer configMutex.Unlock()
	globalConfig = config
	logs.Debug("Global config set: %+v", globalConfig)
}
func GetGlobalConfig() *GlobalConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	logs.Debug("Global config accessed: %+v", globalConfig)
	if globalConfig == nil {
		logs.Error("Global config is nil")
		return nil
	}
	return globalConfig
}
