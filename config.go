package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

const cacheRoot = "_cache/"
const noteCacheDirName = "notes/"
const resourceCacheDirName = "resources/"
const postDirName = "_posts/"
const resourceDirName = "resources/"

const configFilePath = "_evernote.yml"

type config struct {
	ClientKey      string `yaml:"client_key"`
	ClientSecret   string `yaml:"client_secret"`
	DeveloperToken string `yaml:"developer_token"`
	Sandbox        bool   `yaml:"is_sandbox"`
	NotebookName   string `yaml:"notebook_name"`
}

func getConfig() (*config, error) {
	var cfg *config

	configBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "can't read %v", configFilePath)
	}
	if err := yaml.Unmarshal(configBytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "can't unmarshal %v", configFilePath)
	}
	if cfg.ClientKey == "" {
		return nil, errors.Errorf("client key is blank %v", configFilePath)
	}
	if cfg.ClientSecret == "" {
		return nil, errors.Errorf("client secret is blank %v", configFilePath)
	}
	if cfg.DeveloperToken == "" {
		return nil, errors.Errorf("developer token is blank %v", configFilePath)
	}
	if cfg.NotebookName == "" {
		return nil, errors.Errorf("notebook name is blank %v", configFilePath)
	}

	return cfg, nil
}
