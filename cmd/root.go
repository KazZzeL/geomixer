package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "geomixer",
	Short: "command-line tool to mix geofiles",
	Long:  "command-line tool to mix geofiles",
}

func Execute() {
	rootCmd.AddCommand(mixCmd)
	rootCmd.AddCommand(schemaCmd)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
		os.Exit(-1)
	}
}
