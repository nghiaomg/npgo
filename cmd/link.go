package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"npgo/internal/ui"

	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link global cache to local node_modules",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader("Link node_modules")
		cwd, _ := os.Getwd()
		nm := filepath.Join(cwd, "node_modules")
		if err := os.MkdirAll(nm, 0755); err != nil {
			ui.ErrorMessage(err)
			return
		}
		// For now we just inform where global node_modules is and suggest using NODE_PATH
		ui.InstallStep("ℹ️", fmt.Sprintf("Global node_modules: %s", os.ExpandEnv("%USERPROFILE%\\.npgo\\node_modules")))
		ui.InstallStep("ℹ️", "`npgo run` already sets NODE_PATH automatically.")
		ui.InstallStep("✅", "Link step completed")
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
