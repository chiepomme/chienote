package main

import (
	"github.com/chiepomme/chienote/convert"
	"github.com/chiepomme/chienote/sync"

	"github.com/spf13/cobra"
)

func main() {

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
	rootCmd.AddCommand(cmdSync, cmdConvert)
	rootCmd.Execute()
}
