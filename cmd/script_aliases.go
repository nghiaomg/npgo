package cmd

import (
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Run 'start' script from package.json",
	Run:   func(cmd *cobra.Command, args []string) { runScript("start") },
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run 'dev' script from package.json",
	Run:   func(cmd *cobra.Command, args []string) { runScript("dev") },
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run 'build' script from package.json",
	Run:   func(cmd *cobra.Command, args []string) { runScript("build") },
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(buildCmd)
}
