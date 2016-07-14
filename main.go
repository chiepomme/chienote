package main

import (
	"github.com/chiepomme/chienote/convert"
	"github.com/chiepomme/chienote/initialize"
	"github.com/chiepomme/chienote/sync"

	"github.com/spf13/cobra"
)

func main() {

	var cmdInit = &cobra.Command{
		Use:   "init",
		Short: "Initialize chienote environment",
		Long:  `Initialize chienote environment`,
		Run: func(cmd *cobra.Command, args []string) {
			initialize.Initialize()
		},
	}

	var cmdSync = &cobra.Command{
		Use:   "sync",
		Short: "Sync local cache and evernote notes",
		Long:  `Sync local cache and evernote notes`,
		Run: func(cmd *cobra.Command, args []string) {
			sync.Sync()
		},
	}

	var cmdConvert = &cobra.Command{
		Use:   "convert",
		Short: "Convert local cache to static files",
		Long:  `Convert local cache to static files`,
		Run: func(cmd *cobra.Command, args []string) {
			convert.Convert()
		},
	}

	var rootCmd = &cobra.Command{Use: "chienote"}
	rootCmd.AddCommand(cmdInit, cmdSync, cmdConvert)
	rootCmd.Execute()
}
