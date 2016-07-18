// Package initialize is an initialization tool for chienote.
// It creates required directories and helps your setting process.
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

func initialize() error {
	fmt.Println("===== chienote initialization =====")
	panicIfConfigExists()

	var cfg config
	stdin := bufio.NewReader(os.Stdin)

	if err := askClientKey(&cfg, stdin); err != nil {
		return err
	}
	if err := askClientSecret(&cfg, stdin); err != nil {
		return err
	}
	if err := askDeveloperToken(&cfg, stdin); err != nil {
		return err
	}
	if err := askNotebookName(&cfg, stdin); err != nil {
		return err
	}
	if err := askEnvironment(&cfg, stdin); err != nil {
		return err
	}

	if err := saveConfiguration(&cfg); err != nil {
		return err
	}

	fmt.Println("Initialization succeeded.")
	fmt.Println("You can edit your configration file if you want: " + configFilePath)

	return nil
}

func panicIfConfigExists() error {
	if _, err := os.Stat(configFilePath); err == nil {
		return errors.Wrapf(err, "Can't create %v. Check if %v doesn't exists.", configFilePath, configFilePath)
	}
	return nil
}

func askClientKey(cfg *config, stdin *bufio.Reader) error {
	fmt.Print("Enter your client key:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.ClientKey = line
	} else {
		return errors.Errorf("Can't read your client key")
	}
	return nil
}

func askClientSecret(cfg *config, stdin *bufio.Reader) error {
	fmt.Print("Enter your client secret:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.ClientSecret = line
	} else {
		return errors.Errorf("Can't read your client secret")
	}
	return nil
}

func askDeveloperToken(cfg *config, stdin *bufio.Reader) error {
	fmt.Print("Enter your developer token:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.DeveloperToken = line
	} else {
		return errors.Errorf("Can't read your developer token")
	}
	return nil
}

func askNotebookName(cfg *config, stdin *bufio.Reader) error {
	fmt.Print("Enter your notebook name to sync:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.NotebookName = line
	} else {
		return errors.Errorf("Can't read your notebook name")
	}
	return nil
}

func askEnvironment(cfg *config, stdin *bufio.Reader) error {
	fmt.Print("Is it the sandbox?[y/n]:")
	if line, err := readLineTrimmed(stdin); err == nil && (line == "y" || line == "n") {
		cfg.Sandbox = line == "y"
	} else {
		return errors.Errorf("Can't read your environment information")
	}
	return nil
}

func readLineTrimmed(stdin *bufio.Reader) (line string, err error) {
	line, err = stdin.ReadString('\n')
	line = strings.Trim(line, "\n\r")
	return
}

func saveConfiguration(cfg *config) error {
	bytes, err := yaml.Marshal(*cfg)
	if err != nil {
		return errors.Errorf("Can't yamlize your configuration")
	}

	if err := ioutil.WriteFile(configFilePath, bytes, os.ModePerm); err != nil {
		return errors.Wrapf(err, "Can't save your configuration to %v", configFilePath)
	}
	return nil
}
