package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/dreampuf/evernote-sdk-golang/client"
)

// CachePath is a directory path for caches
const CachePath = "cache/note/"

// NoteCachePath is a directory path for note cache
const NoteCachePath = "cache/note/"

// ResourceCachePath is a directory path for resource cache
const ResourceCachePath = "cache/resource/"

// PublicArticlePath is a directory path for public articles
const PublicArticlePath = "public/article/"

// PublicResourcePath is a directory path for public resources
const PublicResourcePath = "public/resource/"

// PublicPath is a public directory path
const PublicPath = "public/"

// NotesPerPage indicates how many articles on one page
const NotesPerPage = 5

// Config is a configuration for chienote
type Config struct {
	ClientKey      string `json:"client_key"`
	ClientSecret   string `json:"client_secret"`
	DeveloperToken string `json:"developer_token"`
	Sandbox        bool   `json:"is_sandbox"`
	NotebookName   string `json:"notebook_name"`
}

var config Config

// GetConfig loads configuration file "config.json"
func GetConfig() (*Config, error) {
	if config.ClientKey == "" {
		configBytes, err := ioutil.ReadFile("config.json")
		if err != nil {
			return nil, err
		}
		jsonErr := json.Unmarshal(configBytes, &config)
		if jsonErr != nil {
			return nil, jsonErr
		}
	}

	return &config, nil
}

// GetEnvironment returns selected environment type on your configuration (Sandbox or Production)
func (cfg *Config) GetEnvironment() client.EnvironmentType {
	if cfg.Sandbox {
		return client.SANDBOX
	}

	return client.PRODUCTION
}
