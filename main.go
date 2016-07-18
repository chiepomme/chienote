package main

import (
	"fmt"
	"os"

	"github.com/chiepomme/chienote/convert"
	"github.com/chiepomme/chienote/sync"
	"github.com/spf13/cobra"
)

var cfg *config

func main() {
	var cmdInit = &cobra.Command{
		Use:   "init",
		Short: "Initialize your configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			if err := initialize(); err != nil {
				fmt.Printf("%+v\n", err)
				os.Exit(-1)
			}
		},
	}

	var cmdSync = &cobra.Command{
		Use:   "sync",
		Short: "Sync local cache and evernote notes",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			if err := sync.Sync(cacheRoot, noteCacheDirName, resourceCacheDirName, cfg.ClientKey, cfg.ClientSecret, cfg.DeveloperToken, cfg.Sandbox, cfg.NotebookName); err != nil {
				fmt.Printf("%+v\n", err)
				os.Exit(-1)
			}
		},
	}

	var cmdConvert = &cobra.Command{
		Use:   "convert",
		Short: "Convert local cache to post files",
		Run: func(cmd *cobra.Command, args []string) {
			loadConfig()
			if err := convert.Convert(cacheRoot, noteCacheDirName, resourceCacheDirName, ".", postDirName, resourceDirName, true); err != nil {
				fmt.Printf("%+v\n", err)
				os.Exit(-1)
			}
		},
	}

	var rootCmd = &cobra.Command{Use: "chienote", Long: "Sync your evernote notebook to your jekyll directory. Execute chienote at your jekyll root."}
	rootCmd.AddCommand(cmdInit, cmdSync, cmdConvert)
	rootCmd.Execute()
}

func loadConfig() *config {
	cfg, err := getConfig()
	if err != nil {
		fmt.Println("configuration file load error")
		fmt.Println("it is recommended to execute init command")
		fmt.Printf("%+v", err)
		os.Exit(-1)
	}
	return cfg
}
