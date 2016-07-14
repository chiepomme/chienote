// Package initialize is an initialization tool for chienote.
// It creates required directories and helps your setting process.
package initialize

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/chiepomme/chienote/config"
)

// Initialize environment for chienote
func Initialize() {
	fmt.Println("===== chienote initialization =====")
	panicIfConfigExists()

	var cfg config.Config
	stdin := bufio.NewReader(os.Stdin)

	askClientKey(&cfg, stdin)
	askClientSecret(&cfg, stdin)
	askDeveloperToken(&cfg, stdin)
	askNotebookName(&cfg, stdin)
	askEnvironment(&cfg, stdin)

	saveConfiguration(&cfg)

	fmt.Println("Initialization succeeded.")
	fmt.Println("You can edit your configration file if you want: " + config.ConfigFilePath)
}

func panicIfConfigExists() {
	if _, err := os.Stat(config.ConfigFilePath); err == nil {
		errMsg := fmt.Sprintf("Can't create %v. Check if %v doesn't exists.", config.ConfigFilePath, config.ConfigFilePath)
		panic(errMsg)
	}
}

func askClientKey(cfg *config.Config, stdin *bufio.Reader) {
	fmt.Print("Enter your client key:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.ClientKey = line
	} else {
		panic("Can't read your client key")
	}
}

func askClientSecret(cfg *config.Config, stdin *bufio.Reader) {
	fmt.Print("Enter your client secret:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.ClientSecret = line
	} else {
		panic("Can't read your client secret")
	}
}

func askDeveloperToken(cfg *config.Config, stdin *bufio.Reader) {
	fmt.Print("Enter your developer token:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.DeveloperToken = line
	} else {
		panic("Can't read your developer token")
	}
}

func askNotebookName(cfg *config.Config, stdin *bufio.Reader) {
	fmt.Print("Enter your notebook name to sync:")
	if line, err := readLineTrimmed(stdin); err == nil && line != "" {
		cfg.NotebookName = line
	} else {
		panic("Can't read your notebook name")
	}
}

func askEnvironment(cfg *config.Config, stdin *bufio.Reader) {
	fmt.Print("Is it the sandbox?[y/n]:")
	if line, err := readLineTrimmed(stdin); err == nil && (line == "y" || line == "n") {
		cfg.Sandbox = line == "y"
	} else {
		panic("Can't read your environment information")
	}
}

func readLineTrimmed(stdin *bufio.Reader) (line string, err error) {
	line, err = stdin.ReadString('\n')
	line = strings.Trim(line, "\n\r")
	return
}

func saveConfiguration(cfg *config.Config) {
	bytes, err := json.MarshalIndent(*cfg, "", "    ")
	if err != nil {
		panic("Can't jsonize your configuration")
	}

	if err := ioutil.WriteFile(config.ConfigFilePath, bytes, os.ModePerm); err != nil {
		panic("Can't save your configuration to " + config.ConfigFilePath + " because of " + err.Error())
	}
}
