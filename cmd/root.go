package cmd

import (
	"fmt"
	"os"

	"npgo/internal/ui"
	"npgo/internal/updater"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "npgo",
	Short: "Fast and beautiful Go-based package manager",
	Long: `NPGO is a fast, beautiful package manager written in Go.
It provides lightning-fast package installation with a delightful CLI experience.

Features:
• ⚡ Ultra-fast package fetching
• 🎨 Beautiful CLI interface  
• 💾 Smart caching system
• 🔄 Parallel downloads
• 📦 npm-compatible commands`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		ui.Welcome()

		// Shorthand: npgo <script> == npgo run <script>
		if len(args) > 0 {
			known := map[string]struct{}{
				"fetch": {}, "install": {}, "i": {}, "run": {}, "update": {}, "help": {}, "--help": {}, "-h": {},
			}
			if _, ok := known[args[0]]; !ok {
				// treat as script name
				runScript(args[0])
				return
			}
		}

		fmt.Println("Available commands:")
		fmt.Println("  npgo fetch <package>@<version>  - Fetch a package")
		fmt.Println("  npgo install <package>         - Install a package")
		fmt.Println("  npgo run <script>              - Run a package.json script (or 'npgo <script>')")
		fmt.Println("  npgo update                    - Update npgo to latest version")
		fmt.Println("  npgo --help                     - Show help")
		fmt.Println()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags here if needed
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "quiet output")

	// Best-effort update check notice (non-blocking)
	go func() {
		latest, hasNew, err := updater.CheckUpdate(currentVersion)
		if err == nil && hasNew {
			ui.Warning.Println("⚠️  A new version is available:", latest, "→ run 'npgo update'")
		}
	}()
}
